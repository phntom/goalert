package district

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/phntom/goalert/internal/config"
)

type ID string

// District represents the JSON structure of each district.
type District struct {
	SettlementName       string `json:"label"`
	Value                string `json:"value"`
	ID                   ID     `json:"id"`
	AreaID               int    `json:"areaid"`
	AreaName             string `json:"areaname"`
	SettlementNameHebrew string `json:"label_he"`
	SafetyBufferSeconds  int    `json:"migun_time"`
}

// Districts is a map where each key is a language and each value is a map from district IDs to Districts.
type Districts map[config.Language]map[ID]District

// Embed the JSON source files.

// download new versions from https://www.oref.org.il/Shared/Ajax/GetDistricts.aspx?lang=he
//
//go:embed districts.*.json
var districtFS embed.FS

// districts holds the unmarshaled JSON data. We use sync.Once to ensure the map is only initialized once.
var (
	districts      Districts
	districtLookup map[string]ID
	once           sync.Once
)

// normalizeCityName performs several normalization steps on a city name.
func normalizeCityName(name string) string {
	// Replace יי with י and remove hyphens, parentheses, single quotes, double quotes
	replacer := strings.NewReplacer(
		"יי", "י",
		"-", "",
		"(", "",
		")", "",
		"'", "",
		"\"", "",
	)
	normalized := replacer.Replace(name)

	// Replace multiple consecutive spaces with a single space
	fields := strings.Fields(normalized)
	normalized = strings.Join(fields, " ")

	// Trim leading and trailing whitespace
	normalized = strings.TrimSpace(normalized)

	return normalized
}

// initDistricts is called once to initialize the districts map.
func initDistricts() {
	numberOfLanguages := len(config.Languages)
	districts = make(Districts, numberOfLanguages)
	for _, lang := range config.Languages {
		var districtList []District

		filename := fmt.Sprintf("districts.%s.json", lang)
		file, err := districtFS.Open(filename)
		if err != nil {
			slog.Error("Failed loading district", "error", err,
				"lang", lang,
				"filename", filename,
			)
			return
		}
		content, err := io.ReadAll(file)
		if err != nil {
			slog.Error("Failed reading district file", "error", err,
				"lang", lang,
				"filename", filename,
			)
			return
		}
		err = json.Unmarshal(content, &districtList)
		if err != nil {
			slog.Error("Failed unmarshaling district file", "error", err,
				"lang", lang,
				"filename", filename,
			)
			return
		}

		d := make(map[ID]District, len(districtList))
		if districtLookup == nil {
			districtLookup = make(map[string]ID, len(districtList)*numberOfLanguages)
		}

		for _, district := range districtList {
			if _, exists := d[district.ID]; exists {
				i := 2
				for {
					newID := ID(fmt.Sprintf("%s_%d", district.ID, i))
					if _, exists := d[newID]; !exists {
						district.ID = newID
						break
					}
					i++
				}
				slog.Warn("Duplicate district ID", "district", district, "newID", string(district.ID))
			}
			d[district.ID] = district
			// Use the normalized city name as the lookup key so variations
			// in hyphenation, quoting and spelling resolve to the district.
			normalizedSettlementName := normalizeCityName(district.SettlementName)
			if normalizedSettlementName != "" {
				districtLookup[normalizedSettlementName] = district.ID
			}
		}

		districts[lang] = d
	}
}

func GetDistricts() Districts {
	once.Do(initDistricts)
	return districts
}

func GetDistrictByCity(city string) ID {
	once.Do(initDistricts)
	return districtLookup[normalizeCityName(city)]
}
