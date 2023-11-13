package main

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/config"
	"github.com/phntom/goalert/internal/district"
	"github.com/phntom/goalert/internal/sources"
	"os"
	"os/signal"
	"strings"
	"time"
)

var client *model.Client4
var webSocketClient *model.WebSocketClient

var botTeam *model.Team
var targetChannel *model.Channel

func main() {
	SetupGracefulShutdown()
	client = model.NewAPIv4Client(os.Getenv("CHAT_DOMAIN"))
	MakeSureServerIsRunning()
	LoginAsTheBotUser()
	FindBotTeam()
	FindBotChannel()
	LoopOnAlerts()
	//TestAlert()
}

func SetupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if webSocketClient != nil {
				webSocketClient.Close()
			}
			os.Exit(0)
		}
	}()
}

func MakeSureServerIsRunning() {
	if props, _, err := client.GetOldClientConfig(context.Background(), ""); err != nil {
		mlog.Error("There was a problem pinging the Mattermost server.  Are you sure it's running?", mlog.Err(err))
		os.Exit(1)
	} else {
		mlog.Info("Server detected and is running", mlog.Any("version", props["Version"]))
	}
}

func LoginAsTheBotUser() {
	client.SetToken(os.Getenv("AUTH_TOKEN"))
	if user, _, err := client.GetMe(context.Background(), ""); err != nil {
		mlog.Error("There was a problem logging into the Mattermost server", mlog.Err(err))
		os.Exit(1)
	} else {
		mlog.Info("Logged in", mlog.Any("username", user.Username))
	}

}

func FindBotTeam() {
	teamName := os.Getenv("TEAM_NAME")
	if team, _, err := client.GetTeamByName(context.Background(), teamName, ""); err != nil {
		mlog.Error("We failed to get the initial load or not in team", mlog.Any("teamName", teamName), mlog.Err(err))
		os.Exit(1)
	} else {
		botTeam = team
	}
}

func FindBotChannel() {
	channelName := os.Getenv("CHANNEL_NAME")
	if rchannel, _, err := client.GetChannelByName(context.Background(), channelName, botTeam.Id, ""); err != nil {
		mlog.Error("We're not in channel", mlog.Any("channelName", channelName), mlog.Err(err))
		os.Exit(2)
	} else {
		targetChannel = rchannel
		return
	}
}

func CitiesToFields(cities map[string][]string) []*model.SlackAttachmentField {
	var fields []*model.SlackAttachmentField
	for n1, n2 := range cities {
		value := strings.Join(n2, "\n")
		if len(n2) == 1 && n1 == n2[0] {
			value = ""
			fields = append(fields, &model.SlackAttachmentField{
				Title: n1,
				Value: value,
				Short: true,
			})
		} else {
			fields = append([]*model.SlackAttachmentField{{
				Title: n1,
				Value: value,
				Short: true,
			}}, fields...)
		}

	}
	return fields
}

func SendMsgToChannel(instructions string, replyToId string, urgent bool, ack bool, cities map[string][]string, legacy string, text string) {
	var previousPostId string

	mlog.Info("Alert",
		mlog.Any("instructions", instructions),
		mlog.Any("urgent", urgent),
		mlog.Any("ack", ack),
		mlog.Any("cities", cities),
		mlog.Any("legacy", legacy),
		mlog.Any("text", text),
	)

	fields := CitiesToFields(cities)

	outerCtx, cancel := context.WithTimeout(context.Background(), maxTimeout)
	defer cancel()

	for i := 0; i < maxRetries; i++ {
		select {
		case <-outerCtx.Done():
			mlog.Error("Failed to send the message within the timeout", mlog.String("reason", outerCtx.Err().Error()))
			return
		default:
			ctx, cancel := context.WithTimeout(outerCtx, postTimeout)
			post, _, err := client.CreatePost(ctx, &model.Post{
				ChannelId: targetChannel.Id,
				Message:   text,
				RootId:    replyToId,
				Props: map[string]any{
					"attachments": []*model.SlackAttachment{
						{
							Text:     instructions,
							Fallback: legacy,
							Color:    "#CF1434",
							Fields:   fields,
						},
					},
				},
				Metadata: &model.PostMetadata{
					Priority: &model.PostPriority{
						Priority:     model.NewString(urgentPriority(urgent)),
						RequestedAck: model.NewBool(ack),
						// PersistentNotifications: model.NewBool(false),
					},
				},
			})
			cancel()

			if err == nil && post != nil {
				mlog.Debug("Create post", mlog.Any("postId", post.Id), mlog.Any("createAt", post.CreateAt), mlog.Any("message", post.Message))
				if previousPostId != "" {
					if _, err := client.DeletePost(ctx, previousPostId); err != nil {
						mlog.Error("Failed to delete previous post", mlog.String("postId", previousPostId), mlog.Err(err))
					}
				}
				return // Exit if the post was successful
			}

			if post != nil {
				previousPostId = post.Id
			}

			if err != nil {
				mlog.Error("Failed to send the message", mlog.Err(err))
			}
			time.Sleep(retryInterval)
		}
	}

	mlog.Fatal("Failed to send the message", mlog.Any("attempts", maxRetries))
}

func urgentPriority(urgent bool) string {
	if urgent {
		return "urgent"
	}
	return "important"
}

func LoopOnAlerts() {
	failed := 0
	ynet := sources.SourceYnet{
		URL: AlertsUrl,
	}
	ynet.Register()
	districts := district.GetDistricts()
	lang := config.Language("he")
	for {
		time.Sleep(250 * time.Millisecond)
		content := ynet.Fetch()
		if content == nil {
			failed += 1
			if failed > 30 {
				os.Exit(4)
			}
			continue
		}
		SendContent(ynet, content, lang, districts)
	}
}

func SendContent(ynet sources.SourceYnet, content []byte, lang config.Language, districts district.Districts) {
	for instructions, cityIDs := range ynet.Added(ynet.Parse(content)) {
		urgent := !strings.Contains(instructions, "נעלו")
		maxResponseTime := 0
		cities, hashtags, mentions, legacy := district.CitiesToHashtagsMentionsLegacy(cityIDs, lang)
		for _, cityID := range cityIDs {
			city := districts[lang][cityID]
			maxResponseTime = max(maxResponseTime, city.SafetyBufferSeconds)
		}
		ack := maxResponseTime == 90
		text := fmt.Sprintf("%s\n%s %s\n%s",
			strings.Join(legacy, ", "),
			strings.Join(hashtags, " "),
			instructions,
			strings.Join(mentions, " "),
		)
		legacyStr := fmt.Sprintf("%s %s %s",
			strings.Join(legacy, ", "),
			instructions,
			strings.Join(mentions, " "),
		)
		SendMsgToChannel(
			instructions,
			"",
			urgent,
			ack,
			cities,
			legacyStr,
			text,
		)
	}
}

func TestAlert() {
	content := []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"22222222-1111-1111-1111-111111111111\",\"pubdate\": \"11:12\",\"title\": \"תלמי אליהו\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"11111111-1111-1111-1111-111111111111\",\"pubdate\": \"11:11\",\"title\": \"תל אביב - מרכז העיר\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});")
	ynet := sources.SourceYnet{}
	ynet.Register()
	districts := district.GetDistricts()
	lang := config.Language("he")
	SendContent(ynet, content, lang, districts)
}
