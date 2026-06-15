package area

import "strings"

// normalizer reconciles the minor spelling differences between sources: it
// collapses doubled yud, and strips hyphens, parentheses and Hebrew/ASCII
// quote marks. Whitespace is collapsed separately.
var normalizer = strings.NewReplacer(
	"יי", "י",
	"-", "",
	"(", "",
	")", "",
	"'", "",
	"\"", "",
	"״", "",
	"׳", "",
)

// Normalize returns a canonical lookup key for an area name so that ynet, oref
// and Tzeva Adom spellings of the same place resolve to one area.
func Normalize(name string) string {
	n := normalizer.Replace(name)
	return strings.Join(strings.Fields(n), " ")
}
