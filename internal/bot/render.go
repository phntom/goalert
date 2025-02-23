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
	secondsReplacer := strings.NewReplacer(
		"{1}", "",
		"{2}", "",
		"{3}", config.GetText("message.immediate", lang),
	)
	if msg.SafetySeconds > 0 {
		secondsTag := "message.seconds"
		secondsReplacer = strings.NewReplacer(
			"{1}", config.GetText(secondsTag+"Prefix", lang),
			"{2}", strconv.Itoa(int(msg.SafetySeconds)),
			"{3}", config.GetText(secondsTag+"Suffix", lang),
		)
	}
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
	} else if msg.Instructions == "uav_event_over" {
		urgent = ""
		text = ""
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
	fields := make([]*model.SlackAttachmentField, 0, len(cities))
	for n1, n2 := range cities {
		value := strings.Join(n2, "\n")
		if len(n2) == 1 && n1 == n2[0] {
			value = ""
		}
		fields = append(fields, &model.SlackAttachmentField{
			Title: n1,
			Value: value,
			Short: true,
		})
	}
	return fields
}

func ChannelToLanguage(channel *model.Channel) config.Language {
	characterSets := map[config.Language]*unicode.RangeTable{
		"he": unicode.Hebrew,
		"ar": unicode.Arabic,
		"ru": unicode.Cyrillic,
	}

	for _, r := range channel.DisplayName {
		for lang, set := range characterSets {
			if unicode.In(r, set) {
				return lang
			}
		}
	}
	return "en"
}
