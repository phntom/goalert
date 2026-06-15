package publish

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/phntom/goalert/internal/area"
	"github.com/phntom/goalert/internal/i18n"
)

const (
	alertColor = "#CF1434"
	clearColor = "#2EB67D"
)

// Render builds the Mattermost post for an incident in one language. The post
// carries both the full trigger text (Message) and the clean card (Props);
// the engine later patches Message to "" so only the card remains.
func Render(inc *Incident, lang i18n.Language) *model.Post {
	title := i18n.Text(inc.category.TitleKey(), lang)
	if inc.origin != "" {
		if name := i18n.TextOr("origin."+inc.origin, lang, ""); name != "" {
			title = title + " · " + name
		}
	}
	if n := len(inc.ids); n > 1 {
		title = fmt.Sprintf("%s (%d)", title, n)
	}
	instructions := renderInstructions(inc, lang)
	fields, hashtags, mentions, legacy := cityViews(inc.areas, lang)

	text := strings.TrimSpace(strings.Join([]string{
		strings.Join(legacy, ", "),
		instructions,
		strings.TrimSpace(strings.Join(mentions, " ") + " " + strings.Join(hashtags, " ")),
	}, "\n"))
	fallback := strings.TrimSpace(fmt.Sprintf("%s %s %s",
		strings.Join(mentions, " "), strings.Join(legacy, ", "), instructions))

	color := alertColor
	if inc.category.IsEndAlert() {
		color = clearColor
	}

	return &model.Post{
		Message: text,
		Props: map[string]any{
			"attachments": []*model.SlackAttachment{{
				Title:    title,
				Text:     instructions,
				Fallback: fallback,
				Color:    color,
				Fields:   fields,
			}},
		},
		Metadata: &model.PostMetadata{
			Priority: &model.PostPriority{
				Priority:     model.NewString(inc.category.Urgency()),
				RequestedAck: model.NewBool(inc.safety >= 60),
			},
		},
	}
}

// renderInstructions fills the {1}{2}{3} shelter-time template.
func renderInstructions(inc *Incident, lang i18n.Language) string {
	tmpl := i18n.Text(inc.category.InstructionsKey(), lang)
	var r *strings.Replacer
	if inc.safety > 0 {
		r = strings.NewReplacer(
			"{1}", i18n.Text("message.secondsPrefix", lang),
			"{2}", strconv.Itoa(inc.safety),
			"{3}", i18n.Text("message.secondsSuffix", lang),
		)
	} else {
		r = strings.NewReplacer("{1}", "", "{2}", "", "{3}", i18n.Text("message.immediate", lang))
	}
	return strings.TrimSpace(r.Replace(tmpl))
}

// cityViews builds the card fields (sub-areas grouped under their parent city)
// plus the hashtag, mention and plain-name lists for the trigger text.
func cityViews(areas []*area.Area, lang i18n.Language) (fields []*model.SlackAttachmentField, hashtags, mentions, legacy []string) {
	prefix := i18n.Text("mention_prefix", lang)
	grouped := make(map[string][]string)
	var order []string
	for _, a := range areas {
		name := a.Name(lang)
		legacy = append(legacy, name)
		hashtags = append(hashtags, "#"+area.HashtagName(name))
		mentions = append(mentions, prefix+area.HashtagName(name))

		parent, sub := area.SplitName(name)
		if _, ok := grouped[parent]; !ok {
			order = append(order, parent)
		}
		if sub != "" {
			grouped[parent] = append(grouped[parent], sub)
		}
	}
	for _, p := range order {
		fields = append(fields, &model.SlackAttachmentField{
			Title: p,
			Value: strings.Join(grouped[p], "\n"),
			Short: true,
		})
	}
	return fields, hashtags, mentions, legacy
}
