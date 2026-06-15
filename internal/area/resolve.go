package area

import "github.com/phntom/goalert/internal/i18n"

// Nationwide is the synthetic area used when a source reports a country-wide
// alert, so it renders as a single message instead of ~1500 city posts.
var Nationwide = &Area{
	ID: "nationwide",
	Names: map[i18n.Language]string{
		"he": "כל הארץ", "en": "Nationwide", "ru": "Вся страна", "ar": "كل البلاد",
	},
}

var nationwideAliases = normalizedSet("כל הארץ", "ברחבי הארץ", "כל המדינה", "בכל הארץ")

var partialAliases = normalizedSet("בחלק מהאזורים בארץ", "חלק מהאזורים")

func normalizedSet(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[Normalize(n)] = true
	}
	return m
}

// IsNationwide reports whether raw is a country-wide aggregate token.
func IsNationwide(raw string) bool { return nationwideAliases[Normalize(raw)] }

// IsPartial reports whether raw is the un-localizable "some areas" token.
func IsPartial(raw string) bool { return partialAliases[Normalize(raw)] }

// Resolve maps a raw source area name to canonical areas. It de-aggregates the
// country-wide token to the single Nationwide area. ok is false for partial or
// unknown names (the caller decides how to log them).
func (s *Set) Resolve(raw string) (areas []*Area, ok bool) {
	if IsNationwide(raw) {
		return []*Area{Nationwide}, true
	}
	if IsPartial(raw) {
		return nil, false
	}
	if a, found := s.ByName(raw); found {
		return []*Area{a}, true
	}
	return nil, false
}
