// Command genareas regenerates internal/area/data.gen.json from the live Pikud
// HaOref and Tzeva Adom datasets. Run it from the repo root:
//
//	go run ./internal/area/cmd/genareas
//
// It reconciles four sources by oref id: GetCitiesMix (authoritative shelter
// time + region), the per-language cities_*.json files (display names), and
// the Tzeva Adom cities.json (numeric id + coordinates).
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/phntom/goalert/internal/area"
	"github.com/phntom/goalert/internal/i18n"
)

const (
	citiesMixURL = "https://alerts-history.oref.org.il/Shared/Ajax/GetCitiesMix.aspx"
	tzevaCityURL = "https://www.tzevaadom.co.il/static/cities.json"
	cityLangURL  = "https://www.oref.org.il/districts/cities_%s.json"
)

// langFile maps our language codes to the oref cities_*.json suffixes.
var langFile = map[i18n.Language]string{"he": "heb", "en": "eng", "ru": "rus", "ar": "arb"}

type mixEntry struct {
	Label     string `json:"label"`
	LabelHe   string `json:"label_he"`
	ID        string `json:"id"`
	MigunTime int    `json:"migun_time"`
	Mixname   string `json:"mixname"`
}

type cityEntry struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type tzevaFile struct {
	Cities map[string]tzevaCity `json:"cities"`
}

type tzevaCity struct {
	ID        int     `json:"id"`
	He        string  `json:"he"`
	En        string  `json:"en"`
	Ru        string  `json:"ru"`
	Ar        string  `json:"ar"`
	Countdown int     `json:"countdown"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
}

func main() {
	out := flag.String("out", "internal/area/data.gen.json", "output path")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "genareas:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	var mix []mixEntry
	if err := fetchJSON(citiesMixURL, &mix); err != nil {
		return fmt.Errorf("GetCitiesMix: %w", err)
	}
	names, err := fetchLangNames()
	if err != nil {
		return err
	}
	var tz tzevaFile
	if err := fetchJSON(tzevaCityURL, &tz); err != nil {
		return fmt.Errorf("tzevaadom cities: %w", err)
	}
	tzByName := indexTzeva(tz)

	areas := build(mix, names, tzByName)
	sort.Slice(areas, func(i, j int) bool { return less(areas[i].ID, areas[j].ID) })

	blob, err := json.MarshalIndent(areas, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, append(blob, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %d areas to %s\n", len(areas), out)
	return nil
}

// fetchLangNames returns id -> {lang -> city name} from the per-language files.
func fetchLangNames() (map[string]map[i18n.Language]string, error) {
	names := make(map[string]map[i18n.Language]string, 2000)
	for lang, suffix := range langFile {
		var cities []cityEntry
		if err := fetchJSON(fmt.Sprintf(cityLangURL, suffix), &cities); err != nil {
			return nil, fmt.Errorf("cities_%s: %w", suffix, err)
		}
		for _, c := range cities {
			n := firstSegment(c.Label)
			if n == "" {
				continue
			}
			if names[c.ID] == nil {
				names[c.ID] = make(map[i18n.Language]string, 4)
			}
			names[c.ID][lang] = n
		}
	}
	return names, nil
}

func indexTzeva(tz tzevaFile) map[string]tzevaCity {
	idx := make(map[string]tzevaCity, len(tz.Cities))
	for _, c := range tz.Cities {
		if key := area.Normalize(c.He); key != "" {
			idx[key] = c
		}
	}
	return idx
}

func build(mix []mixEntry, names map[string]map[i18n.Language]string, tzByName map[string]tzevaCity) []*area.Area {
	byID := make(map[string]*area.Area, len(mix))
	for _, m := range mix {
		he := pick(m.LabelHe, m.Label)
		if he == "" || area.IsNationwide(he) || area.IsPartial(he) {
			continue
		}
		a := &area.Area{
			ID:        m.ID,
			Names:     map[i18n.Language]string{"he": he},
			MigunTime: m.MigunTime,
			Region:    regionFromMix(m.Mixname),
		}
		for lang, n := range names[m.ID] {
			if n != "" {
				a.Names[lang] = n
			}
		}
		enrichTzeva(a, he, tzByName)
		byID[m.ID] = a
	}
	addMissing(byID, names, tzByName)
	areas := make([]*area.Area, 0, len(byID))
	for _, a := range byID {
		areas = append(areas, a)
	}
	return areas
}

// addMissing adds ids that appear in the per-language files but not in
// GetCitiesMix, so we don't lose any settlement.
func addMissing(byID map[string]*area.Area, names map[string]map[i18n.Language]string, tzByName map[string]tzevaCity) {
	for id, langNames := range names {
		if _, ok := byID[id]; ok {
			continue
		}
		he := langNames["he"]
		if he == "" || area.IsNationwide(he) {
			continue
		}
		a := &area.Area{ID: id, Names: map[i18n.Language]string{}}
		for lang, n := range langNames {
			a.Names[lang] = n
		}
		enrichTzeva(a, he, tzByName)
		byID[id] = a
	}
}

// enrichTzeva fills coordinates, Tzeva id, missing translations and a shelter
// time fallback from the Tzeva Adom dataset, matched by normalized Hebrew name.
func enrichTzeva(a *area.Area, he string, tzByName map[string]tzevaCity) {
	c, ok := tzByName[area.Normalize(he)]
	if !ok {
		return
	}
	a.TzevaID, a.Lat, a.Lng = c.ID, c.Lat, c.Lng
	if a.MigunTime == 0 {
		a.MigunTime = c.Countdown
	}
	for lang, n := range map[i18n.Language]string{"en": c.En, "ru": c.Ru, "ar": c.Ar} {
		if a.Names[lang] == "" && n != "" {
			a.Names[lang] = n
		}
	}
	if alias := area.Normalize(c.He); alias != "" && alias != area.Normalize(he) {
		a.Aliases = append(a.Aliases, c.He)
	}
}

func regionFromMix(mix string) string {
	parts := strings.SplitN(mix, "|", 2)
	if len(parts) < 2 {
		return ""
	}
	r := strings.NewReplacer("<span>", "", "</span>", "").Replace(parts[1])
	return strings.TrimSpace(r)
}

func firstSegment(label string) string {
	if i := strings.Index(label, "|"); i >= 0 {
		return strings.TrimSpace(label[:i])
	}
	return strings.TrimSpace(label)
}

func pick(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

// less orders ids numerically when possible, lexically otherwise.
func less(a, b string) bool {
	ai, ae := strconv.Atoi(a)
	bi, be := strconv.Atoi(b)
	if ae == nil && be == nil {
		return ai < bi
	}
	return a < b
}

func fetchJSON(url string, dst any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://www.oref.org.il/")
	client := &http.Client{Timeout: 20 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	body = bytes.TrimPrefix(bytes.TrimLeft(body, "\x00"), []byte{0xEF, 0xBB, 0xBF})
	return json.Unmarshal(body, dst)
}
