package publish

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/area"
	"github.com/phntom/goalert/internal/i18n"
	"github.com/phntom/goalert/internal/metrics"
)

type fakeChat struct {
	mu        sync.Mutex
	channels  []Channel
	created   map[string]*model.Post
	patched   map[string]map[string]any
	patchN    int
	reactions []string
	seq       int
}

func newFakeChat(langs ...i18n.Language) *fakeChat {
	f := &fakeChat{created: map[string]*model.Post{}, patched: map[string]map[string]any{}}
	for i, l := range langs {
		f.channels = append(f.channels, Channel{ID: "ch" + string(rune('a'+i)), Lang: l})
	}
	return f
}

func (f *fakeChat) Channels() []Channel { return f.channels }

func (f *fakeChat) CreatePost(channelID string, post *model.Post) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := "p" + string(rune('0'+f.seq))
	clone := post.Clone()
	f.created[id] = clone
	return id, nil
}

func (f *fakeChat) PatchPost(postID string, props map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.patchN++
	f.patched[postID] = props
	return nil
}

func (f *fakeChat) AddReaction(postID, emoji string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reactions = append(f.reactions, emoji)
	return nil
}

func (f *fakeChat) createdCount() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.created) }
func (f *fakeChat) patchCount() int   { f.mu.Lock(); defer f.mu.Unlock(); return f.patchN }

func testAreas() *area.Set {
	return area.NewSet([]*area.Area{
		{ID: "1", MigunTime: 30, Region: "דן", Names: map[i18n.Language]string{"he": "תל אביב - יפו", "en": "Tel Aviv - Yafo"}},
		{ID: "2", MigunTime: 15, Region: "גליל", Names: map[i18n.Language]string{"he": "קרית שמונה", "en": "Kiryat Shmona"}},
	})
}

func newTestEngine(chat ChatClient) *Engine {
	e := NewEngine(chat, testAreas(), metrics.New())
	e.patchDelay = time.Millisecond
	return e
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func missileAlert(areas ...string) alert.Alert {
	return alert.Alert{Category: alert.CatMissile, Kind: alert.KindAlert, Areas: areas, At: time.Now(), Source: "test", IDs: []string{"id1"}}
}

func attachment(props map[string]any) *model.SlackAttachment {
	return props["attachments"].([]*model.SlackAttachment)[0]
}

func TestPostThenPatchLifecycle(t *testing.T) {
	chat := newFakeChat("he", "en")
	e := newTestEngine(chat)

	e.handle(missileAlert("תל אביב - יפו"))

	if chat.createdCount() != 2 {
		t.Fatalf("expected a post per channel, got %d", chat.createdCount())
	}
	// The initial post must carry the full trigger text with mentions.
	var he *model.Post
	for _, p := range chat.created {
		if strings.Contains(p.Message, "צבעאדום") {
			he = p
		}
	}
	if he == nil {
		t.Fatal("no Hebrew post with mention prefix found")
	}
	if !strings.Contains(he.Message, "תוך 30 שניות") {
		t.Errorf("expected shelter-seconds in trigger text, got %q", he.Message)
	}
	att := attachment(he.Props)
	if att.Title == "" || len(att.Fields) == 0 || att.Fields[0].Title != "תל אביב" {
		t.Errorf("card not built correctly: %+v", att)
	}
	// The patch (card-only) must arrive after the delay.
	waitFor(t, func() bool { return chat.patchCount() == 2 })
}

func TestDedupSuppressesRepost(t *testing.T) {
	chat := newFakeChat("he")
	e := newTestEngine(chat)
	e.handle(missileAlert("תל אביב - יפו"))
	waitFor(t, func() bool { return chat.patchCount() == 1 })
	created := chat.createdCount()
	// Same area + category + same id => nothing new.
	e.handle(missileAlert("תל אביב - יפו"))
	if chat.createdCount() != created {
		t.Errorf("duplicate alert should not repost: %d -> %d", created, chat.createdCount())
	}
}

func TestEndAlertPatchesToAllClear(t *testing.T) {
	chat := newFakeChat("he")
	e := newTestEngine(chat)
	e.handle(missileAlert("תל אביב - יפו"))
	waitFor(t, func() bool { return chat.patchCount() == 1 })

	end := alert.Alert{Category: alert.CatEndAlert, Kind: alert.KindEnd, Areas: []string{"תל אביב - יפו"}, At: time.Now(), Source: "test"}
	e.handle(end)

	waitFor(t, func() bool { return chat.patchCount() >= 2 })
	chat.mu.Lock()
	defer chat.mu.Unlock()
	for _, props := range chat.patched {
		if attachment(props).Color == clearColor {
			return
		}
	}
	t.Error("expected an all-clear (green) patch after end alert")
}

func TestRenderImmediateWhenNoShelterTime(t *testing.T) {
	inc := &Incident{category: alert.CatMissile, areas: testAreas().All()[:1], safety: 0, ids: map[string]bool{}}
	post := Render(inc, "en")
	if !strings.Contains(attachment(post.Props).Text, "Immediately") {
		t.Errorf("expected immediate instruction, got %q", attachment(post.Props).Text)
	}
}
