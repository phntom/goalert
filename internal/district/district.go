package district

import (
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/config"
	"io"
	"sync"
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

////go:embed districts_en.json
//var districtsEnSource string
//
////go:embed districts_he.json
//var districtsHeSource string
//
////go:embed districts_ru.json
//var districtsRuSource string
//
////go:embed districts_ar.json
//var districtsArSource string
// Map of available languages to their corresponding JSON source strings.
//var languages = map[config.Language]string{
//	"en": districtsEnSource,
//	"he": districtsHeSource,
//	"ru": districtsRuSource,
//	"ar": districtsArSource,
//}

// districts holds the unmarshaled JSON data. We use sync.Once to ensure the map is only initialized once.
var (
	districts      Districts
	districtLookup map[string]ID
	once           sync.Once
)

// initDistricts is called once to initialize the districts map.
func initDistricts() {
	numberOfLanguages := len(config.Languages)
	districts = make(Districts, numberOfLanguages)
	for _, lang := range config.Languages {
		var districtList []District

		filename := fmt.Sprintf("districts.%s.json", lang)
		file, err := districtFS.Open(filename)
		if err != nil {
			mlog.Error("Failed loading district", mlog.Err(err),
				mlog.Any("lang", lang),
				mlog.Any("filename", filename),
			)
			return
		}
		content, err := io.ReadAll(file)
		if err != nil {
			mlog.Error("Failed reading district file", mlog.Err(err),
				mlog.Any("lang", lang),
				mlog.Any("filename", filename),
			)
			return
		}
		err = json.Unmarshal(content, &districtList)
		if err != nil {
			mlog.Error("Failed unmarshaling district file", mlog.Err(err),
				mlog.Any("lang", lang),
				mlog.Any("filename", filename),
			)
			return
		}

		d := make(map[ID]District, len(districtList))
		if districtLookup == nil {
			districtLookup = make(map[string]ID)
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
				mlog.Warn("Duplicate district ID", mlog.Any("district", district), mlog.String("newID", string(district.ID)))
			}
			d[district.ID] = district
			districtLookup[district.SettlementName] = district.ID
		}

		districts[lang] = d
	}
}

func GetDistricts() Districts {
	once.Do(initDistricts)
	return districts
}

func GetDistrictByCity(city string) ID {
	return districtLookup[city]
}
