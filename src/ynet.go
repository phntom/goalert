package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

//var /* const */ replaceCityRegex = regexp.MustCompile(`[^א-ת]`)

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
	if len(alertContent) < 15 {
		if len(alertContent) != 3 {
			log.Println("Unexpected content from ynet: ", alertContent)
		}
		return ""
	}
	txtJson := alertContent[13 : len(alertContent)-2]
	var alerts Alerts
	err := json.Unmarshal(txtJson, &alerts)
	if err != nil {
		log.Println("Error in Unmarshal content ", alertContent, " ", err)
		return ""
	}
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

		city := cityNameCleanFull(item)
		cityShort := cityNameCleanShort(item)

		if items != 0 {
			hashtag.WriteString(" ")
			mentions.WriteString(" ")
			verbal.WriteString(", ")
		}

		items += 1

		if strings.Contains(item.Item.Description, "אזעקה במסגרת תרגיל") {
			verbal.WriteString("~~")
			verbal.WriteString(item.Item.Title)
			verbal.WriteString("~~")
			if len(description) == 0 {
				description = item.Item.Description
			}
			continue
		}

		if val, ok := CitiesResponseTime[item.Item.Title]; ok {
			if val > responseTime {
				responseTime = val
			}
		}

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

func cityNameCleanFull(item AlertItem) string {
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
	return city
}

func cityNameCleanShort(item AlertItem) string {
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
	return cityShort
}
