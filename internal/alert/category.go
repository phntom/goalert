package alert

import "strings"

// Category is the canonical Pikud HaOref "history" category number. Every source
// is translated into this space so the rest of the bot is source-agnostic.
type Category int

// Canonical categories (subset we render). Values match oref history categories.
const (
	CatUnknown         Category = 0
	CatMissile         Category = 1
	CatUAV             Category = 2
	CatNonConventional Category = 3
	CatWarning         Category = 4
	CatEarthquake      Category = 7
	CatCBRNE           Category = 9
	CatTerror          Category = 10
	CatTsunami         Category = 11
	CatHazmat          Category = 12
	CatEndAlert        Category = 13 // "update" — the all-clear
	CatPreAlert        Category = 14 // "flash" — early warning
)

type catInfo struct {
	name    string // i18n key suffix under "message."
	emoji   string
	isAlert bool // true for an active-siren category
}

// catMeta is the canonical category metadata, mirroring oref_alert categories.py.
var catMeta = map[Category]catInfo{
	CatMissile:         {"rockets", "🚀", true},
	CatUAV:             {"uav", "✈️", true},
	CatNonConventional: {"nonconventional", "☢️", true},
	CatWarning:         {"warning", "🚨", true},
	7:                  {"earthquake", "🌍", true},
	8:                  {"earthquake", "🌍", true},
	CatCBRNE:           {"cbrne", "☢️", true},
	CatTerror:          {"infiltration", "⚔️", true},
	CatTsunami:         {"tsunami", "🌊", true},
	CatHazmat:          {"hazmat", "☣️", true},
	CatEndAlert:        {"all_clear", "✅", false},
	CatPreAlert:        {"pre_alert", "⚡", false},
}

// Emoji returns the reaction emoji glyph for a category, or "" if unknown.
func (c Category) Emoji() string { return catMeta[c].emoji }

// IsAlert reports whether the category is an active siren (not pre-alert,
// all-clear, memorial siren, or drill).
func (c Category) IsAlert() bool { return catMeta[c].isAlert }

// IsEndAlert reports the all-clear category.
func (c Category) IsEndAlert() bool { return c == CatEndAlert }

// IsPreAlert reports the early-warning category.
func (c Category) IsPreAlert() bool { return c == CatPreAlert }

// Known reports whether the category is one we render.
func (c Category) Known() bool { _, ok := catMeta[c]; return ok }

// TitleKey is the i18n message id for the category's display title.
func (c Category) TitleKey() string {
	if m, ok := catMeta[c]; ok {
		return "message." + m.name
	}
	return ""
}

// InstructionsKey is the i18n message id for the safety instructions to show.
func (c Category) InstructionsKey() string {
	switch c {
	case CatEndAlert:
		return "message.all_clear"
	case CatPreAlert:
		return "message.pre_alert"
	case CatUAV:
		return "message.uav_instructions"
	case CatNonConventional, CatCBRNE, CatHazmat, CatTerror:
		return "message.lockdown"
	default:
		return "message.instructions"
	}
}

// Urgency maps to a Mattermost post priority ("urgent", "important", or "").
func (c Category) Urgency() string {
	switch c {
	case CatEndAlert:
		return ""
	case CatNonConventional, CatCBRNE, CatHazmat, CatTerror:
		return "important"
	default:
		return "urgent"
	}
}

// FromOrefHistory maps an oref history category int into a Category.
func FromOrefHistory(cat int) Category {
	c := Category(cat)
	if c.Known() {
		return c
	}
	return CatUnknown
}

// tzevaadomThreat maps a Tzeva Adom WebSocket threat id to a history category
// (verbatim from oref_alert categories.py).
var tzevaadomThreat = map[int]Category{
	0: CatMissile, 1: CatHazmat, 2: CatTerror, 3: CatEarthquake,
	4: CatTsunami, 5: CatUAV, 6: CatCBRNE, 7: CatNonConventional, 8: CatWarning,
}

// FromTzevaadomThreat maps a Tzeva Adom threat id to a Category. ok is false for
// unmapped threats, which callers should drop.
func FromTzevaadomThreat(threat int) (c Category, ok bool) {
	c, ok = tzevaadomThreat[threat]
	return c, ok
}

// IsHebrewDrill reports whether ynet/source text describes a drill (תרגיל),
// which must not be published.
func IsHebrewDrill(text string) bool {
	return strings.Contains(text, "תרגיל")
}

// FromYnetDescription classifies a ynet alert from its Hebrew description text.
// Returns CatUnknown when nothing matches (caller decides whether to drop).
func FromYnetDescription(desc string) Category {
	switch {
	case strings.Contains(desc, "אלא אם ניתנה התרעה נוספת"):
		return CatUAV
	case strings.Contains(desc, "נעלו"):
		return CatTerror
	case strings.Contains(desc, "היכנסו למרחב המוגן"):
		return CatMissile
	default:
		return CatUnknown
	}
}
