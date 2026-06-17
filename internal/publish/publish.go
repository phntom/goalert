// Package publish turns resolved alerts into Mattermost messages and owns the
// signature latency trick: post the full trigger text (with @mentions and
// #hashtags, so members get an instant push) and then, ~200ms later, patch the
// post down to a clean UI card. It also tracks live incidents for per-area
// dedup, patch-on-new-info, end-of-alert, and expiry.
package publish

import (
	"unicode"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/phntom/goalert/internal/i18n"
)

// Channel is a destination channel and the language to render it in.
type Channel struct {
	ID   string
	Name string
	Lang i18n.Language
}

// ChatClient is the Mattermost boundary the engine depends on; the real
// implementation is Mattermost, and tests substitute a fake.
type ChatClient interface {
	Channels() []Channel
	CreatePost(channelID string, post *model.Post) (postID string, err error)
	PatchPost(postID string, props map[string]any) error
	AddReaction(postID, emoji string) error
}

// LanguageOf picks a language from a channel display name by the script of its
// first Hebrew/Arabic/Cyrillic letter, defaulting to English.
func LanguageOf(displayName string) i18n.Language {
	sets := map[i18n.Language]*unicode.RangeTable{
		"he": unicode.Hebrew, "ar": unicode.Arabic, "ru": unicode.Cyrillic,
	}
	for _, r := range displayName {
		for lang, set := range sets {
			if unicode.In(r, set) {
				return lang
			}
		}
	}
	return "en"
}
