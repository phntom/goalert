package district

import (
	"fmt"
	"github.com/phntom/goalert/internal/config"
	"strings"
	"unicode"
)

var SubdivisionsSet = map[ID]ID{
	"6004": "6004",
	"6005": "6004",
	"6006": "6004",
	"6007": "6004",
	"6008": "1310",
	"1310": "1310",
	"6009": "1310",
	"6027": "6027",
	"6028": "6027",
	"6029": "6029",
	"6030": "6029",
	"6031": "6029",
	"6032": "6029",
	"272":  "751",
	"751":  "751",
	"910":  "910",
	"60":   "910",
	"1047": "1059",
	"1059": "1059",
	"6025": "6025",
	"6026": "6025",
	"6038": "6018",
	"6022": "6018",
	"6019": "6018",
	"6036": "6018",
	"6020": "6018",
	"6018": "6018",
	"6021": "6018",
	"6000": "6000",
	"6001": "6000",
	"6002": "6000",
	"1386": "6000",
	"6003": "6000",
	"159":  "159",
	"1359": "159",
	"1052": "1052",
	"1053": "1052",
	"6012": "6010",
	"6010": "6010",
	"6013": "6010",
	"6011": "6010",
	"6014": "6014",
	"6016": "6014",
	"6034": "6014",
	"6017": "6014",
	"6015": "6014",
	"6023": "6023",
	"6024": "6023",
	"483":  "483",
	"53":   "483",
	"1117": "1117",
	"1116": "1117",
	"369":  "369",
	"49":   "369",
	"1017": "1016",
	"1016": "1016",
	"6039": "6037",
	"6037": "6037",
	"68":   "6037",
	"50":   "6037",
	"717":  "717",
	"718":  "717",
	"1357": "1357",
	"1379": "1357",
	"216":  "248",
	"248":  "248",
}

func capitalizeUnicode(s string) string {
	for _, v := range s {
		// Convert first character to uppercase
		return string(unicode.ToUpper(v)) + s[len(string(v)):]
	}
	return ""
}

func GetCity(id ID, lang config.Language) (string, []string) {
	subsetId := SubdivisionsSet[id]
	if subsetId == "" {
		subsetId = id
	}
	lid := fmt.Sprintf("subdivision.replace.%s", id)
	n1 := config.GetTextOptional(lid, lang, districts[lang][subsetId].SettlementName)
	n2 := config.GetTextOptional(lid, lang, districts[lang][id].SettlementName)
	if strings.Contains(n1, " - ") {
		n1 = strings.Trim(strings.Split(n1, " - ")[0], " -")
	} else if strings.Contains(n1, ", ") {
		spl := strings.Split(n1, ", ")
		var result []string
		for _, s := range spl {
			result = append(result, capitalizeUnicode(s))
		}
		return capitalizeUnicode(spl[0]), result
	}
	if strings.Contains(n1, config.GetText("subdivision.industrial_zone", lang)) {
		n2 = config.GetText("subdivision.industrial_zone", lang)
		n1 = strings.Trim(strings.Replace(n1, n2, "", 1), " ")
	} else if strings.Contains(strings.ToLower(n1), strings.ToLower(config.GetTextOptional("subdivision.industrial_zone_alt", lang, "@@@"))) {
		n2 = config.GetText("subdivision.industrial_zone", lang)
		n1 = strings.Trim(strings.Replace(n1, config.GetText("subdivision.industrial_zone_alt", lang), "", 1), " ")
		n1 = strings.Trim(strings.Replace(n1, config.GetTextOptional("subdivision.industrial_zone_alt2", lang, ""), "", 1), " ")
	} else if strings.Contains(n1, config.GetText("subdivision.regional_center", lang)) {
		n2 = config.GetText("subdivision.regional_center", lang)
		n1 = strings.Trim(strings.Replace(n1, n2, "", 1), " ")
	} else if strings.Contains(n1, config.GetText("subdivision.regional_council", lang)) {
		n2 = config.GetText("subdivision.regional_council", lang)
		n1 = strings.Trim(strings.Replace(n1, n2, "", 1), " ")
	} else {
		n2 = strings.Trim(strings.Replace(n2, n1, "", 1), " -")
	}
	if n2 == "" {
		return capitalizeUnicode(n1), []string{}
	}
	return capitalizeUnicode(n1), []string{capitalizeUnicode(n2)}
}
