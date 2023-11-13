package bot

import (
	"context"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"os"
	"os/signal"
)

type Bot struct {
	IsOnline        bool
	client          *model.Client4
	webSocketClient *model.WebSocketClient
	serverVersion   string
	username        string
	userId          string
	channels        []model.Channel
}

func (b *Bot) Register() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			mlog.Info("Exiting")
			b.Disconnect()
			os.Exit(0)
		}
	}()
}

func (b *Bot) Connect() {
	b.client = model.NewAPIv4Client(os.Getenv("CHAT_DOMAIN"))
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
	props, _, err := b.client.GetOldClientConfig(context.Background(), "")
	if err != nil {
		mlog.Error("There was a problem pinging the Mattermost server.  Are you sure it's running?", mlog.Err(err))
		os.Exit(1)
	}
	b.serverVersion = props["Version"]
	mlog.Info("Server detected and is running", mlog.Any("serverVersion", b.serverVersion))
}

func (b *Bot) LoginAsTheBotUser() {
	b.client.SetToken(os.Getenv("AUTH_TOKEN"))
	user, _, err := b.client.GetMe(context.Background(), "")
	if err != nil {
		mlog.Error("There was a problem logging into the Mattermost server", mlog.Err(err))
		os.Exit(1)
	}
	b.username = user.Username
	b.userId = user.Id
	mlog.Info("Logged in", mlog.Any("username", b.username), mlog.Any("userId", b.userId))
}

func (b *Bot) FindBotChannel() {
	teams, _, err := b.client.GetTeamsForUser(context.Background(), b.userId, "")
	if err != nil {
		mlog.Error("There was a problem fetching active teams for bot", mlog.Err(err))
		os.Exit(1)
	}
	for _, team := range teams {
		if team == nil {
			continue
		}
		channels, _, err := b.client.GetChannelsForTeamForUser(context.Background(), team.Id, b.userId, false, "")
		if err != nil {
			mlog.Error("There was a problem fetching channels for team", mlog.Any("teamId", team.Id), mlog.Err(err))
			os.Exit(1)
		}
		for _, channel := range channels {
			if channel == nil {
				continue
			}
			b.channels = append(b.channels, *channel)
		}
		if b.channels == nil || len(b.channels) == 0 {
			mlog.Fatal("Bot is not a member of any channel, invite him and try again")
			os.Exit(2)
		}
	}
}
