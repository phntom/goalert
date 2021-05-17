package main

import (
	"encoding/json"
	"fmt"
	"github.com/mattermost/mattermost-server/v5/model"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

const (
	AlertsUrl = "https://alerts.ynet.co.il/alertsRss/YnetPicodeHaorefAlertFiles.js?callback=jsonCallback"
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

type Alerts struct {
	Alerts Items `json:"alerts"`
}

type Items struct {
	Items []AlertItem `json:"items"`
}

type AlertItem struct {
	Item AlertItemConcrete `json:"item"`
}

type AlertItemConcrete struct {
	Guid        string `json:"guid"`
	Time        string `json:"pubdate"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

func LoopOnAlerts() {
	announced := make(map[string]bool)
	for {
		time.Sleep(250 * time.Millisecond)
		res, err := http.Get(AlertsUrl)
		if err != nil {
			log.Fatal(err)
			continue
		}
		if res.StatusCode != 200 {
			log.Fatal(res.StatusCode)
			os.Exit(3)
		}
		content, err := ioutil.ReadAll(res.Body)
		message := GenerateMessageFromAlert(content, announced)
		if len(message) == 0 {
			continue
		}
		SendMsgToChannel(message, "")
	}
	//alertContent := `jsonCallback({"alerts": {"items": [{"item": {"guid": "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e","pubdate": "20:53","title": "שדה ניצן","description": "היכנסו למרחב המוגן","link": ""}},{"item": {"guid": "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e","pubdate": "20:53","title": "תלמי אליהו","description": "היכנסו למרחב המוגן","link": ""}},{"item": {"guid": "8a299260-c12c-4e2e-adc7-671b325474a3","pubdate": "20:53","title": "צוחר ואוהד","description": "היכנסו למרחב המוגן","link": ""}},{"item": {"guid": "56d79011-549d-4862-85f5-58a2240c12a7","pubdate": "20:53","title": "מבטחים עמיעוז ישע","description": "היכנסו למרחב המוגן","link": ""}}]}});`
}

func GenerateMessageFromAlert(alertContent []byte, announced map[string]bool) string {
	txtJson := alertContent[13 : len(alertContent)-2]
	var alerts Alerts
	json.Unmarshal([]byte(txtJson), &alerts)
	var hashtag strings.Builder
	var mentions strings.Builder
	var verbal strings.Builder
	var description string

	if len(alerts.Alerts.Items) == 0 {
		for k := range announced {
			delete(announced, k)
		}
		return ""
	}

	items := 0

	for _, item := range alerts.Alerts.Items {
		if _, ok := announced[item.Item.Guid]; ok {
			continue
		}

		if items != 0 {
			hashtag.WriteString(" ")
			mentions.WriteString(" ")
			verbal.WriteString(", ")
		}

		items += 1

		city := strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(
						item.Item.Title, " - ", " ",
					), ",", "",
				), " ", "_",
			), "-", "_",
		)
		cityShort := strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(
					item.Item.Title, " ", "",
				), ",", "",
			), "-", "",
		)
		hashtag.WriteString("#")
		hashtag.WriteString(city)
		mentions.WriteString("צבעאדום")
		mentions.WriteString(cityShort)
		verbal.WriteString(item.Item.Title)

		description = item.Item.Description
	}

	for _, item := range alerts.Alerts.Items {
		announced[item.Item.Guid] = true
	}

	if items == 0 {
		return ""
	}

	verbal.WriteString(": ")
	verbal.WriteString(description)

	return fmt.Sprintf(
		"%s\n%s\n%s",
		verbal.String(),
		hashtag.String(),
		mentions.String(),
	)
}
