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
	return fetcher.FetchSource(s.client, s.URL, "ynet", YnetReferrer)
}

func (s *SourceYnet) Parse(content []byte) []bot.Message {
	districts := district.GetDistricts()
	lenAlertContent := len(content)
	if lenAlertContent < 40 {
		if lenAlertContent > 3 && lenAlertContent != 39 {
			mlog.Warn("invalid content fetched - wrong length",
				mlog.Any("length", lenAlertContent),
				mlog.Any("content", string(content)),
				mlog.Any("source", "ynet"),
			)
		}
		// empty - i.e. no messages
		for s2 := range s.seen {
			delete(s.seen, s2)
		}
		return nil
	}
	dedup := make(map[string]*bot.Message)
	txtJson := content[13 : len(content)-2]
	var alerts YnetMessage
	err := json.Unmarshal(txtJson, &alerts)
	if err != nil {
		mlog.Error("failed to unmarshal",
			mlog.Err(err),
			mlog.Any("content", content),
			mlog.Any("source", "ynet"),
		)
		return nil
	}
	for _, item := range alerts.Alerts.Items {
		if s.seen[item.Item.Guid] {
			continue
		}
		districtID := district.GetDistrictByCity(item.Item.Title)
		if districtID == "" {
			mlog.Warn("district not found",
				mlog.Any("title", item.Item.Title),
				mlog.Any("source", "ynet"),
			)
			continue
		}
		instructions := "instructions"
		if strings.Contains(item.Item.Description, "נעלו") {
			instructions = "lockdown"
		} else if strings.Contains(item.Item.Description, "תרגיל") {
			mlog.Warn("drill",
				mlog.Any("item", item),
				mlog.Any("source", "ynet"),
			)
			s.seen[item.Item.Guid] = true
			continue
		}
		cityObj := districts["he"][districtID]
		msg := bot.Message{
			Instructions:  instructions,
			Category:      "",
			SafetySeconds: uint(cityObj.SafetyBufferSeconds),
			Expire:        time.Now().Add(time.Second * 10),
			Cities:        nil,
			RocketIDs:     map[string]bool{item.Item.Guid: true},
		}
		hash := msg.GetHash()
		if _, ok := dedup[hash]; !ok {
			dedup[hash] = &msg
		}
		dedup[hash].Cities = append(dedup[hash].Cities, districtID)
	}
	var result []bot.Message
	for _, message := range dedup {
		result = append(result, *message)
		for rocketGuid := range message.RocketIDs {
			s.seen[rocketGuid] = true
		}
	}
	return result
}

func (s *SourceYnet) Run() {
	failed := 0
	delay := 250 * time.Millisecond
	for {
		time.Sleep(delay)
		content := s.Fetch()
		//content := []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"f038657b-99e1-48ec-b5c2-3e49c409b3bb\",\"pubdate\": \"11:40\",\"title\": \"בית שקמה\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});")
		if content == nil {
			failed += 1
			if failed > 30 {
				os.Exit(4)
			}
			continue
		}
		for _, m := range s.Parse(content) {
			mlog.Debug("ynet", mlog.Any("content", string(content)))
			s.Bot.SubmitMessage(&m)
		}
	}
}
