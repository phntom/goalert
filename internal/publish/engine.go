package publish

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/area"
	"github.com/phntom/goalert/internal/i18n"
	"github.com/phntom/goalert/internal/metrics"
)

const (
	maxAreasPerPost = 20
	defaultPatch    = 200 * time.Millisecond
	alertTTL        = 3 * time.Minute
	preAlertTTL     = 90 * time.Second
	endTTL          = 30 * time.Second
	cleanupInterval = time.Second
)

type postRef struct {
	channelID string
	postID    string
	lang      i18n.Language
}

// Incident is the live state for an alert affecting a set of areas: the posts
// it produced, the source ids seen, and when it expires.
type Incident struct {
	category alert.Category
	areas    []*area.Area
	safety   int
	at       time.Time

	mu     sync.Mutex
	ids    map[string]bool
	posts  []postRef
	expire time.Time
}

func (inc *Incident) expired() bool {
	inc.mu.Lock()
	defer inc.mu.Unlock()
	return time.Now().After(inc.expire)
}

func (inc *Incident) addIDs(ids []string) (added bool) {
	inc.mu.Lock()
	defer inc.mu.Unlock()
	for _, id := range ids {
		if !inc.ids[id] {
			inc.ids[id] = true
			added = true
		}
	}
	return added
}

func (inc *Incident) addPost(p postRef) {
	inc.mu.Lock()
	inc.posts = append(inc.posts, p)
	inc.mu.Unlock()
}

func (inc *Incident) snapshotPosts() []postRef {
	inc.mu.Lock()
	defer inc.mu.Unlock()
	return slices.Clone(inc.posts)
}

func (inc *Incident) markEnded() {
	inc.mu.Lock()
	inc.category = alert.CatEndAlert
	inc.expire = time.Now().Add(endTTL)
	inc.mu.Unlock()
}

// Engine consumes alerts and drives Mattermost posting/patching.
type Engine struct {
	chat       ChatClient
	areas      *area.Set
	metrics    *metrics.Metrics
	patchDelay time.Duration

	mu   sync.Mutex
	live map[string]*Incident
}

// NewEngine builds an engine that publishes to chat.
func NewEngine(chat ChatClient, areas *area.Set, m *metrics.Metrics) *Engine {
	return &Engine{
		chat: chat, areas: areas, metrics: m, patchDelay: defaultPatch,
		live: make(map[string]*Incident),
	}
}

// Run consumes alerts until ctx is cancelled or in is closed.
func (e *Engine) Run(ctx context.Context, in <-chan alert.Alert) {
	go e.cleanup(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case a, ok := <-in:
			if !ok {
				return
			}
			e.handle(a)
		}
	}
}

func (e *Engine) handle(a alert.Alert) {
	areas, unknown := e.resolve(a.Areas)
	for _, u := range unknown {
		mlog.Warn("unresolved area", mlog.String("source", a.Source), mlog.String("area", u))
	}
	if len(areas) == 0 {
		return
	}
	if a.Kind == alert.KindEnd {
		e.handleEnd(areas)
		return
	}
	fresh, toPatch := e.reconcile(a, areas)
	for _, inc := range fresh {
		e.publish(inc)
	}
	for _, inc := range toPatch {
		e.repatch(inc)
	}
}

// resolve maps raw source names to canonical, de-duplicated areas.
func (e *Engine) resolve(raw []string) (areas []*area.Area, unknown []string) {
	seen := make(map[string]bool)
	for _, name := range raw {
		rs, ok := e.areas.Resolve(name)
		if !ok {
			if !area.IsPartial(name) {
				unknown = append(unknown, name)
			}
			continue
		}
		for _, a := range rs {
			if !seen[a.ID] {
				seen[a.ID] = true
				areas = append(areas, a)
			}
		}
	}
	return areas, unknown
}

// reconcile partitions areas into fresh incidents to post and existing
// incidents to patch (because new source ids arrived), all under one lock.
func (e *Engine) reconcile(a alert.Alert, areas []*area.Area) (fresh, toPatch []*Incident) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var freshAreas []*area.Area
	patched := make(map[*Incident]bool)
	for _, ar := range areas {
		inc, ok := e.live[ar.ID]
		if ok && !inc.expired() && inc.category == a.Category {
			if inc.addIDs(a.IDs) && !patched[inc] {
				patched[inc] = true
				toPatch = append(toPatch, inc)
			}
			continue
		}
		freshAreas = append(freshAreas, ar)
	}
	for _, chunk := range chunkAreas(freshAreas, maxAreasPerPost) {
		inc := newIncident(a, chunk)
		for _, ar := range chunk {
			e.live[ar.ID] = inc
		}
		fresh = append(fresh, inc)
	}
	return fresh, toPatch
}

func (e *Engine) publish(inc *Incident) {
	e.observe(inc)
	for _, ch := range e.chat.Channels() {
		post := Render(inc, ch.Lang)
		post.ChannelId = ch.ID
		id, err := e.chat.CreatePost(ch.ID, post)
		if err != nil {
			mlog.Error("create post failed", mlog.String("channel", ch.ID), mlog.Err(err))
			continue
		}
		inc.addPost(postRef{channelID: ch.ID, postID: id, lang: ch.Lang})
		go e.finalize(inc.category, id, post)
	}
}

// finalize reacts to the post and, after the patch delay, strips the trigger
// text leaving only the card — the second half of the post→patch trick.
func (e *Engine) finalize(category alert.Category, postID string, post *model.Post) {
	if emoji := reactionEmoji(category); emoji != "" {
		if err := e.chat.AddReaction(postID, emoji); err != nil {
			mlog.Warn("add reaction failed", mlog.String("emoji", emoji), mlog.Err(err))
		}
	}
	time.Sleep(e.patchDelay)
	if err := e.chat.PatchPost(postID, post.Props); err != nil {
		mlog.Error("patch post failed", mlog.String("postID", postID), mlog.Err(err))
	}
}

func (e *Engine) repatch(inc *Incident) {
	for _, p := range inc.snapshotPosts() {
		post := Render(inc, p.lang)
		if err := e.chat.PatchPost(p.postID, post.Props); err != nil {
			mlog.Error("repatch failed", mlog.String("postID", p.postID), mlog.Err(err))
		}
	}
}

func (e *Engine) handleEnd(areas []*area.Area) {
	e.mu.Lock()
	seen := make(map[*Incident]bool)
	var ended []*Incident
	for _, ar := range areas {
		if inc, ok := e.live[ar.ID]; ok && !seen[inc] {
			seen[inc] = true
			inc.markEnded()
			ended = append(ended, inc)
		}
	}
	e.mu.Unlock()
	for _, inc := range ended {
		e.repatch(inc)
	}
}

func (e *Engine) observe(inc *Incident) {
	e.metrics.AreasPerAlert.Observe(float64(len(inc.areas)))
	regions := make(map[string]bool)
	for _, a := range inc.areas {
		if a.Region != "" {
			regions[a.Region] = true
		}
	}
	e.metrics.RegionsPerAlert.Observe(float64(len(regions)))
	now := time.Now()
	e.metrics.HourOfDay.Observe(float64(now.Hour()))
	e.metrics.DayOfWeek.Observe(float64(now.Weekday()))
}

func (e *Engine) cleanup(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.mu.Lock()
			for id, inc := range e.live {
				if inc.expired() {
					delete(e.live, id)
				}
			}
			e.mu.Unlock()
		}
	}
}

func newIncident(a alert.Alert, areas []*area.Area) *Incident {
	inc := &Incident{
		category: a.Category, areas: areas, safety: minMigun(areas),
		at: a.At, ids: make(map[string]bool), expire: time.Now().Add(ttlFor(a.Kind)),
	}
	for _, id := range a.IDs {
		inc.ids[id] = true
	}
	return inc
}

func ttlFor(k alert.Kind) time.Duration {
	if k == alert.KindPreAlert {
		return preAlertTTL
	}
	return alertTTL
}

func minMigun(areas []*area.Area) int {
	m := 0
	for _, a := range areas {
		if a.MigunTime > 0 && (m == 0 || a.MigunTime < m) {
			m = a.MigunTime
		}
	}
	return m
}

func chunkAreas(areas []*area.Area, size int) [][]*area.Area {
	var out [][]*area.Area
	for i := 0; i < len(areas); i += size {
		end := min(i+size, len(areas))
		out = append(out, areas[i:end])
	}
	return out
}

func reactionEmoji(c alert.Category) string {
	switch c {
	case alert.CatUAV:
		return "uav-alert"
	case alert.CatTerror:
		return "infiltration-alert"
	default:
		return ""
	}
}
