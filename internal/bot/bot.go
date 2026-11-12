package bot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/phntom/goalert/internal/config"
	"github.com/phntom/goalert/internal/district"
	"github.com/phntom/goalert/internal/monitoring"
)

type Bot struct {
	IsOnline        bool
	Client          *model.Client4
	apiSubmitPost   func(context.Context, *model.Post) (*model.Post, *model.Response, error)
	apiPatchPost    func(context.Context, string, *model.PostPatch) (*model.Post, *model.Response, error)
	apiSaveReaction func(context.Context, *model.Reaction) (*model.Reaction, *model.Response, error)
	webSocketClient *model.WebSocketClient
	serverVersion   string
	username        string
	userId          string
	Channels        []*model.Channel
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
	// The Mattermost API seam dispatches through b.Client at call time.
	// Register runs before Connect assigns b.Client, so method values would
	// capture a nil client here.
	b.apiSubmitPost = func(ctx context.Context, post *model.Post) (*model.Post, *model.Response, error) {
		return b.Client.CreatePost(ctx, post)
	}
	b.apiPatchPost = func(ctx context.Context, postID string, patch *model.PostPatch) (*model.Post, *model.Response, error) {
		return b.Client.PatchPost(ctx, postID, patch)
	}
	b.apiSaveReaction = func(ctx context.Context, reaction *model.Reaction) (*model.Reaction, *model.Response, error) {
		return b.Client.SaveReaction(ctx, reaction)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range c {
			slog.Info("Exiting")
			b.Disconnect()
			os.Exit(0)
		}
	}()
	b.Monitoring.Setup(func() bool { return b.IsOnline })
}

func (b *Bot) Connect() {
	b.Client = model.NewAPIv4Client(os.Getenv("CHAT_DOMAIN"))
	b.MakeSureServerIsRunning()
	b.LoginAsTheBotUser()
	b.IsOnline = true
	slog.Info("Connected")
}

func (b *Bot) Disconnect() {
	if b.webSocketClient != nil {
		b.webSocketClient.Close()
	}
	b.IsOnline = false
	slog.Info("Disconnected")
}

func (b *Bot) MakeSureServerIsRunning() {
	_, resp, err := b.Client.GetPing(context.Background())
	if err != nil {
		slog.Error("There was a problem pinging the Mattermost server.  Are you sure it's running?", "error", err)
		os.Exit(1)
	}
	b.serverVersion = resp.ServerVersion
	slog.Info("Server detected and is running", "serverVersion", b.serverVersion)
}

func (b *Bot) LoginAsTheBotUser() {
	b.Client.SetToken(os.Getenv("AUTH_TOKEN"))
	user, _, err := b.Client.GetMe(context.Background(), "")
	if err != nil {
		slog.Error("There was a problem logging into the Mattermost server", "error", err)
		os.Exit(1)
	}
	b.username = user.Username
	b.userId = user.Id
	slog.Info("Logged in", "username", b.username, "userId", b.userId)
}

func (b *Bot) FindBotChannel() {
	configTeam := os.Getenv("CONFIG_TEAM")
	if configTeam == "" {
		configTeam = "phantom"
	}
	configChannel := os.Getenv("CONFIG_CHANNEL")
	if configChannel == "" {
		configChannel = "config"
	}
	teams, _, err := b.Client.GetTeamsForUser(context.Background(), b.userId, "")
	if err != nil {
		slog.Error("There was a problem fetching active teams for bot", "error", err)
		os.Exit(1)
	}
	for _, team := range teams {
		if team == nil {
			continue
		}
		channels, _, err := b.Client.GetChannelsForTeamForUser(context.Background(), team.Id, b.userId, false, "")
		if err != nil {
			slog.Error("There was a problem fetching channels for team", "teamId", team.Id, "error", err)
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
			if team.Name == configTeam && channel.Name == configChannel {
				b.ConfigChannel = channel
				continue
			}
			channel.AddProp("teamName", team.Name)
			b.Channels = append(b.Channels, channel)
		}
	}
	if b.ConfigChannel == nil {
		slog.Warn("config channel not found", "team", configTeam, "channel", configChannel)
	}
	if len(b.Channels) == 0 {
		slog.Error("Bot is not a member of any channel, invite him and try again")
		os.Exit(2)
	}
	var channelNames []string
	for _, channel := range b.Channels {
		channelNames = append(channelNames, fmt.Sprintf("%s/%s", channel.Props["teamName"], channel.Name))
	}
	slog.Info("Joined channels", "channels", channelNames)
}

func (b *Bot) SubmitMessage(m *Message) {
	b.alertFeed <- m
	go b.UpdateMonitor(m)
}

func (b *Bot) AwaitMessage() {
	for message := range b.alertFeed {
		// If there are more than 20 cities, split the message
		if len(message.Cities) > 20 {
			originalCities := message.Cities

			for i := 0; i < len(originalCities); i += 20 {
				end := min(i+20, len(originalCities))
				chunk := originalCities[i:end]

				newMessage := &Message{
					Instructions:   message.Instructions,
					Category:       message.Category,
					Cities:         make([]district.ID, len(chunk)),
					SafetySeconds:  message.SafetySeconds,
					RocketIDs:      make(map[string]bool),
					PubDate:        message.PubDate,
					Expire:         message.Expire,
					Rendered:       make(map[config.Language]*model.Post),
					PostMutex:      sync.Mutex{},
					PostIDs:        []string{},
					ChannelsPosted: []*model.Channel{},
					Changed:        true,
				}
				copy(newMessage.Cities, chunk)
				for _, cityID := range chunk {
					if relevant, ok := message.RocketIDs[string(cityID)]; ok {
						newMessage.RocketIDs[string(cityID)] = relevant
					}
				}

				newMessage.PostMutex.Lock()
				newMessage.Prerender()
				newMessage.PostMutex.Unlock()
				b.UpdateMonitor(newMessage)

				for _, channel := range b.Channels {
					post := newMessage.PostForChannel(channel)
					result, err := executeSubmitPost(b, post, newMessage, channel)
					if err != nil {
						slog.Error("Failed to submit post for chunked message", "error", err, "channel", channel.Id, "chunkCities", newMessage.Cities)
						// Continue to the next channel for this chunk
						continue
					}

					// Goroutine for reactions and patching, specific to this newMessage and result
					go func(msgToProcess *Message, postResult *model.Post, ch *model.Channel) {
						if msgToProcess.Category == "uav" || msgToProcess.Category == "infiltration" {
							emoji := msgToProcess.Category + "-alert"
							executeAddReaction(b, postResult, emoji)
						}

						time.Sleep(200 * time.Millisecond) // Small delay before patching
						// Prepare the post for patching using the channel the message was actually sent to.
						patchContent := msgToProcess.PostForChannel(ch)
						executePatchPost(b, patchContent, postResult.Id)
					}(newMessage, result, channel)
				}
			}
			// The original message is fully handled by the chunks above.
			continue
		}

		if len(message.Cities) == 0 {
			slog.Warn("no cities", "message", message)
			for _, channel := range b.Channels {
				post := message.PostForChannel(channel)
				_, err := executeSubmitPost(b, post, message, channel)
				if err != nil {
					// Log error and continue to next channel, rather than stopping AwaitMessage
					slog.Error("Failed to submit post for message with no cities", "error", err, "channel", channel.Id)
					continue
				}
			}
			continue
		}

		msgsToPatch, citiesNotFound := b.GetPrevMsgs(message)
		for _, prevMsg := range msgsToPatch {
			if !prevMsg.PatchData(message) {
				// no new data in patch message
				continue
			}
			prevMsg.PatchPosts(b)
		}

		if len(citiesNotFound) > 0 {
			for _, channel := range b.Channels {
				post := message.PostForChannel(channel)
				result, err := executeSubmitPost(b, post, message, channel)
				if err != nil {
					// Log error and continue to next channel
					slog.Error("Failed to submit post for message with cities not found", "error", err, "channel", channel.Id)
					continue
				}

				// Goroutine for reactions and patching for original message flow
				go func(msgToProcess *Message, postResult *model.Post, ch *model.Channel) {
					if msgToProcess.Category == "uav" || msgToProcess.Category == "infiltration" {
						emoji := msgToProcess.Category + "-alert"
						executeAddReaction(b, postResult, emoji)
					}

					time.Sleep(200 * time.Millisecond)
					// Prepare the post for patching using the channel the message was actually sent to.
					patchContent := msgToProcess.PostForChannel(ch)
					executePatchPost(b, patchContent, postResult.Id)
				}(message, result, channel)
			}
		}
	}
}

func (m *Message) PatchPosts(b *Bot) {
	m.PostMutex.Lock()
	postIDsCpy := slices.Clone(m.PostIDs)
	channelsPostsCpy := slices.Clone(m.ChannelsPosted)
	m.PostMutex.Unlock()
	for i, postID := range postIDsCpy {
		channel := channelsPostsCpy[i]
		post := m.PostForChannel(channel)
		executePatchPost(b, post, postID)
	}
}

func (b *Bot) Cleanup() {
	for {
		time.Sleep(1 * time.Second)
		b.dedupMutex.Lock()
		for id, message := range b.dedup {
			message.PostMutex.Lock()
			changed := message.Changed
			message.Changed = false
			message.PostMutex.Unlock()
			if changed {
				message.PatchPosts(b)
			}
			if message.IsExpired() {
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
	b.Monitoring.RegionsHistogram.Observe(float64(len(regions)))
}

// SubmitPostToChannel posts a pre-built post to one specific channel through
// the Mattermost API seam. The post is not tracked for later patching.
func (b *Bot) SubmitPostToChannel(post *model.Post, channel *model.Channel) (*model.Post, error) {
	return executeSubmitPost(b, post, nil, channel)
}

func (b *Bot) DirectMessage(post *model.Post, language config.Language) {
	if b.Channels == nil {
		slog.Error("No channels available for direct messaging")
		return
	}

	for _, channel := range b.Channels {
		if ChannelToLanguage(channel) == language {
			post.ChannelId = channel.Id
			_, err := executeSubmitPost(b, post, nil, channel)
			if err != nil {
				slog.Error("Failed to submit post", "error", err)
				continue
			}
		}
	}
}
