package bot

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/phntom/goalert/internal/config"
	"github.com/phntom/goalert/internal/district"
	"github.com/phntom/goalert/internal/monitoring"
)

func setupTestBot(t *testing.T) *Bot {
	t.Helper()
	b := &Bot{
		alertFeed:  make(chan *Message, 50),
		dedup:      make(map[district.ID]*Message),
		userId:     "test_bot_user_id",
		Monitoring: monitoring.Monitoring{},
		// Client needs to be non-nil to avoid panics if any execute... functions are called.
		Client: model.NewAPIv4Client("http://localhost:8065"), // Dummy client
	}
	// Monitoring.Setup() must not be called here: it registers promauto collectors
	// on the default registry and breaks tests that run in parallel.
	b.Monitoring.SuccessfulPosts = prometheus.NewCounter(prometheus.CounterOpts{Name: "test_successful_posts"})
	b.Monitoring.SuccessfulPatches = prometheus.NewCounter(prometheus.CounterOpts{Name: "test_successful_patches"})
	b.Monitoring.FailedPatches = prometheus.NewCounter(prometheus.CounterOpts{Name: "test_failed_patches"})
	b.Monitoring.CitiesHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_cities_histogram"})
	b.Monitoring.RegionsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_regions_histogram"})
	b.Monitoring.TimeOfDayHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_time_of_day_histogram"})
	b.Monitoring.DayOfWeekHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_day_of_week_histogram"})
	// SuccessfulSourceFetches and FailedSourceFetches are CounterVecs that are not
	// used by Bot.UpdateMonitor, so they are left nil here.

	dummyChannel := &model.Channel{
		Id:          "test_channel_id_" + model.NewId(),
		Name:        "test-channel",
		DisplayName: "Test Channel", // English display name for ChannelToLanguage default
		TeamId:      "test_team_id",
		Type:        model.ChannelTypeOpen,
	}
	b.Channels = []*model.Channel{dummyChannel}
	return b
}

func createTestMessage(t *testing.T, numCities int, category string, instructions string, safetySeconds int) *Message {
	t.Helper()
	cities := make([]district.ID, numCities)
	rocketIDs := make(map[string]bool)
	for i := range numCities {
		cityID := district.ID(fmt.Sprintf("city%d", i+1))
		cities[i] = cityID
		if i%3 == 0 {
			rocketIDs[string(cityID)] = true
		}
	}

	msg := NewMessage(instructions, category, safetySeconds, "2024-01-15T10:00:00Z")
	msg.Cities = cities
	msg.RocketIDs = rocketIDs
	msg.PostMutex.Lock()
	msg.Prerender()
	msg.PostMutex.Unlock()
	return &msg
}

// runAwaitMessageTest feeds msgToTest into the bot and verifies AwaitMessage exits
// cleanly once the feed is closed.
func runAwaitMessageTest(t *testing.T, b *Bot, msgToTest *Message) {
	t.Helper()

	var wg sync.WaitGroup
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("AwaitMessage panicked: %v", r)
			}
		}()
		b.AwaitMessage()
	})

	b.alertFeed <- msgToTest
	time.Sleep(250 * time.Millisecond)

	close(b.alertFeed)
	wg.Wait()
}

func TestAwaitMessage_SplittingLogic(t *testing.T) {
	t.Run("NoSplit_LessThan20Cities", func(t *testing.T) {
		b := setupTestBot(t)
		msg := createTestMessage(t, 5, "rockets", "instr_lessthan20", 60)
		runAwaitMessageTest(t, b, msg)
	})

	t.Run("NoSplit_Exactly20Cities", func(t *testing.T) {
		b := setupTestBot(t)
		msg := createTestMessage(t, 20, "uav", "instr_exact20", 60)
		runAwaitMessageTest(t, b, msg)
	})

	t.Run("NoSplit_ZeroCities", func(t *testing.T) {
		b := setupTestBot(t)
		msg := createTestMessage(t, 0, "other", "instr_zero", 60)
		runAwaitMessageTest(t, b, msg)
	})

	t.Run("Split_MoreThan20Cities_25", func(t *testing.T) {
		b := setupTestBot(t)
		runAwaitMessageTest(t, b, createTestMessage(t, 25, "infiltration", "instr_25cities", 90))
	})

	t.Run("Split_ExactMultipleOf20Cities_40", func(t *testing.T) {
		b := setupTestBot(t)
		runAwaitMessageTest(t, b, createTestMessage(t, 40, "rockets", "instr_40cities", 120))
	})

	t.Run("PropertyPreservation_Split_RunsWithoutPanic", func(t *testing.T) {
		b := setupTestBot(t)
		runAwaitMessageTest(t, b, createTestMessage(t, 22, "uav", "split_prop_test", 120))
	})
}

// recorder captures calls made through the Mattermost API seam and returns
// plausible fake results.
type recorder struct {
	mu        sync.Mutex
	submits   []*model.Post // in call order, with the assigned fake ID set
	patches   []string      // patched post IDs in call order
	reactions []*model.Reaction
	next      int
}

func (r *recorder) submitCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.submits)
}

func (r *recorder) submitsFor(channelID string) []*model.Post {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*model.Post
	for _, post := range r.submits {
		if post.ChannelId == channelID {
			out = append(out, post)
		}
	}
	return out
}

func (r *recorder) patchCount(postID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, id := range r.patches {
		if id == postID {
			count++
		}
	}
	return count
}

func (r *recorder) reactionsFor(emoji string) []*model.Reaction {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*model.Reaction
	for _, reaction := range r.reactions {
		if reaction.EmojiName == emoji {
			out = append(out, reaction)
		}
	}
	return out
}

func newTestChannel(id, name, display string) *model.Channel {
	return &model.Channel{
		Id:          id,
		Name:        name,
		DisplayName: display, // Latin script -> English rendering
		TeamId:      "test_team_id",
		Type:        model.ChannelTypeOpen,
	}
}

// recorderBot returns a bot whose Mattermost API seam records every call.
func recorderBot(t *testing.T, channels ...*model.Channel) (*Bot, *recorder) {
	t.Helper()
	b := setupTestBot(t)
	if len(channels) > 0 {
		b.Channels = channels
	}
	rec := &recorder{}
	b.apiSubmitPost = func(_ context.Context, post *model.Post) (*model.Post, *model.Response, error) {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.next++
		created := post.Clone()
		created.Id = fmt.Sprintf("p%d", rec.next)
		rec.submits = append(rec.submits, created)
		return created, nil, nil
	}
	b.apiPatchPost = func(_ context.Context, postID string, _ *model.PostPatch) (*model.Post, *model.Response, error) {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.patches = append(rec.patches, postID)
		return nil, nil, nil
	}
	b.apiSaveReaction = func(_ context.Context, reaction *model.Reaction) (*model.Reaction, *model.Response, error) {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.reactions = append(rec.reactions, reaction)
		return nil, nil, nil
	}
	return b, rec
}

// startAwaitMessage runs AwaitMessage in the background and returns a func
// that stops it once the test is done with the bot.
func startAwaitMessage(t *testing.T, b *Bot) func() {
	t.Helper()
	var wg sync.WaitGroup
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("AwaitMessage panicked: %v", r)
			}
		}()
		b.AwaitMessage()
	})
	return func() {
		close(b.alertFeed)
		wg.Wait()
	}
}

// waitUntil polls cond until it holds or the timeout expires. Callers assert
// the condition separately after the wait.
func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
}

// testDistrictIDs returns n deterministic real district IDs from the embedded
// dataset, so rendering (which looks up district names) works.
func testDistrictIDs(t *testing.T, n int) []district.ID {
	t.Helper()
	byID := district.GetDistricts()["en"]
	ids := make([]district.ID, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	if len(ids) < n {
		t.Fatalf("embedded dataset has %d districts, need %d", len(ids), n)
	}
	return ids[:n]
}

func TestAwaitMessage_SplitsLargeAlerts(t *testing.T) {
	chA := newTestChannel("ch_a", "alerts", "Alerts")
	chB := newTestChannel("ch_b", "alerts2", "Alerts 2")
	b, rec := recorderBot(t, chA, chB)
	finish := startAwaitMessage(t, b)
	defer finish()

	ids := testDistrictIDs(t, 45)
	msg := NewMessage("instructions", "rockets", 60, "10:00")
	for _, id := range ids {
		msg.AppendDistrict(id)
	}
	b.alertFeed <- &msg

	waitUntil(t, 2*time.Second, func() bool { return rec.submitCount() == 6 })
	if got := rec.submitCount(); got != 6 {
		t.Fatalf("submits = %d, want 6 (3 chunks x 2 channels)", got)
	}

	districts := district.GetDistricts()["en"]
	wantSizes := []int{20, 20, 5}
	for _, ch := range []*model.Channel{chA, chB} {
		posts := rec.submitsFor(ch.Id)
		if len(posts) != len(wantSizes) {
			t.Fatalf("channel %s: %d posts, want %d", ch.Id, len(posts), len(wantSizes))
		}
		seen := make(map[string]bool)
		for i, post := range posts {
			// One hashtag is rendered per city in the chunk.
			if got := strings.Count(post.Message, "#"); got != wantSizes[i] {
				t.Errorf("channel %s post %d: %d hashtags, want %d", ch.Id, i, got, wantSizes[i])
			}
			for field := range strings.FieldsSeq(post.Message) {
				if strings.HasPrefix(field, "#") {
					seen[field] = true
				}
			}
			attachments, ok := post.Props["attachments"].([]*model.MessageAttachment)
			if !ok || len(attachments) != 1 {
				t.Fatalf("channel %s post %d: no attachment in props", ch.Id, i)
			}
			// Category and safety time are preserved on every chunk.
			if got, want := attachments[0].Title, "Rocket and missile fire"; got != want {
				t.Errorf("channel %s post %d: title = %q, want %q", ch.Id, i, got, want)
			}
			if got, want := attachments[0].Text, "You have 60 seconds to seek shelter"; got != want {
				t.Errorf("channel %s post %d: text = %q, want %q", ch.Id, i, got, want)
			}
		}
		// The union of the chunks covers every city of the original message.
		for _, id := range ids {
			want := "#" + district.CityNameCleanFull(districts[id].SettlementName)
			if !seen[want] {
				t.Errorf("channel %s: chunk posts miss city hashtag %s (id %s)", ch.Id, want, id)
			}
		}
	}
}

func TestAwaitMessage_DedupPatchesPreviousPost(t *testing.T) {
	chA := newTestChannel("ch_a", "alerts", "Alerts")
	chB := newTestChannel("ch_b", "alerts2", "Alerts 2")
	b, rec := recorderBot(t, chA, chB)
	finish := startAwaitMessage(t, b)
	defer finish()

	city := testDistrictIDs(t, 1)[0]

	first := NewMessage("instructions", "rockets", 60, "10:00")
	first.AppendDistrict(city)
	b.alertFeed <- &first
	waitUntil(t, 2*time.Second, func() bool { return rec.submitCount() == 2 })
	if got := rec.submitCount(); got != 2 {
		t.Fatalf("submits after first message = %d, want 2", got)
	}

	// Same city and category, no new rocket IDs: no new post is created, the
	// previous posts are patched instead (the patch arrives well within the
	// 200 ms the post-submit patch goroutines sleep).
	second := NewMessage("uav_instructions", "rockets", 60, "10:01")
	second.AppendDistrict(city)
	b.alertFeed <- &second
	waitUntil(t, 2*time.Second, func() bool {
		return rec.patchCount("p1") >= 1 && rec.patchCount("p2") >= 1
	})
	if got := rec.submitCount(); got != 2 {
		t.Errorf("submits after duplicate message = %d, want 2 (no new post)", got)
	}
	if rec.patchCount("p1") < 1 || rec.patchCount("p2") < 1 {
		t.Errorf("duplicate message did not patch previous posts: p1=%d p2=%d",
			rec.patchCount("p1"), rec.patchCount("p2"))
	}

	// Let the first message's patch goroutines settle, then check the exact
	// patch counts: one from the dedup merge, one from the goroutine.
	time.Sleep(400 * time.Millisecond)
	for _, id := range []string{"p1", "p2"} {
		if got := rec.patchCount(id); got != 2 {
			t.Errorf("patch count for %s = %d, want 2", id, got)
		}
	}

	// A message with a new rocket ID for the same city creates a new post
	// (NewRocketIDsPresent semantics).
	third := NewMessage("instructions", "rockets", 60, "10:02")
	third.AppendDistrict(city)
	third.RocketIDs["new-rocket-id"] = true
	b.alertFeed <- &third
	waitUntil(t, 2*time.Second, func() bool { return rec.submitCount() == 4 })
	if got := rec.submitCount(); got != 4 {
		t.Fatalf("submits after message with new rocket ID = %d, want 4", got)
	}
}

func TestAwaitMessage_ZeroCities(t *testing.T) {
	chA := newTestChannel("ch_a", "alerts", "Alerts")
	chB := newTestChannel("ch_b", "alerts2", "Alerts 2")
	b, rec := recorderBot(t, chA, chB)
	finish := startAwaitMessage(t, b)
	defer finish()

	msg := NewMessage("instructions", "rockets", 60, "10:00")
	b.alertFeed <- &msg

	waitUntil(t, 2*time.Second, func() bool { return rec.submitCount() == 2 })
	if got := rec.submitCount(); got != 2 {
		t.Fatalf("submits for zero-city message = %d, want 2 (one per channel)", got)
	}
}

func TestAwaitMessage_Reaction(t *testing.T) {
	chA := newTestChannel("ch_a", "alerts", "Alerts")
	chB := newTestChannel("ch_b", "alerts2", "Alerts 2")
	b, rec := recorderBot(t, chA, chB)
	finish := startAwaitMessage(t, b)
	defer finish()

	msg := NewMessage("uav_instructions", "uav", 0, "10:00")
	msg.AppendDistrict(testDistrictIDs(t, 1)[0])
	b.alertFeed <- &msg

	waitUntil(t, 2*time.Second, func() bool { return len(rec.reactionsFor("uav-alert")) == 2 })
	reactions := rec.reactionsFor("uav-alert")
	if len(reactions) != 2 {
		t.Fatalf("uav-alert reactions = %d, want 2 (one per post)", len(reactions))
	}
	postIDs := map[string]bool{"p1": true, "p2": true}
	for _, reaction := range reactions {
		if !postIDs[reaction.PostId] {
			t.Errorf("reaction on unexpected post %q", reaction.PostId)
		}
	}
}

func TestChannelToLanguage(t *testing.T) {
	tests := []struct {
		name    string
		display string
		want    config.Language
	}{
		{"hebrew", "התרעות", "he"},
		{"arabic", "تنبيه", "ar"},
		{"cyrillic", "Ракеты", "ru"},
		{"latin", "Alerts", "en"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &model.Channel{DisplayName: tt.display}
			if got := ChannelToLanguage(channel); got != tt.want {
				t.Errorf("ChannelToLanguage(%q) = %q, want %q", tt.display, got, tt.want)
			}
		})
	}
}

func TestRegisterInitialization(t *testing.T) {
	b := &Bot{}
	b.Register()

	if b.alertFeed == nil {
		t.Fatal("Register did not initialize alertFeed")
	}
	if b.dedup == nil {
		t.Fatal("Register did not initialize dedup")
	}
	if b.apiSubmitPost == nil || b.apiPatchPost == nil || b.apiSaveReaction == nil {
		t.Fatal("Register did not initialize the Mattermost API seam")
	}
}

func TestRegisterAndSubmitMessage(t *testing.T) {
	// Dummies are assigned before Register: once registerMetricsOnce has
	// fired for this test binary, Setup no longer overwrites the fields,
	// so UpdateMonitor keeps working on the dummies.
	b := &Bot{}
	b.Monitoring.CitiesHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_reg_cities_histogram"})
	b.Monitoring.RegionsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_reg_regions_histogram"})
	b.Monitoring.TimeOfDayHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_reg_time_of_day"})
	b.Monitoring.DayOfWeekHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_reg_day_of_week"})
	b.Register()

	msg := createTestMessage(t, 2, "rockets", "instructions", 60)

	// alertFeed is unbuffered: read it in the background so SubmitMessage
	// can deliver, then check that the exact message came through and that
	// UpdateMonitor (which observes the CitiesHistogram) did not panic.
	delivered := make(chan *Message, 1)
	go func() {
		delivered <- <-b.alertFeed
	}()
	b.SubmitMessage(msg)

	select {
	case got := <-delivered:
		if got != msg {
			t.Error("SubmitMessage delivered a different message than the one submitted")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SubmitMessage did not deliver to alertFeed within 2s")
	}
}
