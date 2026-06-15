package source

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-faster/errors"
	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	updhook "github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/i18n"
	"github.com/phntom/goalert/internal/metrics"
)

// Telegram channel ids the bot listens to.
const (
	chanPikudHaorefAll = 1441886157 // official aggregated alerts (with launch origin)
	chanIDFTelegram    = 1155294424 // IDF spokesperson news (early missile warnings)
	chanIsraelNews     = 2335255539 // Israel news, keyword-filtered
)

const (
	telegramAlertTTL = 90 * time.Second
	telegramEarlyTTL = 300 * time.Second
	earlyAlertPhrase = "בדקות הקרובות צפויות להתקבל התרעות באזורך"
	eventOverPhrase  = "האירוע הסתיים"
)

var (
	telegramCityRe    = regexp.MustCompile(`\n(.*?) *\((\d+ שניות|מיידי)\)`)
	telegramPubTimeRe = regexp.MustCompile(`\((\d{1,2}/\d{1,2}/\d{4})\) (\d{1,2}:\d{2})`)
)

// Forwarder posts non-alert news text to Mattermost (implemented by
// publish.Mattermost).
type Forwarder interface {
	Broadcast(lang i18n.Language, text string) error
	PostToNamed(name, text string) error
}

// Telegram ingests the Pikud HaOref aggregated-alert channel (the only source
// carrying launch origin) and forwards IDF/Israel news channels.
type Telegram struct {
	client    *telegram.Client
	gaps      *updates.Manager
	forwarder Forwarder
	metrics   *metrics.Metrics
	out       chan<- alert.Alert
}

// NewTelegram builds the source. It reads APP_ID/APP_HASH from the environment
// and stores its session in the Mattermost "config" channel.
func NewTelegram(mmClient *model.Client4, configChannelID string, fwd Forwarder, m *metrics.Metrics) (*Telegram, error) {
	t := &Telegram{forwarder: fwd, metrics: m}
	d := tg.NewUpdateDispatcher()
	gaps := updates.New(updates.Config{Handler: d})
	d.OnNewChannelMessage(t.onMessage)
	client, err := telegram.ClientFromEnvironment(telegram.Options{
		UpdateHandler: gaps,
		Middlewares:   []telegram.Middleware{updhook.UpdateHook(gaps.Handle)},
		SessionStorage: &mattermostSession{
			client:    mmClient,
			channelID: configChannelID,
		},
	})
	if err != nil {
		return nil, err
	}
	t.client = client
	t.gaps = gaps
	return t, nil
}

// Name implements Source.
func (t *Telegram) Name() string { return "telegram" }

// Run implements Source.
func (t *Telegram) Run(ctx context.Context, out chan<- alert.Alert) {
	t.out = out
	flow := auth.NewFlow(examples.Terminal{}, auth.SendCodeOptions{})
	err := t.client.Run(ctx, func(ctx context.Context) error {
		if err := t.client.Auth().IfNecessary(ctx, flow); err != nil {
			return errors.Wrap(err, "auth")
		}
		user, err := t.client.Self(ctx)
		if err != nil {
			return errors.Wrap(err, "self")
		}
		return t.gaps.Run(ctx, t.client.API(), user.ID, updates.AuthOptions{
			OnStart: func(context.Context) { mlog.Info("telegram source started") },
		})
	})
	if err != nil && ctx.Err() == nil {
		mlog.Error("telegram run error", mlog.Err(err))
	}
}

func (t *Telegram) onMessage(ctx context.Context, _ tg.Entities, update *tg.UpdateNewChannelMessage) error {
	defer func() {
		if r := recover(); r != nil {
			mlog.Error("telegram handler panic", mlog.Any("recover", r))
		}
	}()
	if update == nil {
		return nil
	}
	m, ok := update.Message.AsNotEmpty()
	if !ok {
		return nil
	}
	channelID, ok := m.GetPeerID().(*tg.PeerChannel)
	if !ok {
		return nil
	}
	msg, ok := m.(*tg.Message)
	if !ok {
		return nil
	}
	text := strings.Trim(msg.GetMessage(), " \n\t")
	if text == "" {
		return nil
	}
	switch channelID.ChannelID {
	case chanPikudHaorefAll:
		t.handleAlert(ctx, text, time.Now())
	case chanIDFTelegram:
		t.handleIDFNews(text)
	case chanIsraelNews:
		t.handleIsraelNews(text)
	}
	return nil
}

func (t *Telegram) handleAlert(ctx context.Context, text string, now time.Time) {
	early := strings.Contains(text, earlyAlertPhrase)
	pubDate := extractTelegramPubTime(text)
	if pubDate != "" && telegramExpired(pubDate, now, early) {
		if !strings.Contains(text, eventOverPhrase) {
			mlog.Warn("expired telegram alert", mlog.String("text", text))
			return
		}
	}
	cities := extractTelegramCities(text)
	if len(cities) == 0 {
		return
	}
	cat, kind := telegramCategory(text, early)
	if strings.Contains(text, eventOverPhrase) {
		cat, kind = alert.CatEndAlert, alert.KindEnd
	}
	t.metrics.SourceFetchOK.WithLabelValues("telegram").Inc()
	t.emit(ctx, alert.Alert{
		Category: cat,
		Kind:     kind,
		Areas:    cities,
		At:       telegramTime(pubDate, now),
		Source:   "telegram",
		Origin:   alert.ExtractOrigin(text),
		IDs:      []string{pubDate},
	})
}

func (t *Telegram) handleIDFNews(text string) {
	important := strings.Contains(text, "התרע") || strings.Contains(text, "פיגוע") ||
		strings.Contains(text, "יירט") || strings.Contains(text, "מדיניות") ||
		strings.Contains(text, "הנחיות")
	if !important {
		return
	}
	mlog.Info("IDF news", mlog.String("text", text))
	if err := t.forwarder.Broadcast("he", text); err != nil {
		mlog.Error("idf broadcast failed", mlog.Err(err))
	}
}

func (t *Telegram) handleIsraelNews(text string) {
	keywords := []string{"ירוט", "ירט", "אזעק", "תימן", "תימני", "יורט", "שיגור", "פיצוץ"}
	found := false
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			found = true
			break
		}
	}
	if !found {
		return
	}
	if err := t.forwarder.PostToNamed("telegram-2335255539", "חדשות ישראל בטלגרם: "+text); err != nil {
		mlog.Warn("israel-news forward failed", mlog.Err(err))
	}
}

func (t *Telegram) emit(ctx context.Context, a alert.Alert) {
	select {
	case t.out <- a:
	case <-ctx.Done():
	}
}

func telegramCategory(text string, early bool) (alert.Category, alert.Kind) {
	switch {
	case early:
		return alert.CatPreAlert, alert.KindPreAlert
	case strings.Contains(text, "חדירת כלי טיס עוין"):
		return alert.CatUAV, alert.KindAlert
	case strings.Contains(text, "חדירת מחבלים"):
		return alert.CatTerror, alert.KindAlert
	default:
		// Any other aggregated alert with cities defaults to seek-shelter.
		return alert.CatMissile, alert.KindAlert
	}
}

func extractTelegramCities(text string) []string {
	matches := telegramCityRe.FindAllStringSubmatch(text, -1)
	cities := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			cities = append(cities, strings.TrimSpace(m[1]))
		}
	}
	return cities
}

// extractTelegramPubTime returns "DD/MM/YYYY HH:MM" (zero-padded) or "".
func extractTelegramPubTime(text string) string {
	res := telegramPubTimeRe.FindStringSubmatch(text)
	if len(res) != 3 {
		return ""
	}
	parts := strings.Split(res[1], "/")
	if len(parts) != 3 {
		return ""
	}
	day, month, year := parts[0], parts[1], parts[2]
	if len(day) == 1 {
		day = "0" + day
	}
	if len(month) == 1 {
		month = "0" + month
	}
	return day + "/" + month + "/" + year + " " + res[2]
}

func telegramTime(pubDate string, now time.Time) time.Time {
	loc, _ := time.LoadLocation("Asia/Jerusalem")
	if t, err := time.ParseInLocation("02/01/2006 15:04", pubDate, loc); err == nil {
		return t
	}
	return now
}

func telegramExpired(pubDate string, now time.Time, early bool) bool {
	loc, _ := time.LoadLocation("Asia/Jerusalem")
	t, err := time.ParseInLocation("02/01/2006 15:04", pubDate, loc)
	if err != nil {
		return false
	}
	ttl := telegramAlertTTL
	if early {
		ttl = telegramEarlyTTL
	}
	return t.Add(ttl).Before(now)
}
