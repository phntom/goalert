package main

import (
	"github.com/mattermost/mattermost-server/v5/model"
	"os"
	"os/signal"
	"time"
)

const (
	AlertsUrl = "https://source-alerts.ynet.co.il/alertsRss/YnetPicodeHaorefAlertFiles.js?callback=jsonCallback"
)

var /* const */ CitiesResponseTime = map[string]int{
	"אשדוד - יא,יב,טו,יז,מרינה":      45,
	"אשדוד - ח,ט,י,יג,יד,טז":         45,
	"אשדוד - ג,ו,ז":                  45,
	"אשדוד - אזור תעשייה צפוני ונמל": 45,
	"אשדוד - א,ב,ד,ה":                45,
	"קריית גת, כרמי גת":              45,
	"אופקים":                         45,
	"באר שבע - מערב":                 60,
	"באר שבע - מזרח":                 60,
	"באר שבע - צפון":                 60,
	"באר שבע - דרום":                 60,
	"לוד":                            90,
	"תל אביב - מרכז העיר":            90,
	"תל אביב - מזרח":                 90,
	"תל אביב - דרום העיר ויפו":       90,
	"תל אביב - עבר הירקון":           90,
	"רמת גן - מזרח":                  90,
	"רמת גן - מערב":                  90,
	"ראש העין":                       90,
	"נס ציונה":                       90,
	"ראשון לציון - מערב":             90,
	"ראשון לציון - מזרח":             90,
	"בת-ים":                          90,
	"חולון":                          90,
	"גבעתיים":                        90,
	"פתח תקווה":                      90,
}

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
	if props, resp := client.GetOldClientConfig(""); resp.Error != nil {
		println("There was a problem pinging the Mattermost server.  Are you sure it's running?")
		PrintError(resp.Error)
		os.Exit(1)
	} else {
		println("Server detected and is running version " + props["Version"])
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
	if user, resp := client.GetMe(""); resp.Error != nil {
		println("There was a problem logging into the Mattermost server.  Are you sure ran the setup steps from the README.md?")
		PrintError(resp.Error)
		os.Exit(1)
	} else {
		println("Logged in as " + user.Username)
	}

}

func FindBotTeam() {
	teamName := os.Getenv("TEAM_NAME")
	if team, resp := client.GetTeamByName(teamName, ""); resp.Error != nil {
		println("We failed to get the initial load")
		println("or we do not appear to be a member of the team '" + teamName + "'")
		PrintError(resp.Error)
		os.Exit(1)
	} else {
		botTeam = team
	}
}

func FindBotChannel() {
	channelName := os.Getenv("CHANNEL_NAME")
	if rchannel, resp := client.GetChannelByName(channelName, botTeam.Id, ""); resp.Error != nil {
		println("We're not in channel " + channelName)
		PrintError(resp.Error)
	} else {
		targetChannel = rchannel
		return
	}
}

func SendMsgToChannel(msg string, replyToId string) {
	post := &model.Post{
		ChannelId: targetChannel.Id,
		Message:   msg,
		RootId:    replyToId,
	}
	if _, resp := client.CreatePost(post); resp.Error != nil {
		println("We failed to send a message to the logging channel")
		PrintError(resp.Error)
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
		message := GenerateMessageFromAlert(content, announced)
		if len(message) == 0 {
			continue
		}
		SendMsgToChannel(message, "")
	}
}
