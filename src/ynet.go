package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var /* const */ replaceCityRegex = regexp.MustCompile(`[^א-ת]`)

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

func GetYnetAlertContent() []byte {
	res, err := http.Get(AlertsUrl)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	if res.StatusCode != 200 {
		log.Fatal(res.StatusCode)
		return nil
	}
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return content
}

func GenerateMessageFromAlert(alertContent []byte, announced map[string]bool) string {
	txtJson := alertContent[13 : len(alertContent)-2]
	var alerts Alerts
	json.Unmarshal(txtJson, &alerts)
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
	responseTime := 0

	for _, item := range alerts.Alerts.Items {
		if _, ok := announced[item.Item.Guid]; ok {
			continue
		}

		if val, ok := CitiesResponseTime[item.Item.Title]; ok {
			if val > responseTime {
				responseTime = val
			}
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
						strings.ReplaceAll(
							strings.ReplaceAll(
								item.Item.Title, " - ", " ",
							), "'", "",
						), "\"", "",
					), ",", "",
				), " ", "_",
			), "-", "",
		)
		cityShort := strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(
						strings.ReplaceAll(
							item.Item.Title, " ", "",
						), "'", "",
					), "\"", "",
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

	hashtag.WriteString(" ")
	hashtag.WriteString(description)

	largeCityTitle := ""
	if responseTime >= 90 {
		largeCityTitle = "#"
	} else if responseTime >= 60 {
		largeCityTitle = "##"
	} else if responseTime >= 45 {
		largeCityTitle = "###"
	}

	return fmt.Sprintf(
		"%s %s\n%s\n%s",
		largeCityTitle,
		verbal.String(),
		hashtag.String(),
		mentions.String(),
	)
}

//alertContent := `jsonCallback({"alerts": {"items": [{"item": {"guid": "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e","pubdate": "20:53","title": "שדה ניצן","description": "היכנסו למרחב המוגן","link": ""}},{"item": {"guid": "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e","pubdate": "20:53","title": "תלמי אליהו","description": "היכנסו למרחב המוגן","link": ""}},{"item": {"guid": "8a299260-c12c-4e2e-adc7-671b325474a3","pubdate": "20:53","title": "צוחר ואוהד","description": "היכנסו למרחב המוגן","link": ""}},{"item": {"guid": "56d79011-549d-4862-85f5-58a2240c12a7","pubdate": "20:53","title": "מבטחים עמיעוז ישע","description": "היכנסו למרחב המוגן","link": ""}}]}});`
