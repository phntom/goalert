package bot

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/config"
	"github.com/phntom/goalert/internal/district"
	"github.com/phntom/goalert/internal/monitoring"
	"os"
	"os/signal"
	"slices"
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
			b.Channels = append(b.Channels, channel)
		}
	}
	if b.Channels == nil || len(b.Channels) == 0 {
		mlog.Fatal("Bot is not a member of any channel, invite him and try again")
		os.Exit(2)
	}
	var channelNames []string
	for _, channel := range b.Channels {
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
		// If there are more than 20 cities, split the message
		if len(message.Cities) > 20 {
			originalCities := message.Cities // Keep a copy of the original cities
			// Also copy RocketIDs associated with the original set of cities if necessary.
			// For simplicity, we assume RocketIDs apply to the whole message context for now.
			// If RocketIDs are per-city, this logic would need adjustment.

			for i := 0; i < len(originalCities); i += 20 {
				end := i + 20
				if end > len(originalCities) {
					end = len(originalCities)
				}
				chunk := originalCities[i:end]

				// Create a new message for the chunk
				newMessage := &Message{
					Instructions:   message.Instructions,
					Category:       message.Category,
					Cities:         make([]district.ID, len(chunk)),
					SafetySeconds:  message.SafetySeconds,
					RocketIDs:      make(map[string]bool), // New map for each chunk
					PubDate:        message.PubDate,
					Expire:         message.Expire, // Copy expiration time
					Rendered:       make(map[config.Language]*model.Post),
					PostMutex:      sync.Mutex{},
					PostIDs:        []string{},
					ChannelsPosted: []*model.Channel{},
					Changed:        true,
				}
				copy(newMessage.Cities, chunk)
				// If RocketIDs are relevant per city, filter them here.
				// For now, copying all RocketIDs if they are not city-specific.
				// This assumes that if a rocket ID was relevant for the original large message,
				// it's relevant for all its constituent smaller messages.
				// If RocketIDs are tied to specific cities, this needs more granular handling.
				for _, cityID := range chunk {
					if relevant, ok := message.RocketIDs[string(cityID)]; ok {
						newMessage.RocketIDs[string(cityID)] = relevant
					}
				}
				// If RocketIDs are not per city but global for the alert:
				// for k, v := range message.RocketIDs {
				// 	newMessage.RocketIDs[k] = v
				// }


				newMessage.Prerender() // Prerender for the new set of cities
				b.UpdateMonitor(newMessage) // Update monitor for this new (chunked) message

				for _, channel := range b.Channels {
					post := newMessage.PostForChannel(channel)
					result, err := executeSubmitPost(b, post, newMessage, channel)
					if err != nil {
						mlog.Error("Failed to submit post for chunked message", mlog.Err(err), mlog.Any("channel", channel.Id), mlog.Any("chunkCities", newMessage.Cities))
						continue // Continue to the next channel for this chunk
					}

					// Goroutine for reactions and patching, specific to this newMessage and result
					go func(msgToProcess *Message, postResult *model.Post, ch *model.Channel) {
						if msgToProcess.Category == "uav" || msgToProcess.Category == "infiltration" {
							emoji := msgToProcess.Category + "-alert"
							executeAddReaction(b, postResult, emoji)
						}
						// The original code had a check:
						// if message.Category != "" && message.Category != "rockets" { /* no operation */ }
						// This seems to be a placeholder or no-op, so it's omitted.

						time.Sleep(200 * time.Millisecond) // Small delay before patching
						// Prepare the post for patching using the channel the message was actually sent to.
						patchContent := msgToProcess.PostForChannel(ch)
						executePatchPost(b, patchContent, postResult.Id)
					}(newMessage, result, channel) // Pass current newMessage, result, and channel
				}
			}
			// After processing all chunks, continue to the next message in alertFeed
			// This prevents the original large message from being processed by the subsequent logic.
			continue
		}

		// Original message processing logic starts here for messages with <= 20 cities
		if len(message.Cities) == 0 {
			mlog.Warn("no cities", mlog.Any("message", message))
			for _, channel := range b.Channels {
				post := message.PostForChannel(channel)
				_, err := executeSubmitPost(b, post, message, channel)
				if err != nil {
					// Log error and continue to next channel, rather than stopping AwaitMessage
					mlog.Error("Failed to submit post for message with no cities", mlog.Err(err), mlog.Any("channel", channel.Id))
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
					mlog.Error("Failed to submit post for message with cities not found", mlog.Err(err), mlog.Any("channel", channel.Id))
					continue
				}

				// Goroutine for reactions and patching for original message flow
				go func(msgToProcess *Message, postResult *model.Post, ch *model.Channel) {
					if msgToProcess.Category == "uav" || msgToProcess.Category == "infiltration" {
						emoji := msgToProcess.Category + "-alert"
						executeAddReaction(b, postResult, emoji)
					}
					// if message.Category != "" && message.Category != "rockets" { /* no-op */ }

					time.Sleep(200 * time.Millisecond)
					// Prepare the post for patching using the channel the message was actually sent to.
					patchContent := msgToProcess.PostForChannel(ch)
					executePatchPost(b, patchContent, postResult.Id)
				}(message, result, channel) // Pass current message, result, and channel
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
			if message.Changed {
				message.Changed = false
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
	b.Monitoring.RegionsHistogram.Observe(float64(len(districts)))
}

func (b *Bot) GetMonitoring() monitoring.Monitoring {
	return b.Monitoring
}

func (b *Bot) DirectMessage(post *model.Post, language config.Language) {
	if b.Channels == nil {
		mlog.Error("No channels available for direct messaging")
		return
	}

	for _, channel := range b.Channels {
		if ChannelToLanguage(channel) == language {
			post.ChannelId = channel.Id
			_, err := executeSubmitPost(b, post, nil, channel)
			if err != nil {
				mlog.Error("Failed to submit post", mlog.Err(err))
				continue
			}
		}
	}
}
