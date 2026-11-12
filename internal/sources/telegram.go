package sources

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-faster/errors"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	updhook "github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
)

// Regular expression to match city names followed by duration in parentheses:
// immediate (מיידי), N seconds (N שניות), a minute (דקה), or a minute and a half (דקה וחצי).
var extractCityNamesRe = regexp.MustCompile(`\n(.*?) *\((מיידי|\d+ שניות|דקה וחצי|דקה)\)`)
var extractPubTimeRe = regexp.MustCompile(`\((\d{1,2}/\d{1,2}/\d{4})\) (\d{1,2}:\d{2})`)

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
		slog.Error("telegram client error", "error", err)
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
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic", "panic", r)
		}
	}()

	if update == nil {
		slog.Warn("Update is nil")
		return nil
	}

	m, ok := update.Message.AsNotEmpty()
	if !ok {
		return nil
	}

	peer := m.GetPeerID()
	if peer == nil {
		slog.Warn("Peer is nil")
		return nil
	}

	channelId, ok := peer.(*tg.PeerChannel)
	if !ok {
		slog.Warn("Peer is not a channel")
		return nil
	}

	text := m.(*tg.Message).GetMessage()
	text = strings.Trim(text, " \n\t")
	if text == "" {
		return nil
	}

	switch channelId.ChannelID {
	case 1441886157: // pikudhaoref_all
		now := time.Now()
		var err error
		if strings.Contains(text, "בדקות הקרובות צפויות להתקבל התרעות באזורך") {
			// Early alert: an alert is expected in the area shortly; process
			// it as a rockets alert ahead of the actual alert message.
			err = processMessage(text, district.GetDistricts(), now, s.Bot, "rockets")
		} else {
			err = processMessage(text, district.GetDistricts(), now, s.Bot, "")
		}
		if err != nil {
			return err
		}
	case 1155294424: // idf_telegram
		important := false
		if strings.Contains(text, "התרע") || strings.Contains(text, "פיגוע") || strings.Contains(text, "יירט") || strings.Contains(text, "מדיניות") || strings.Contains(text, "הנחיות") {
			important = true
		}
		slog.Info("IDF message", "text", text, "important", important)
		post := model.Post{
			Message: text,
		}
		if important {
			s.Bot.DirectMessage(&post, "he")
		}
	case 2335255539: // israel_news
		keywords := []string{"ירוט", "ירט", "אזעק", "תימן", "תימני", "יורט", "שיגור", "פיצוץ"}
		if !slices.ContainsFunc(keywords, func(keyword string) bool { return strings.Contains(text, keyword) }) {
			return nil
		}
		targetChannelName := fmt.Sprintf("telegram-%d", channelId.ChannelID)
		if s.Bot == nil {
			slog.Warn("Bot is not initialized, cannot forward message", "targetChannelName", targetChannelName)
			return nil
		}
		var target *model.Channel
		for _, ch := range s.Bot.Channels {
			if ch.Name == targetChannelName {
				target = ch
				break
			}
		}
		if target == nil {
			slog.Warn("Could not find corresponding Mattermost channel for Telegram channel",
				"telegramChannelId", channelId.ChannelID,
				"expectedName", targetChannelName,
			)
			return nil
		}
		post := model.Post{
			Message:   "חדשות ישראל בטלגרם: " + text,
			ChannelId: target.Id,
		}
		if _, err := s.Bot.SubmitPostToChannel(&post, target); err != nil {
			slog.Warn("Failed to forward message to Mattermost channel", "targetChannelName", targetChannelName, "error", err)
		}
	default:
		slog.Debug("Unknown channel id", "channelId", channelId.ChannelID)
	}
	return nil
}

func processMessage(text string, districts district.Districts, now time.Time, b *bot.Bot, overrideCategory string) error {
	cities := extractCityNames(text)
	pubDate := extractPubTime(text)

	// An "event over" message announces the end of an alert (e.g. a hostile
	// aircraft incursion); it must be posted whether or not it is stale, so
	// check it before rejecting expired messages.
	if strings.Contains(text, "האירוע הסתיים") {
		msg := bot.NewMessage("uav_event_over", "", 0, "")
		b.SubmitMessage(&msg)
		return nil
	}

	isEarlyAlert := overrideCategory == "rockets"
	err := checkExpired(pubDate, text, now, isEarlyAlert)
	if err != nil {
		return err
	}

	slog.Info("Channel message", "text", text, "cities", cities)
	messages := buildAlertMessages(text, districts, overrideCategory)
	for _, message := range messages {
		b.SubmitMessage(message)
	}
	if len(messages) > 0 {
		b.Monitoring.SuccessfulSourceFetches.WithLabelValues("telegram").Inc()
	} else {
		b.Monitoring.FailedSourceFetches.WithLabelValues("telegram").Inc()
	}
	return nil
}

// buildAlertMessages groups the extracted cities of a Home Front Command
// message into alert messages, one per distinct instructions/category/
// safety-buffer/pub-date combination, preserving first-seen order. A
// non-empty overrideCategory replaces the automatic category detection
// (early alerts carry no category keywords yet).
func buildAlertMessages(text string, districts district.Districts, overrideCategory string) []*bot.Message {
	cities := extractCityNames(text)
	pubDate := extractPubTime(text)

	category := overrideCategory
	if category == "" {
		if strings.Contains(text, "ירי רקטות וטילים") {
			category = "rockets"
		} else if strings.Contains(text, "חדירת כלי טיס עוין") {
			category = "uav"
		} else if strings.Contains(text, "חדירת מחבלים") {
			category = "infiltration"
		}
	}
	instructions := instructionsFor(category)

	dedup := make(map[string]*bot.Message)
	var dedupOrder []string
	for _, cityName := range cities {
		districtID := district.GetDistrictByCity(cityName)
		if districtID == "" {
			slog.Warn("city not found in districts", "city", cityName)
			continue
		}
		cityObj := districts["he"][districtID]
		msg := bot.NewMessage(instructions, category, cityObj.SafetyBufferSeconds, pubDate)
		hash := msg.GetHash()
		if _, ok := dedup[hash]; !ok {
			dedup[hash] = &msg
			dedupOrder = append(dedupOrder, hash)
		}
		dedup[hash].AppendDistrict(districtID)
	}
	messages := make([]*bot.Message, 0, len(dedupOrder))
	for _, hash := range dedupOrder {
		messages = append(messages, dedup[hash])
	}
	return messages
}

func checkExpired(pubDate string, text string, now time.Time, isEarlyAlert bool) error {
	location, _ := time.LoadLocation("Asia/Jerusalem")
	parsedPubDate, err := time.ParseInLocation("02/01/2006 15:04", pubDate, location)
	if err != nil {
		slog.Error("Error parsing pubDate", "Message", text, "pubDate", pubDate, "error", err)
		return err
	}

	expirationDuration := 90 * time.Second
	if isEarlyAlert {
		// Early alerts announce an alert expected within the coming minutes
		// and stay valid longer than regular alerts.
		expirationDuration = 300 * time.Second
	}

	if parsedPubDate.Add(expirationDuration).Before(now) {
		slog.Error("Expired telegram message", "Message", text, "ParsedPubDate", parsedPubDate, "isEarlyAlert", isEarlyAlert)
		return errors.New("expired telegram message " + text)
	}
	return nil
}

func (s *SourceTelegram) Run() {
	flow := auth.NewFlow(terminalAuth{}, auth.SendCodeOptions{})

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
				slog.Info("Telegram gaps message parser started")
			},
		})
	})
	if err != nil {
		slog.Error("telegram run error", "error", err)
	}
}

func extractCityNames(text string) []string {
	// Find all matches
	matches := extractCityNamesRe.FindAllStringSubmatch(text, -1)
	var cities []string
	for _, match := range matches {
		if len(match) > 1 {
			// A single line may list several cities separated by commas.
			for name := range strings.SplitSeq(match[1], ",") {
				if name = strings.TrimSpace(name); name != "" {
					cities = append(cities, name)
				}
			}
		}
	}
	return cities
}

func extractPubTime(text string) string {
	res := extractPubTimeRe.FindStringSubmatch(text)
	if len(res) == 3 { // Expect 3 parts: full match, date (DD/MM/YYYY), and time (HH:MM)
		dateParts := strings.Split(res[1], "/")
		if len(dateParts) == 3 {
			day := dateParts[0]
			month := dateParts[1]
			year := dateParts[2]

			if len(day) == 1 {
				day = "0" + day
			}
			if len(month) == 1 {
				month = "0" + month
			}
			formattedDate := fmt.Sprintf("%s/%s/%s", day, month, year)
			return formattedDate + " " + res[2] // Combine formatted date and time
		}
	}
	return ""
}
