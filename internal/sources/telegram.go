package sources

import (
	"context"
	"github.com/go-faster/errors"
	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	updhook "github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
	"os"
	"regexp"
	"strings"
	"time"
)

// Regular expression to match city names followed by duration in parentheses
var extractCityNamesRe = regexp.MustCompile(`\n(.*?) *\((\d+ שניות|מיידי)\)`)
var extractPubTimeRe = regexp.MustCompile(`\[[^\]]+\] (\d{1,2}:\d{2})`)

type SourceTelegram struct {
	Bot    *bot.Bot
	client *telegram.Client
	gaps   *updates.Manager
}

func (s *SourceTelegram) Register() {
	d := tg.NewUpdateDispatcher()
	gaps := updates.New(updates.Config{
		Handler: d,
	})
	d.OnNewChannelMessage(s.ParseMessage)

	client, err := telegram.ClientFromEnvironment(telegram.Options{
		UpdateHandler: gaps,
		Middlewares: []telegram.Middleware{
			updhook.UpdateHook(gaps.Handle),
		},
		SessionStorage: &StorageMattermost{
			client:        s.Bot.Client,
			configChannel: s.Bot.ConfigChannel,
		},
	})
	if err != nil {
		mlog.Error("telegram client error", mlog.Err(err))
		os.Exit(7)
	}
	s.client = client
	s.gaps = gaps
}

func (s *SourceTelegram) Fetch() []byte {
	return nil
}

func (s *SourceTelegram) Parse(content []byte) []bot.Message {
	return nil
}

func (s *SourceTelegram) ParseMessage(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
	districts := district.GetDistricts()
	m, ok := update.Message.AsNotEmpty()
	if !ok {
		return nil
	}
	peer := m.GetPeerID()
	channelId := peer.(*tg.PeerChannel).ChannelID
	if channelId == 1441886157 { // pikudhaoref_all
		text := m.(*tg.Message).GetMessage()
		now := time.Now()
		err := processMessage(text, districts, now, s.Bot)
		if err != nil {
			return err
		}
	} else if channelId == 1155294424 { // idf_telegram
		text := m.(*tg.Message).GetMessage()
		mlog.Info("IDF message", mlog.String("text", text))
	} else {
		mlog.Debug("Unknown channel id", mlog.Any("channelId", channelId))
	}
	return nil
}

func processMessage(text string, districts district.Districts, now time.Time, b *bot.Bot) error {
	dedup := make(map[string]*bot.Message)
	var dedupOrder []string
	cities := extractCityNames(text)
	pubDate := extractPubTime(text)

	err := checkExpired(pubDate, text, now)
	if err != nil {
		return err
	}

	mlog.Info("Channel message", mlog.String("text", text), mlog.Any("cities", cities))
	category := ""
	if strings.Contains(text, "ירי רקטות וטילים") {
		category = "rockets"
	} else if strings.Contains(text, "חדירת כלי טיס עוין") {
		category = "uav"
	} else if strings.Contains(text, "חדירת מחבלים") {
		category = "infiltration"
	}
	instructions := "instructions"
	if category == "infiltration" || category == "radiological" || category == "biohazard" {
		instructions = "lockdown"
	}
	for _, cityName := range cities {
		districtID := district.GetDistrictByCity(cityName)
		cityObj := districts["he"][districtID]
		msg := bot.NewMessage(instructions, category, cityObj.SafetyBufferSeconds, pubDate)
		hash := msg.GetHash()
		if _, ok := dedup[hash]; !ok {
			dedup[hash] = &msg
			dedupOrder = append(dedupOrder, hash)
		}
		msg.AppendDistrict(districtID)
	}
	for _, hash := range dedupOrder {
		b.SubmitMessage(dedup[hash])
	}
	if len(dedup) > 0 {
		b.Monitoring.SuccessfulSourceFetches.WithLabelValues("telegram").Inc()
	} else {
		b.Monitoring.FailedSourceFetches.WithLabelValues("telegram").Inc()
	}
	return nil
}

func checkExpired(pubDate string, text string, now time.Time) error {
	location, _ := time.LoadLocation("Asia/Jerusalem")
	currentDate := time.Now().In(location).Format("2006-01-02")
	parsedPubDate, err := time.ParseInLocation("2006-01-02 15:04", currentDate+" "+pubDate, location)
	if err != nil {
		mlog.Error("Error parsing pubDate", mlog.Any("Message", text), mlog.Err(err))
		return err
	}

	if parsedPubDate.Add(90 * time.Second).Before(now) {
		mlog.Error("Expired telegram message", mlog.Any("Message", text), mlog.Any("ParsedPubDate", parsedPubDate))
		return errors.New("expired telegram message " + text)
	}
	return nil
}

func (s *SourceTelegram) Run() {
	flow := auth.NewFlow(examples.Terminal{}, auth.SendCodeOptions{})

	err := s.client.Run(context.Background(), func(ctx context.Context) error {
		if err := s.client.Auth().IfNecessary(ctx, flow); err != nil {
			return errors.Wrap(err, "auth")
		}

		// Fetch user info.
		user, err := s.client.Self(ctx)
		if err != nil {
			return errors.Wrap(err, "call self")
		}

		return s.gaps.Run(ctx, s.client.API(), user.ID, updates.AuthOptions{
			OnStart: func(ctx context.Context) {
				mlog.Info("Telegram gaps message parser started")
			},
		})
	})
	if err != nil {
		mlog.Error("telegram run error", mlog.Err(err))
		//os.Exit(7)
	}
}

func extractCityNames(text string) []string {
	// Find all matches
	matches := extractCityNamesRe.FindAllStringSubmatch(text, -1)
	var cities []string
	for _, match := range matches {
		if len(match) > 1 {
			// Add the city name to the list
			cities = append(cities, match[1])
		}
	}
	return cities
}

func extractPubTime(text string) string {
	res := extractPubTimeRe.FindStringSubmatch(text)
	if len(res) == 2 {
		return res[1]
	}
	return ""
}
