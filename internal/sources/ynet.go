package sources

import (
	"encoding/json"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/config"
	"github.com/phntom/goalert/internal/district"
	"github.com/phntom/goalert/internal/fetcher"
	"io"
	"net/http"
	"strings"
)

type SourceYnet struct {
	client *http.Client
	URL    string
	seen   map[string][]district.ID
}

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

func (s *SourceYnet) Register() {
	s.client = fetcher.CreateHTTPClient()
}

func (s *SourceYnet) Fetch() []byte {
	res, err := s.client.Get(s.URL)
	if err != nil {
		mlog.Error("failed to fetch source ynet - client error", mlog.Err(err))
		return nil
	}
	if res.StatusCode != 200 {
		mlog.Warn("failed to fetch source ynet - wrong status code",
			mlog.Any("status", res.StatusCode),
			mlog.Any("res", res),
		)
		return nil
	}
	content, err := io.ReadAll(res.Body)
	if err != nil {
		mlog.Error("failed to fetch source ynet - io error", mlog.Err(err))
		return nil
	}
	return content
}

func (s *SourceYnet) Parse(content []byte) map[string][]district.ID {
	result := make(map[string][]district.ID)
	lenAlertContent := len(content)
	if lenAlertContent < 40 {
		if lenAlertContent > 3 && lenAlertContent != 39 {
			mlog.Warn("invalid content fetched from ynet - wrong length",
				mlog.Any("length", lenAlertContent),
				mlog.Any("content", string(content)),
			)
		}
		// empty - i.e. no messages
		return result
	}
	txtJson := content[13 : len(content)-2]
	var alerts YnetMessage
	err := json.Unmarshal(txtJson, &alerts)
	if err != nil {
		mlog.Error("failed to unmarshal ynet", mlog.Err(err), mlog.Any("content", content))
		return nil
	}
	for _, item := range alerts.Alerts.Items {
		districtID := district.GetDistrictByCity(item.Item.Title)
		if districtID == "" {
			mlog.Warn("district not found", mlog.Any("title", item.Item.Title))
			continue
		}
		result[item.Item.Description] = append(result[item.Item.Description], districtID)
	}
	return result
}

func (s *SourceYnet) Added(parsed map[string][]district.ID) map[string][]district.ID {
	result := make(map[string][]district.ID)
	for instructions, cities := range parsed {
		if strings.Contains(instructions, config.GetText("ynet.drill", "he")) {
			continue
		}
		prev, ok := s.seen[instructions]
		prevCitySet := make(map[district.ID]bool)
		if ok {
			for _, id := range prev {
				prevCitySet[id] = true
			}
			for _, city := range cities {
				if !prevCitySet[city] {
					result[instructions] = append(result[instructions], city)
				}
			}
		} else {
			result[instructions] = cities
		}
	}
	s.seen = parsed
	return result
}
