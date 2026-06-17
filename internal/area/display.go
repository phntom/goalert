package area

import "strings"

// SplitName separates a settlement display name into a parent city and an
// optional sub-area, e.g. "Tel Aviv - East" -> ("Tel Aviv", "East"). Names
// without the " - " separator are returned unchanged with an empty sub.
func SplitName(name string) (parent, sub string) {
	const sep = " - "
	if i := strings.Index(name, sep); i >= 0 {
		return strings.TrimSpace(name[:i]), strings.TrimSpace(name[i+len(sep):])
	}
	return name, ""
}

var hashtagCleaner = strings.NewReplacer(
	" ", "", "'", "", "\"", "", ",", "", "-", "", "(", "", ")", "", "״", "", "׳", "",
)

// HashtagName turns a display name into a hashtag/mention-safe token with no
// spaces or punctuation: "Tel Aviv - East" -> "TelAvivEast".
func HashtagName(name string) string {
	return hashtagCleaner.Replace(name)
}
