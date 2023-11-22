package district

import (
	"fmt"
	"github.com/phntom/goalert/internal/config"
	"strings"
)

var replacerFull = strings.NewReplacer(
	" - ", " ",
	"'", "",
	"\"", "",
	",", "",
	" ", "_",
	"-", "",
)
var replacerHashtag = strings.NewReplacer(
	" ", "",
	"'", "",
	"\"", "",
	",", "",
	"-", "",
)

func CityNameCleanFull(title string) string {
	for {
		newTitle := replacerFull.Replace(title)
		if newTitle == title {
			return title
		}
		title = newTitle
	}
}

func CityNameCleanShort(title string) string {
	for {
		newTitle := replacerHashtag.Replace(title)
		if newTitle == title {
			return title
		}
		title = newTitle
	}
}

func CitiesToHashtagsMentionsLegacy(cities []ID, lang config.Language) (map[string][]string, []string, []string, []string) {
	result := make(map[string][]string)
	var hashtags []string
	var mentions []string
	var legacy []string
	districts := GetDistricts()
	for _, city := range cities {
		name := districts[lang][city].SettlementName
		hashtags = append(hashtags, fmt.Sprintf("#%s", CityNameCleanFull(name)))
		mentions = append(mentions, config.GetText("district.mention_prefix", lang)+CityNameCleanShort(name))
		legacy = append(legacy, name)
		cityName, subs := GetCity(city, lang)
		if len(subs) == 0 {
			subs = []string{cityName}
		}
		result[cityName] = append(result[cityName], subs...)
	}
	return result, hashtags, mentions, legacy
}
