package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/phntom/goalert/internal/config"
	"github.com/phntom/goalert/internal/district"
	"strconv"
	"sync"
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
	PostMutex      sync.Mutex
	PostIDs        []string
	ChannelsPosted []*model.Channel
	Changed        bool
	PubDate        string
}

func NewMessage(instructions string, category string, safetySeconds int, pubDate string) Message {
	return Message{
		Instructions:   instructions,
		Category:       category,
		SafetySeconds:  uint(safetySeconds),
		Cities:         nil,
		RocketIDs:      make(map[string]bool),
		Rendered:       make(map[config.Language]*model.Post, len(config.Languages)),
		Expire:         time.Now().Add(max(time.Second*30, time.Duration(safetySeconds))),
		PostMutex:      sync.Mutex{},
		PostIDs:        nil,
		ChannelsPosted: nil,
		Changed:        true,
		PubDate:        pubDate,
	}
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
	for _, lang := range config.Languages {
		m.Rendered[lang] = Render(m, lang)
	}
}

func (m *Message) IsExpired() bool {
	return time.Now().After(m.Expire)
}

func (m *Message) PatchData(n *Message) bool {
	if n.Category != "" && m.Category != n.Category {
		m.Category = n.Category
		m.Changed = true
	}
	if n.Instructions != "" && m.Instructions != n.Instructions {
		m.Instructions = n.Instructions
		m.Changed = true
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

func (m *Message) AppendDistrict(districtID district.ID) {
	m.Cities = append(m.Cities, districtID)
}
