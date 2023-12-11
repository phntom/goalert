package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/phntom/goalert/internal/config"
	"github.com/phntom/goalert/internal/district"
	"strconv"
	"time"
)

type Message struct {
	Instructions   string
	Category       string
	SafetySeconds  uint
	Cities         []district.ID
	RocketIDs      map[string]bool
	Rendered       map[config.Language]*model.Post
	Expire         time.Time
	PostIDs        []string
	ChannelsPosted []*model.Channel
	Changed        bool
	PubDate        string
}

func (m *Message) GetHash() string {
	var rocketID string
	for r := range m.RocketIDs {
		rocketID = r
		break
	}
	data := m.Instructions + m.Category + strconv.FormatUint(uint64(m.SafetySeconds), 10) + rocketID + m.PubDate
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (m *Message) PostForChannel(c *model.Channel) *model.Post {
	if len(m.Rendered) == 0 {
		m.Prerender()
	}
	lang := ChannelToLanguage(c)
	post := m.Rendered[lang].Clone()
	post.ChannelId = c.Id
	return post
}

func (m *Message) Prerender() {
	m.Rendered = make(map[config.Language]*model.Post, len(config.Languages))
	for _, lang := range config.Languages {
		m.Rendered[lang] = Render(m, lang)
	}
}

func (m *Message) IsExpired() bool {
	return time.Now().After(m.Expire)
}

func (m *Message) PatchData(n *Message) bool {
	if m.Category == "" && n.Category != "" {
		// m is ynet, n is oref, should patch title
		m.Category = n.Category
		m.Instructions = n.Instructions
		m.Changed = true
	}
	if m.RocketIDs == nil {
		m.RocketIDs = make(map[string]bool)
	}
	if n.RocketIDs == nil {
		n.RocketIDs = make(map[string]bool)
	}
	for rocketID := range n.RocketIDs {
		if m.RocketIDs[rocketID] {
			// no new information
			continue
		}
		m.Changed = true
		m.RocketIDs[rocketID] = true
	}
	if m.Changed {
		//m.Expire = time.Now().Add(5 * time.Second)
		m.Changed = false
		m.Prerender()
		return true
	}
	return false
}
