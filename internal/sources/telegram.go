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
	if peer.(*tg.PeerChannel).ChannelID == 1441886157 {

		text := m.(*tg.Message).GetMessage()
		dedup := make(map[string]bot.Message)
		cities := extractCityNames(text)
		pubDate := extractPubTime(text)
		mlog.Info("Channel message", mlog.Any("message", m.(*tg.Message)), mlog.Any("cities", cities))
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
			msg := bot.Message{
				Instructions:  instructions,
				Category:      category,
				SafetySeconds: uint(cityObj.SafetyBufferSeconds),
				Expire:        time.Now().Add(time.Second * 90),
				Cities:        nil,
				RocketIDs:     nil,
				Changed:       true,
				PubDate:       pubDate,
			}
			hash := msg.GetHash()
			if _, ok := dedup[hash]; !ok {
				dedup[hash] = msg
			}
			msg = dedup[hash]
			msg.Cities = append(msg.Cities, districtID)
			dedup[hash] = msg
		}
		for _, message := range dedup {
			s.Bot.SubmitMessage(&message)
		}
		if len(dedup) > 0 {
			s.Bot.Monitoring.SuccessfulSourceFetches.WithLabelValues("telegram").Inc()
		} else {
			s.Bot.Monitoring.FailedSourceFetches.WithLabelValues("telegram").Inc()
		}
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
