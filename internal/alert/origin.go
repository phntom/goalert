package alert

import "strings"

// originKeywords maps Hebrew launch-origin mentions to canonical origin keys.
// Order matters only for overlapping substrings; these do not overlap.
var originKeywords = []struct{ keyword, origin string }{
	{"תימן", "yemen"},
	{"איראן", "iran"},
	{"אירן", "iran"},
	{"חיזבאללה", "lebanon"},
	{"לבנון", "lebanon"},
	{"רצועת עזה", "gaza"},
	{"עזה", "gaza"},
	{"סוריה", "syria"},
	{"עיראק", "iraq"},
}

// ExtractOrigin returns the canonical launch-origin key (e.g. "yemen") mentioned
// in Hebrew alert text, or "" if none is present. Origin names live under the
// i18n key "origin.<key>".
func ExtractOrigin(text string) string {
	for _, o := range originKeywords {
		if strings.Contains(text, o.keyword) {
			return o.origin
		}
	}
	return ""
}
