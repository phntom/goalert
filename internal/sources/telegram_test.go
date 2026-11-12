package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
)

func Test_extractCityNames(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "single city, immediate",
			text: `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.`,
			want: []string{"מטולה"},
		},
		{
			name: "single city, minute duration",
			text: "אזור השפלה\nגאליה (דקה)",
			want: []string{"גאליה"},
		},
		{
			name: "single city, seconds duration",
			text: "אזור לכיש\nבית שקמה (90 שניות)",
			want: []string{"בית שקמה"},
		},
		{
			name: "multi-city line, minute and a half",
			text: "אזור שומרון\nדולב, טלמון, נריה (דקה וחצי)",
			want: []string{"דולב", "טלמון", "נריה"},
		},
		{
			name: "multi-city line, minute, double space before duration",
			text: "אזור לכיש\nבית אלעזרי, בית גמליאל, יבנה, כפר הנגיד, מעון צופיה, מפעל אגריגדה  (דקה)",
			want: []string{"בית אלעזרי", "בית גמליאל", "יבנה", "כפר הנגיד", "מעון צופיה", "מפעל אגריגדה"},
		},
		{
			name: "full message with region headers extracts only city lines",
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:22

אזור השפלה
גאליה (דקה)

אזור לכיש
בית אלעזרי, בית גמליאל, יבנה, כפר הנגיד, מעון צופיה, מפעל אגריגדה  (דקה)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			want: []string{"גאליה", "בית אלעזרי", "בית גמליאל", "יבנה", "כפר הנגיד", "מעון צופיה", "מפעל אגריגדה"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCityNames(tt.text)
			if !slices.Equal(got, tt.want) {
				t.Errorf("extractCityNames() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_processMessage(t *testing.T) {
	districts := district.GetDistricts()
	s := &SourceTelegram{
		Bot: &bot.Bot{},
	}
	s.Bot.Register()
	go s.Bot.AwaitMessage()

	tests := []struct {
		name             string
		now              time.Time
		text             string
		overrideCategory string
		wantErr          bool
	}{
		{
			name: "Early Alert - Valid (over 90s, under 300s)",
			// Message time: (10/10/2024) 12:00 IDT (09:00 UTC). "now" is 150
			// seconds after: expired for a regular alert, valid for an early
			// alert, which uses the 5-minute window.
			now:              time.Date(2024, 10, 10, 9, 2, 30, 0, time.UTC),
			overrideCategory: "rockets",
			text:             `(10/10/2024) 12:00 התרעה מוקדמת: בדקות הקרובות צפויות להתקבל התרעות באזורך.`,
			wantErr:          false,
		},
		{
			name: "Early Alert - Expired (just over 300s)",
			// Message time: (10/10/2024) 12:00 IDT (09:00 UTC). "now" is 301
			// seconds after: beyond even the early-alert window.
			now:              time.Date(2024, 10, 10, 9, 5, 1, 0, time.UTC),
			overrideCategory: "rockets",
			text:             `(10/10/2024) 12:00 התרעה מוקדמת: בדקות הקרובות צפויות להתקבל התרעות באזורך.`,
			wantErr:          true,
		},
		{
			name: "Regular Alert - Expired (299s diff, expired by the 90s rule)",
			// Message time: (29/05/2025) 21:23 IDT (18:23 UTC). "now" is 299
			// seconds after: would have been valid under an old blanket 300s
			// rule, but a regular alert expires after 90 seconds.
			now: time.Date(2025, 5, 29, 18, 27, 59, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:23

אזור שומרון
דולב, טלמונ, נריה (דקה וחצי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: true,
		},
		{
			name: "Original Valid message (updated format)",
			// Message time: (10/10/2024) 11:19 IDT. For UTC, assuming IDT is UTC+3, this is 08:19 UTC.
			// "now" is 30 seconds after message time.
			now: time.Date(2024, 10, 10, 8, 19, 30, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: false,
		},
		{
			name: "Original Expired message (updated format)",
			// Message time: (10/10/2024) 11:19 IDT (08:19 UTC)
			// "now" is 121 seconds after message time (08:21:01 UTC)
			now: time.Date(2024, 10, 10, 8, 21, 1, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: true,
		},
		{
			name: "New Message 1 - Non-expired",
			// Message time: (29/05/2025) 21:23 IDT. Assuming IDT is UTC+3, this is 18:23 UTC.
			// "now" is 30 seconds after message time.
			now: time.Date(2025, 5, 29, 18, 23, 30, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:23

אזור שומרון
דולב, טלמון, נריה (דקה וחצי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: false,
		},
		{
			name: "New Message 1 - Expired",
			// Message time: (29/05/2025) 21:23 IDT (18:23 UTC)
			// "now" is 120 seconds after message time (18:25:00 UTC)
			now: time.Date(2025, 5, 29, 18, 25, 0, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:23

אזור שומרון
דולב, טלמון, נריה (דקה וחצי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: true,
		},
		{
			name: "New Message 2 - Non-expired",
			// Message time: (29/05/2025) 21:22 IDT (18:22 UTC)
			// "now" is 30 seconds after message time.
			now: time.Date(2025, 5, 29, 18, 22, 30, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:22

אזור השפלה
גאליה (דקה)

אזור לכיש
בית אלעזרי, בית גמליאל, יבנה, כפר הנגיד, מעון צופיה, מפעל אגריגדה  (דקה)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: false,
		},
		{
			name: "New Message 2 - Expired",
			// Message time: (29/05/2025) 21:22 IDT (18:22 UTC)
			// "now" is 120 seconds after message time (18:24:00 UTC)
			now: time.Date(2025, 5, 29, 18, 24, 0, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:22

אזור השפלה
גאליה (דקה)

אזור לכיש
בית אלעזרי, בית גמליאל, יבנה, כפר הנגיד, מעון צופיה, מפעל אגריגדה  (דקה)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: true,
		},
		{
			name: "Event over - fresh message is posted",
			// Message time: (29/05/2025) 21:40 IDT (18:40 UTC). "now" is 10 seconds
			// after message time: the event-over announcement must be posted.
			now: time.Date(2025, 5, 29, 18, 40, 10, 0, time.UTC),
			text: `✅ האירוע הסתיים - חדירת כלי טיס עוין (29/5/2025) 21:40

אזור שומרון

התושבים מתבקשים לחזור לשגרה.`,
			wantErr: false,
		},
		{
			name: "Event over - stale message is still posted",
			// Message time: (29/05/2025) 21:40 IDT (18:40 UTC). "now" is 120 seconds
			// after message time: the event-over announcement is posted regardless.
			now: time.Date(2025, 5, 29, 18, 42, 0, 0, time.UTC),
			text: `✅ האירוע הסתיים - חדירת כלי טיס עוין (29/5/2025) 21:40

אזור שומרון

התושבים מתבקשים לחזור לשגרה.`,
			wantErr: false,
		},
		{
			name: "Unknown city name is skipped",
			// Message time: (29/05/2025) 21:23 IDT (18:23 UTC). The extracted city
			// name does not resolve to a district; it must be skipped, not fail.
			now: time.Date(2025, 5, 29, 18, 23, 30, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:23

אזור שומרון
עיר לא קיימת בכלל (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := processMessage(tt.text, districts, tt.now, s.Bot, tt.overrideCategory)
			if (err != nil) != tt.wantErr {
				t.Errorf("processMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_buildAlertMessages(t *testing.T) {
	districts := district.GetDistricts()

	// The "New Message 1" alert: all three settlements share one
	// (דקה וחצי) line and the same 90-second safety buffer.
	newMessage1 := `🚨 ירי רקטות וטילים (29/5/2025) 21:23

אזור שומרון
דולב, טלמון, נריה (דקה וחצי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`

	t.Run("same-duration multi-city line yields one message with all cities", func(t *testing.T) {
		messages := buildAlertMessages(newMessage1, districts, "")
		if len(messages) != 1 {
			t.Fatalf("got %d messages, want 1 (all cities share one safety buffer)", len(messages))
		}
		msg := messages[0]
		want := []district.ID{
			district.GetDistrictByCity("דולב"),
			district.GetDistrictByCity("טלמון"),
			district.GetDistrictByCity("נריה"),
		}
		if !slices.Equal(msg.Cities, want) {
			t.Errorf("Cities = %v, want %v", msg.Cities, want)
		}
		if msg.Category != "rockets" {
			t.Errorf("Category = %q, want %q", msg.Category, "rockets")
		}
		if msg.Instructions != "instructions" {
			t.Errorf("Instructions = %q, want %q", msg.Instructions, "instructions")
		}
		if msg.SafetySeconds != 90 {
			t.Errorf("SafetySeconds = %d, want 90", msg.SafetySeconds)
		}
	})

	t.Run("mixed durations yield one message per safety buffer", func(t *testing.T) {
		text := `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

אזור שומרון
דולב (דקה וחצי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.`
		messages := buildAlertMessages(text, districts, "")
		if len(messages) != 2 {
			t.Fatalf("got %d messages, want 2 (one per distinct safety buffer)", len(messages))
		}
		// First-seen order: מטולה (immediate, 0-second buffer) before דולב (90).
		if !slices.Equal(messages[0].Cities, []district.ID{district.GetDistrictByCity("מטולה")}) {
			t.Errorf("first message Cities = %v, want [%v]", messages[0].Cities, district.GetDistrictByCity("מטולה"))
		}
		if !slices.Equal(messages[1].Cities, []district.ID{district.GetDistrictByCity("דולב")}) {
			t.Errorf("second message Cities = %v, want [%v]", messages[1].Cities, district.GetDistrictByCity("דולב"))
		}
	})

	t.Run("unknown city is skipped", func(t *testing.T) {
		text := `🚨 ירי רקטות וטילים (29/5/2025) 21:23

אזור שומרון
עיר לא קיימת בכלל (מיידי)

אזור השפלה
גאליה (דקה)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.`
		messages := buildAlertMessages(text, districts, "")
		if len(messages) != 1 {
			t.Fatalf("got %d messages, want 1 (only the resolvable city)", len(messages))
		}
		if !slices.Equal(messages[0].Cities, []district.ID{district.GetDistrictByCity("גאליה")}) {
			t.Errorf("Cities = %v, want [%v]", messages[0].Cities, district.GetDistrictByCity("גאליה"))
		}
	})

	t.Run("early-alert override forces the rockets category", func(t *testing.T) {
		// An early alert carries no rocket-fire keywords; without the
		// override the category would stay empty.
		text := `אזור קו העימות
מטולה (מיידי)`
		messages := buildAlertMessages(text, districts, "rockets")
		if len(messages) != 1 {
			t.Fatalf("got %d messages, want 1", len(messages))
		}
		if messages[0].Category != "rockets" {
			t.Errorf("Category = %q, want %q", messages[0].Category, "rockets")
		}
		if !slices.Equal(messages[0].Cities, []district.ID{district.GetDistrictByCity("מטולה")}) {
			t.Errorf("Cities = %v, want [מטולה]", messages[0].Cities)
		}
	})
}

// capturedPost mirrors the fields of a Mattermost post the test API server
// needs to inspect.
type capturedPost struct {
	Message   string `json:"message"`
	ChannelId string `json:"channel_id"`
	Props     struct {
		Attachments []struct {
			Title string `json:"title"`
		} `json:"attachments"`
	} `json:"props"`
}

// apiCaptureServer is a stand-in Mattermost server that records every post
// created through it.
type apiCaptureServer struct {
	server *httptest.Server
	mu     sync.Mutex
	posts  []capturedPost
}

func newAPICaptureServer() *apiCaptureServer {
	c := &apiCaptureServer{}
	c.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p capturedPost
		_ = json.NewDecoder(r.Body).Decode(&p)
		c.mu.Lock()
		c.posts = append(c.posts, p)
		c.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"post1"}`))
	}))
	return c
}

func (c *apiCaptureServer) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.posts)
}

func (c *apiCaptureServer) snapshot() []capturedPost {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]capturedPost(nil), c.posts...)
}

// newSourceTestBot returns a Bot wired to the capture server. Metric fields
// are assigned before Register: the promauto registration inside Setup runs
// only once per process, so later bots keep these safe dummies.
func newSourceTestBot(c *apiCaptureServer, channels ...*model.Channel) *bot.Bot {
	b := &bot.Bot{
		Client:   model.NewAPIv4Client(c.server.URL),
		Channels: channels,
	}
	b.Monitoring.SuccessfulPosts = prometheus.NewCounter(prometheus.CounterOpts{Name: "test_src_successful_posts"})
	b.Monitoring.SuccessfulPatches = prometheus.NewCounter(prometheus.CounterOpts{Name: "test_src_successful_patches"})
	b.Monitoring.FailedPatches = prometheus.NewCounter(prometheus.CounterOpts{Name: "test_src_failed_patches"})
	b.Monitoring.SuccessfulSourceFetches = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_src_successful_source_fetches"}, []string{"source"})
	b.Monitoring.FailedSourceFetches = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_src_failed_source_fetches"}, []string{"source"})
	b.Monitoring.CitiesHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_src_cities"})
	b.Monitoring.RegionsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_src_regions"})
	b.Monitoring.TimeOfDayHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_src_time_of_day"})
	b.Monitoring.DayOfWeekHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_src_day_of_week"})
	b.Register()
	return b
}

func channelUpdate(text string, channelID int64) *tg.UpdateNewChannelMessage {
	return &tg.UpdateNewChannelMessage{
		Message: &tg.Message{
			Out:     false,
			PeerID:  &tg.PeerChannel{ChannelID: channelID},
			Message: text,
			Date:    int(time.Now().Unix()),
		},
	}
}

func waitPosts(t *testing.T, c *apiCaptureServer, want int) []capturedPost {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for c.count() < want && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	return c.snapshot()
}

func TestParseMessage_EarlyAlert(t *testing.T) {
	c := newAPICaptureServer()
	defer c.server.Close()
	channel := &model.Channel{Id: "ch_early", Name: "alerts", DisplayName: "Alerts", Type: model.ChannelTypeOpen}
	b := newSourceTestBot(c, channel)
	go b.AwaitMessage()

	// The early-alert announcement phrase plus a city line: the message must
	// be processed as a rockets alert despite carrying no rocket keywords.
	text := `(10/10/2024) 11:19 התרעה מוקדמת: בדקות הקרובות צפויות להתקבל התרעות באזורך.

אזור קו העימות
מטולה (מיידי)`
	// Message time: (10/10/2024) 11:19 IDT (08:19 UTC); now is 30s after.
	now := time.Date(2024, 10, 10, 8, 19, 30, 0, time.UTC)

	err := processMessage(text, district.GetDistricts(), now, b, "rockets")
	if err != nil {
		t.Fatalf("processMessage() error = %v, want nil", err)
	}

	posts := waitPosts(t, c, 1)
	if len(posts) < 1 {
		t.Fatal("no post reached the Mattermost API server")
	}
	if posts[0].ChannelId != channel.Id {
		t.Errorf("ChannelId = %q, want %q", posts[0].ChannelId, channel.Id)
	}
	if len(posts[0].Props.Attachments) == 0 || posts[0].Props.Attachments[0].Title != "Rocket and missile fire" {
		t.Errorf("attachment title = %+v, want the rockets category title", posts[0].Props.Attachments)
	}
}

func TestParseMessage_Channel2335255539(t *testing.T) {
	target := &model.Channel{Id: "target_channel_id", Name: "telegram-2335255539", DisplayName: "Israel News", Type: model.ChannelTypeOpen}

	t.Run("keyword found and channel present forwards the message", func(t *testing.T) {
		c := newAPICaptureServer()
		defer c.server.Close()
		b := newSourceTestBot(c, target, &model.Channel{Id: "other_id", Name: "some-other-channel", DisplayName: "Other", Type: model.ChannelTypeOpen})

		text := "This message contains a ירוט keyword."
		err := (&SourceTelegram{Bot: b}).ParseMessage(context.Background(), tg.Entities{}, channelUpdate(text, 2335255539))
		if err != nil {
			t.Fatalf("ParseMessage() error = %v, want nil", err)
		}

		posts := waitPosts(t, c, 1)
		if len(posts) != 1 {
			t.Fatalf("posts = %d, want 1", len(posts))
		}
		if want := "חדשות ישראל בטלגרם: " + text; posts[0].Message != want {
			t.Errorf("Message = %q, want %q", posts[0].Message, want)
		}
		if posts[0].ChannelId != target.Id {
			t.Errorf("ChannelId = %q, want %q", posts[0].ChannelId, target.Id)
		}
	})

	t.Run("keyword found but channel missing does not post", func(t *testing.T) {
		c := newAPICaptureServer()
		defer c.server.Close()
		b := newSourceTestBot(c, &model.Channel{Id: "other_id", Name: "another-channel", DisplayName: "Other", Type: model.ChannelTypeOpen})

		err := (&SourceTelegram{Bot: b}).ParseMessage(context.Background(), tg.Entities{}, channelUpdate("Alert: אזעק in the area!", 2335255539))
		if err != nil {
			t.Fatalf("ParseMessage() error = %v, want nil", err)
		}
		if got := c.count(); got != 0 {
			t.Errorf("posts = %d, want 0 (no matching Mattermost channel)", got)
		}
	})

	t.Run("keyword absent does not post", func(t *testing.T) {
		c := newAPICaptureServer()
		defer c.server.Close()
		b := newSourceTestBot(c, target)

		err := (&SourceTelegram{Bot: b}).ParseMessage(context.Background(), tg.Entities{}, channelUpdate("A quiet day, no news here.", 2335255539))
		if err != nil {
			t.Fatalf("ParseMessage() error = %v, want nil", err)
		}
		if got := c.count(); got != 0 {
			t.Errorf("posts = %d, want 0 (no keyword matched)", got)
		}
	})
}
