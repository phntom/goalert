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

func PrintError(err *model.AppError) {
	println("\tError Details:")
	println("\t\t" + err.Message)
	println("\t\t" + err.Id)
	println("\t\t" + err.DetailedError)
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
	priorityStr := "important"
	if urgent {
		priorityStr = "urgent"
	}
	postPriority := model.PostPriority{
		Priority:                model.NewString(priorityStr),
		RequestedAck:            model.NewBool(ack),
		PersistentNotifications: model.NewBool(false),
	}
	metadata := model.PostMetadata{
		Priority: &postPriority,
	}
	post := &model.Post{
		ChannelId: targetChannel.Id,
		Message:   msg,
		RootId:    replyToId,
		Metadata:  &metadata,
	}

	// Create a context that will timeout after maxTimeout
	ctx, cancel := context.WithTimeout(context.Background(), maxTimeout)
	defer cancel()

	success := false
	for i := 0; i < maxRetries && !success; i++ {
		select {
		case <-ctx.Done():
			mlog.Error("Failed to send the message within the timeout", mlog.Any("post", post), mlog.String("reason", ctx.Err().Error()))
			return
		default:
			if _, _, err := client.CreatePost(ctx, post); err == nil {
				success = true
			} else {
				mlog.Error("Attempt to send a message failed", mlog.Any("post", post), mlog.Err(err))
				if i < maxRetries-1 {
					time.Sleep(retryInterval)
				}
			}
		}
	}

	if !success {
		mlog.Error("Failed to send a message to the logging channel after all retries", mlog.Any("post", post))
	}
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
