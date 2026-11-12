package sources

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
	"github.com/phntom/goalert/internal/fetcher"
)

type YnetMessage struct {
	Alerts YnetMessageItems `json:"alerts"`
}

type YnetMessageItems struct {
	Items []YnetMessageItem `json:"items"`
}

type YnetMessageItem struct {
	Item YnetMessageItemConcrete `json:"item"`
}

type YnetMessageItemConcrete struct {
	Guid        string `json:"guid"`
	Time        string `json:"pubdate"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

type SourceYnet struct {
	client *http.Client
	URL    string
	seen   map[string]bool
	Bot    *bot.Bot
}

func (s *SourceYnet) Register() {
	s.client = fetcher.CreateHTTPClient()
	s.seen = make(map[string]bool)
}

func (s *SourceYnet) Fetch() []byte {
	return fetcher.FetchSource(s.client, s.URL, "ynet", YnetReferrer, &s.Bot.Monitoring)
}

func (s *SourceYnet) Parse(content []byte) []*bot.Message {
	districts := district.GetDistricts()
	lenAlertContent := len(content)
	if lenAlertContent < 40 {
		if lenAlertContent > 3 && lenAlertContent != 39 {
			slog.Warn("invalid content fetched - wrong length",
				"length", lenAlertContent,
				"content", string(content),
				"source", "ynet",
			)
		}
		// empty - i.e. no messages
		for s2 := range s.seen {
			delete(s.seen, s2)
		}
		return nil
	}
	dedup := make(map[string]*bot.Message)
	var dedupOrder []string
	txtJson := content[13 : len(content)-2]
	var alerts YnetMessage
	err := json.Unmarshal(txtJson, &alerts)
	if err != nil {
		slog.Error("failed to unmarshal",
			"error", err,
			"content", string(content),
			"source", "ynet",
		)
		return nil
	}
	for _, item := range alerts.Alerts.Items {
		if s.seen[item.Item.Guid] {
			continue
		}
		districtID := district.GetDistrictByCity(item.Item.Title)
		if districtID == "" {
			slog.Warn("district not found",
				"title", item.Item.Title,
				"source", "ynet",
			)
			continue
		}
		category := ""
		instructions := "instructions"
		if strings.Contains(item.Item.Description, "אלא אם ניתנה התרעה נוספת") {
			category = "uav"
			instructions = "uav_instructions"
		} else if item.Item.Description == "היכנסו למרחב המוגן ושהו בו 10 דקות" {
			category = "rockets"
		} else if strings.Contains(item.Item.Description, "נעלו") {
			instructions = "lockdown"
			category = "infiltration"
		} else if strings.Contains(item.Item.Description, "תרגיל") {
			slog.Warn("drill",
				"item", item,
				"source", "ynet",
			)
			s.seen[item.Item.Guid] = true
			continue
		}
		cityObj := districts["he"][districtID]
		msg := bot.NewMessage(instructions, category, cityObj.SafetyBufferSeconds, item.Item.Time)
		msg.RocketIDs[item.Item.Guid] = true
		hash := msg.GetHash()
		if _, ok := dedup[hash]; !ok {
			dedup[hash] = &msg
			dedupOrder = append(dedupOrder, hash)
		}
		dedup[hash].AppendDistrict(districtID)
	}
	var result []*bot.Message
	for _, hash := range dedupOrder {
		message := dedup[hash]
		result = append(result, message)
		for rocketGuid := range message.RocketIDs {
			s.seen[rocketGuid] = true
		}
	}
	return result
}

func (s *SourceYnet) Run() {
	failed := 0
	nextQuarterSecond := time.Now()
	for {
		if nextQuarterSecond.After(time.Now()) {
			// Calculate the duration until the next quarter second
			durationUntilNextQuarter := time.Until(nextQuarterSecond)
			// Sleep until the next quarter second
			time.Sleep(durationUntilNextQuarter)
		}

		content := s.Fetch()
		if content == nil {
			failed += 1
			if failed > 30 {
				os.Exit(4)
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}
		failed = 0
		for _, m := range s.Parse(content) {
			slog.Info("ynet", "content", string(content))
			s.Bot.SubmitMessage(m)
		}

		// Calculate the next quarter-second boundary
		// Add 250ms (a quarter second), then truncate to the nearest quarter second
		nextQuarterSecond = time.Now().Add(250 * time.Millisecond).Truncate(250 * time.Millisecond)
	}
}
