package main

import (
	"context"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
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
}

func SetupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
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

func SendMsgToChannel(msg string, replyToId string, urgent bool, ack bool) {
	var previousPostId string

	mlog.Info("Alert", mlog.Any("msg", msg), mlog.Any("urgent", urgent), mlog.Any("ack", ack))

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
				Message:   msg,
				RootId:    replyToId,
				Metadata: &model.PostMetadata{
					Priority: &model.PostPriority{
						Priority:                model.NewString(urgentPriority(urgent)),
						RequestedAck:            model.NewBool(ack),
						PersistentNotifications: model.NewBool(false),
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
	announced := make(map[string]bool)
	failed := 0
	for {
		time.Sleep(250 * time.Millisecond)
		content := GetYnetAlertContent()
		if content == nil {
			failed += 1
			if failed > 30 {
				os.Exit(4)
			}
			continue
		}
		message, responseTime := GenerateMessageFromAlert(content, announced)
		if len(message) == 0 {
			continue
		}
		urgent := !strings.Contains(message, "נעלו")
		ack := responseTime == 90
		SendMsgToChannel(message, "", urgent, ack)
	}
}

//func TestAlert() {
//	announced := make(map[string]bool)
//	content := []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"22222222-1111-1111-1111-111111111111\",\"pubdate\": \"11:12\",\"title\": \"באר שבע - מערב\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"11111111-1111-1111-1111-111111111111\",\"pubdate\": \"11:11\",\"title\": \"תל אביב - מרכז העיר\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});")
//	message, responseTime := GenerateMessageFromAlert(content, announced)
//	urgent := !strings.Contains(message, "נעלו")
//	ack := responseTime == 90
//	SendMsgToChannel(message, "", urgent, ack)
//}
