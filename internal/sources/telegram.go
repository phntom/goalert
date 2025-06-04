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
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
	"log"
	"regexp"
	"strings"
	"time"
	"fmt" // Added for formatting
)

// Regular expression to match city names followed by duration in parentheses
var extractCityNamesRe = regexp.MustCompile(`\n(.*?) *\((\d+ שניות|מיידי)\)`)
var extractPubTimeRe = regexp.MustCompile(`\((\d{1,2}/\d{1,2}/\d{4})\) (\d{1,2}:\d{2})`)

// CreatePostTestHook is a hook for testing CreatePost calls.
// It should be nil in production.
var CreatePostTestHook func(post *model.Post) bool

type SourceTelegram struct {
	Bot *bot.Bot
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
		// os.Exit(7) // Consider removing os.Exit from library code.
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
			log.Printf("Recovered from panic: %v", r) // This log might be an issue if std log is problematic
		}
	}()
	if update == nil {
		log.Println("Update is nil")
		return nil
	}

	m, ok := update.Message.AsNotEmpty()
	if !ok {
		return nil
	}

	peer := m.GetPeerID()
	if peer == nil {
		log.Println("Peer is nil")
		return nil
	}

	channelId, ok := peer.(*tg.PeerChannel)
	if !ok {
		log.Println("Peer is not a channel")
		return nil
	}

	text := m.(*tg.Message).GetMessage()
	text = strings.Trim(text, " \n\t")
	if text == "" {
		return nil
	}

	if channelId.ChannelID == 1441886157 { // pikudhaoref_all
		now := time.Now()
		var err error
		if strings.Contains(text, "בדקות הקרובות צפויות להתקבל התרעות באזורך") {
			// Early alert detected, process with "rockets" category
			err = processMessage(text, district.GetDistricts(), now, s.Bot, "rockets")
		} else {
			err = processMessage(text, district.GetDistricts(), now, s.Bot, "")
		}
		if err != nil {
			return err
		}
	} else if channelId.ChannelID == 1155294424 { // idf_telegram
		important := false
		if strings.Contains(text, "התרע") || strings.Contains(text, "פיגוע") || strings.Contains(text, "יירט") || strings.Contains(text, "מדיניות") || strings.Contains(text, "הנחיות") {
			important = true
		}
		mlog.Info("IDF message", mlog.String("text", text), mlog.Bool("important", important))
		post := model.Post{
			Message: text,
		}
		if important {
			// post.Metadata.Priority.Priority = model.NewString("important")
			s.Bot.DirectMessage(&post, "he")
		}
	} else if channelId.ChannelID == 2335255539 {
		keywords := []string{"ירוט", "ירט", "אזעק", "תימן", "תימני", "יורט", "שיגור", "פיצוץ"}
		foundKeyword := false
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				foundKeyword = true
				break
			}
		}

		if foundKeyword {
			modifiedText := "חדשות ישראל בטלגרם: " + text
			// Attempt to find the corresponding Mattermost channel by a conventional name.
			// Assumes Mattermost channel is named e.g., "telegram-2335255539"
			// Note: s.Bot.channels might not be populated if FindBotChannel hasn't run or is run by a different instance.
			// This part of the logic relies on s.Bot.channels being available and correctly populated.
			targetChannelName := fmt.Sprintf("telegram-%d", channelId.ChannelID)
			var targetMattermostChannelID string

			if s.Bot != nil && s.Bot.Client != nil && s.Bot.Channels != nil {
				for _, ch := range s.Bot.Channels {
					if ch.Name == targetChannelName {
						targetMattermostChannelID = ch.Id
						break
					}
				}
			} else {
				mlog.Warn("Bot instance or channels list not initialized, cannot find target channel by name", mlog.String("targetChannelName", targetChannelName))
			}

			if targetMattermostChannelID != "" {
				post := model.Post{
					Message:   modifiedText,
					ChannelId: targetMattermostChannelID,
				}
				if CreatePostTestHook != nil && CreatePostTestHook(&post) {
					// Test hook handled the post, do nothing further for this branch.
				} else {
					if _, _, err := s.Bot.Client.CreatePost(context.Background(), &post); err != nil {
						mlog.Error("Failed to send message to specific Mattermost channel", mlog.String("channelId", targetMattermostChannelID), mlog.Err(err))
					}
				}
			} else {
				mlog.Warn("Could not find corresponding Mattermost channel for Telegram channel ID", mlog.Any("telegramChannelId", channelId.ChannelID), mlog.String("assumedName", targetChannelName))
				// Fallback or error: Potentially send to a default channel or log that the specific channel was not found.
				// For now, it just logs. If general broadcast is desired as fallback, s.Bot.DirectMessage could be called.
				// However, the requirement is "send back to the same channel".
			}
		}
	} else {
		mlog.Debug("Unknown channel id", mlog.Any("channelId", channelId.ChannelID))
	}
	return nil
}

func processMessage(text string, districts district.Districts, now time.Time, b *bot.Bot, overrideCategory string) error {
	dedup := make(map[string]*bot.Message)
	var dedupOrder []string
	cities := extractCityNames(text)
	pubDate := extractPubTime(text)

	isEarlyAlert := overrideCategory == "rockets"
	err := checkExpired(pubDate, text, now, isEarlyAlert)
	if err != nil {
		if strings.Contains(text, "האירוע הסתיים") {
			msg := bot.NewMessage("uav_event_over", "", 0, "")
			b.SubmitMessage(&msg)
		} else {
			return err
		}
	}

	mlog.Info("Channel message", mlog.String("text", text), mlog.Any("cities", cities))
	category := ""
	if overrideCategory != "" {
		category = overrideCategory
	} else {
		if strings.Contains(text, "ירי רקטות וטילים") {
			category = "rockets"
		} else if strings.Contains(text, "חדירת כלי טיס עוין") {
			category = "uav"
		} else if strings.Contains(text, "חדירת מחבלים") {
			category = "infiltration"
		}
	}
	instructions := "instructions"
	if category == "infiltration" || category == "radiological" || category == "biohazard" {
		instructions = "lockdown"
	} else if category == "uav" {
		instructions = "uav_instructions"
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

func checkExpired(pubDate string, text string, now time.Time, isEarlyAlert bool) error {
	location, _ := time.LoadLocation("Asia/Jerusalem")
	// currentDate := time.Now().In(location).Format("2006-01-02") // Not needed anymore as date is in pubDate
	parsedPubDate, err := time.ParseInLocation("02/01/2006 15:04", pubDate, location)
	if err != nil {
		mlog.Error("Error parsing pubDate", mlog.Any("Message", text), mlog.String("pubDate", pubDate), mlog.Err(err))
		return err
	}

	expirationDuration := 90 * time.Second
	if isEarlyAlert {
		expirationDuration = 300 * time.Second
	}

	if parsedPubDate.Add(expirationDuration).Before(now) {
		mlog.Error("Expired telegram message", mlog.Any("Message", text), mlog.Any("ParsedPubDate", parsedPubDate), mlog.Bool("isEarlyAlert", isEarlyAlert))
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
		//os.Exit(7) // Consider removing os.Exit
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
