package sources

import (
	"encoding/json"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
	"github.com/phntom/goalert/internal/fetcher"
	"net/http"
	"os"
	"strings"
	"time"
)

var categories = map[string]string{
	"1":  "rockets",
	"3":  "earthquake",
	"4":  "radiological",
	"5":  "tsunami",
	"6":  "uav",
	"7":  "biohazard",
	"13": "infiltration",
}

type OrefMessage struct {
	ID           string   `json:"id"`
	CategoryInt  string   `json:"cat"`
	CategoryStr  string   `json:"title"`
	Cities       []string `json:"data"`
	Instructions string   `json:"desc"`
}

type SourceOref struct {
	client *http.Client
	seen   map[string]bool
	Bot    *bot.Bot
}

func (s *SourceOref) Register() {
	s.client = fetcher.CreateHTTPClient()
	s.seen = make(map[string]bool)
}

func (s *SourceOref) Fetch() []byte {
	return fetcher.FetchSource(s.client, OrefURL, "oref", OrefReferrer, &s.Bot.Monitoring)
}

func (s *SourceOref) Parse(content []byte) []bot.Message {
	districts := district.GetDistricts()
	start := strings.IndexByte(string(content), '{')
	if start == -1 {
		for s2, v2 := range s.seen {
			if v2 {
				delete(s.seen, s2)
			}
		}
		return nil
	}
	dedup := make(map[string]bot.Message)
	var alerts OrefMessage
	content = []byte(string(content)[start:])
	err := json.Unmarshal(content, &alerts)
	if err != nil {
		mlog.Error("failed to unmarshal",
			mlog.Err(err),
			mlog.Any("content", content),
			mlog.Any("source", "oref"),
		)
		return nil
	}
	if _, ok := s.seen[alerts.ID]; ok {
		return nil
	}
	s.seen[alerts.ID] = false
	for _, city := range alerts.Cities {
		if s.seen[city] {
			continue
		}
		category := categories[alerts.CategoryInt]
		s.seen[city] = true
		if category == "" {
			mlog.Warn("unknown oref category", mlog.Any("alerts", alerts))
			continue
		}
		districtID := district.GetDistrictByCity(city)
		if districtID == "" {
			mlog.Warn("district not found",
				mlog.Any("data", city),
				mlog.Any("source", "oref"),
			)
			continue
		}
		cityObj := districts["he"][districtID]
		instructions := "instructions"
		if category == "infiltration" || category == "radiological" || category == "biohazard" {
			instructions = "lockdown"
		}
		msg := bot.Message{
			Instructions:  instructions,
			Category:      category,
			SafetySeconds: uint(cityObj.SafetyBufferSeconds),
			Expire:        time.Now().Add(time.Second * 90),
			Cities:        nil,
			RocketIDs:     nil,
		}
		hash := msg.GetHash()
		if _, ok := dedup[hash]; !ok {
			dedup[hash] = msg
		}
		msg = dedup[hash]
		msg.Cities = append(msg.Cities, districtID)
		dedup[hash] = msg
	}
	var result []bot.Message
	for _, message := range dedup {
		result = append(result, message)
	}
	return result
}

func (s *SourceOref) Run() {
	failed := 0
	counter := 0
	for {
		// Get the current time
		now := time.Now()
		// Calculate the time until the next round second
		untilNextSecond := time.Until(time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second()+1, 0, now.Location()))
		// Add 200 milliseconds
		sleepDuration := untilNextSecond + 200*time.Millisecond
		// Sleep for the calculated duration
		time.Sleep(sleepDuration)

		content := s.Fetch()
		//content := []byte("{\"id\": \"133449412450000000\",\"cat\": \"1\",\"title\": \"ירי רקטות וטילים\",\"data\": [\"בית שקמה\"],\"desc\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\"}")
		//content := []byte("{\n\t\t \"id\": \"133451945860000000\",\n\t\t \"cat\": \"1\",\n\t\t \"title\": \"ירי רקטות וטילים\",\n\t\t \"data\": [\n\t\t   \"ערב אל עראמשה\"\n\t\t ],\n\t\t \"desc\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\"\n\t\t}")
		if content == nil {
			failed += 1
			if failed > 30 {
				os.Exit(4)
			}
			continue
		}
		counter = counter + 1
		for _, m := range s.Parse(content) {
			mlog.Debug("oref", mlog.Any("content", string(content)))
			s.Bot.SubmitMessage(&m)
		}
		if counter > 1000 {
			for s2, v2 := range s.seen {
				if v2 == false {
					delete(s.seen, s2)
				}
			}
			counter = 0
		}
	}
}
