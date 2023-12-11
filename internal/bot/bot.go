package bot

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/district"
	"github.com/phntom/goalert/internal/monitoring"
	"os"
	"os/signal"
	"sync"
	"time"
)

type Bot struct {
	IsOnline        bool
	Client          *model.Client4
	webSocketClient *model.WebSocketClient
	serverVersion   string
	username        string
	userId          string
	channels        []*model.Channel
	ConfigChannel   *model.Channel
	alertFeed       chan *Message
	dedup           map[district.ID]*Message
	dedupMutex      sync.Mutex
	Monitoring      monitoring.Monitoring
}

const postTimeout = 10 * time.Second

func (b *Bot) Register() {
	b.alertFeed = make(chan *Message)
	b.dedup = make(map[district.ID]*Message)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			mlog.Info("Exiting")
			b.Disconnect()
			os.Exit(0)
		}
	}()
	b.Monitoring.Setup()
}

func (b *Bot) Connect() {
	b.Client = model.NewAPIv4Client(os.Getenv("CHAT_DOMAIN"))
	b.MakeSureServerIsRunning()
	b.LoginAsTheBotUser()
	b.IsOnline = true
	mlog.Info("Connected")
}

func (b *Bot) Disconnect() {
	if b.webSocketClient != nil {
		b.webSocketClient.Close()
	}
	b.IsOnline = false
	mlog.Info("Disconnected")
}

func (b *Bot) MakeSureServerIsRunning() {
	props, _, err := b.Client.GetOldClientConfig(context.Background(), "")
	if err != nil {
		mlog.Error("There was a problem pinging the Mattermost server.  Are you sure it's running?", mlog.Err(err))
		os.Exit(1)
	}
	b.serverVersion = props["Version"]
	mlog.Info("Server detected and is running", mlog.Any("serverVersion", b.serverVersion))
}

func (b *Bot) LoginAsTheBotUser() {
	b.Client.SetToken(os.Getenv("AUTH_TOKEN"))
	user, _, err := b.Client.GetMe(context.Background(), "")
	if err != nil {
		mlog.Error("There was a problem logging into the Mattermost server", mlog.Err(err))
		os.Exit(1)
	}
	b.username = user.Username
	b.userId = user.Id
	mlog.Info("Logged in", mlog.Any("username", b.username), mlog.Any("userId", b.userId))
}

func (b *Bot) FindBotChannel() {
	teams, _, err := b.Client.GetTeamsForUser(context.Background(), b.userId, "")
	if err != nil {
		mlog.Error("There was a problem fetching active teams for bot", mlog.Err(err))
		os.Exit(1)
	}
	for _, team := range teams {
		if team == nil {
			continue
		}
		channels, _, err := b.Client.GetChannelsForTeamForUser(context.Background(), team.Id, b.userId, false, "")
		if err != nil {
			mlog.Error("There was a problem fetching channels for team", mlog.Any("teamId", team.Id), mlog.Err(err))
			os.Exit(1)
		}
		for _, channel := range channels {
			if channel == nil {
				continue
			}
			if channel.IsGroupOrDirect() {
				continue
			}
			if channel.Name == "off-topic" || channel.Name == "town-square" {
				continue
			}
			if team.Name == "phantom" && channel.Name == "config" {
				b.ConfigChannel = channel
				continue
			}
			channel.AddProp("teamName", team.Name)
			b.channels = append(b.channels, channel)
		}
	}
	if b.channels == nil || len(b.channels) == 0 {
		mlog.Fatal("Bot is not a member of any channel, invite him and try again")
		os.Exit(2)
	}
	var channelNames []string
	for _, channel := range b.channels {
		channelNames = append(channelNames, fmt.Sprintf("%s/%s", channel.Props["teamName"], channel.Name))
	}
	mlog.Info("Joined channels", mlog.Any("channels", channelNames))
}

func (b *Bot) SubmitMessage(m *Message) {
	b.alertFeed <- m
	go b.UpdateMonitor(m)
}

func (b *Bot) AwaitMessage() {
	for message := range b.alertFeed {
		if len(message.Cities) == 0 {
			mlog.Warn("invalid message no cities", mlog.Any("message", message))
			continue
		}
		msgsToPatch, citiesNotFound := b.GetPrevMsgs(message)
		for _, prevMsg := range msgsToPatch {
			if !prevMsg.PatchData(message) {
				// no new data in patch message
				continue
			}
			for i, postID := range prevMsg.PostIDs {
				channel := prevMsg.ChannelsPosted[i]
				post := prevMsg.PostForChannel(channel)
				//goland:noinspection GoDeprecation
				patch := model.PostPatch{
					Message: model.NewString(""),
					Props:   &post.Props,
				}
				ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
				_, response, err := b.Client.PatchPost(ctx, postID, &patch)
				cancel()
				if err != nil {
					mlog.Error("failed patching post",
						mlog.Err(err),
						mlog.Any("postID", postID),
						mlog.Any("patch", patch),
						mlog.Any("response", response),
					)
					b.Monitoring.FailedPatches.Inc()
				} else {
					b.Monitoring.SuccessfulPatches.Inc()
				}
			}
		}
		if len(citiesNotFound) > 0 {
			for _, channel := range b.channels {
				post := message.PostForChannel(channel)
				ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
				result, response, err := b.Client.CreatePost(ctx, post)
				cancel()
				if err != nil {
					mlog.Error("failed creating post",
						mlog.Err(err),
						mlog.Any("post", post),
						mlog.Any("response", response),
					)
					return
				}
				message.PostIDs = append(message.PostIDs, result.Id)
				message.ChannelsPosted = append(message.ChannelsPosted, channel)
				b.Monitoring.SuccessfulPosts.Inc()
			}
		}
	}
}

func (b *Bot) Cleanup() {
	for {
		time.Sleep(1 * time.Second)
		b.dedupMutex.Lock()
		for id, message := range b.dedup {
			if message.IsExpired() {
				if message.Changed {
					message.PatchData(&Message{})
				}
				delete(b.dedup, id)
			}
		}
		b.dedupMutex.Unlock()
	}
}

func (b *Bot) UpdateMonitor(m *Message) {
	b.Monitoring.CitiesHistogram.Observe(float64(len(m.Cities)))
	b.Monitoring.DayOfWeekHistogram.Observe(float64(time.Now().Weekday()))
	b.Monitoring.TimeOfDayHistogram.Observe(float64(time.Now().Hour()))
	regions := make(map[string]bool)
	districts := district.GetDistricts()
	for _, city := range m.Cities {
		regions[districts["he"][city].AreaName] = true
	}
	b.Monitoring.RegionsHistogram.Observe(float64(len(districts)))
}
