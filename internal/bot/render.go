package bot

import (
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/phntom/goalert/internal/config"
	"github.com/phntom/goalert/internal/district"
	"strconv"
	"strings"
	"unicode"
)

func Render(msg *Message, lang config.Language) *model.Post {
	ack := msg.SafetySeconds >= 60
	cities, hashtags, mentions, legacy := district.CitiesToHashtagsMentionsLegacy(msg.Cities, lang)
	title := ""
	if msg.Category != "" {
		title = config.GetText(fmt.Sprintf("message.%s", msg.Category), lang)
		if len(msg.RocketIDs) > 1 {
			title = fmt.Sprintf("%s (%d)", title, len(msg.RocketIDs))
		}
	}
	secondsTag := "message.seconds"
	if msg.SafetySeconds == 0 {
		secondsTag = "message.immediate"
	}
	secondsReplacer := strings.NewReplacer(
		"{1}", config.GetText(secondsTag+"Prefix", lang),
		"{2}", strconv.Itoa(int(msg.SafetySeconds)),
		"{3}", config.GetText(secondsTag+"Suffix", lang),
	)
	instructions := secondsReplacer.Replace(config.GetText(fmt.Sprintf("message.%s", msg.Instructions), lang))
	fields := CitiesToFields(cities)
	text := fmt.Sprintf("%s\n%s\n%s %s",
		strings.Join(legacy, ", "),
		instructions,
		strings.Join(mentions, " "),
		strings.Join(hashtags, " "),
	)
	legacyStr := fmt.Sprintf("%s %s %s",
		strings.Join(mentions, " "),
		strings.Join(legacy, ", "),
		instructions,
	)
	replyToId := ""
	urgent := "urgent"
	if msg.Category == "lockdown" || msg.Category == "biohazard" {
		urgent = "important"
	}
	//goland:noinspection GoDeprecation
	return &model.Post{
		Message: text,
		RootId:  replyToId,
		Props: map[string]any{
			"attachments": []*model.SlackAttachment{
				{
					Title:    title,
					Text:     instructions,
					Fallback: legacyStr,
					Color:    "#CF1434",
					Fields:   fields,
				},
			},
		},
		Metadata: &model.PostMetadata{
			Priority: &model.PostPriority{
				Priority:     model.NewString(urgent),
				RequestedAck: model.NewBool(ack),
			},
		},
	}
}

func CitiesToFields(cities map[string][]string) []*model.SlackAttachmentField {
	var fields []*model.SlackAttachmentField
	for n1, n2 := range cities {
		value := strings.Join(n2, "\n")
		if len(n2) == 1 && n1 == n2[0] {
			value = ""
			fields = append(fields, &model.SlackAttachmentField{
				Title: n1,
				Value: value,
				Short: true,
			})
		} else {
			fields = append([]*model.SlackAttachmentField{{
				Title: n1,
				Value: value,
				Short: true,
			}}, fields...)
		}

	}
	return fields
}

func ChannelToLanguage(channel *model.Channel) config.Language {
	displayName := channel.DisplayName
	for _, r := range displayName {
		// Check for Hebrew characters
		if unicode.In(r, unicode.Hebrew) {
			return "he"
		}
		// Check for Arabic characters
		if unicode.In(r, unicode.Arabic) {
			return "ar"
		}
		// Check for Cyrillic (Russian) characters
		if unicode.In(r, unicode.Cyrillic) {
			return "ru"
		}
	}
	return "en"
}
