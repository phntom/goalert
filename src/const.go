package main

const (
	AlertsUrl = "https://source-alerts.ynet.co.il/alertsRss/YnetPicodeHaorefAlertFiles.js?callback=jsonCallback"
)

var /* const */ CitiesResponseTime = map[string]int{
	"אשדוד - יא,יב,טו,יז,מרינה":      45,
	"אשדוד - ח,ט,י,יג,יד,טז":         45,
	"אשדוד - ג,ו,ז":                  45,
	"אשדוד - אזור תעשייה צפוני ונמל": 45,
	"אשדוד - א,ב,ד,ה":                45,
	"קריית גת, כרמי גת":              45,
	"אופקים":                         45,
	"באר שבע - מערב":                 60,
	"באר שבע - מזרח":                 60,
	"באר שבע - צפון":                 60,
	"באר שבע - דרום":                 60,
	"לוד":                            90,
	"תל אביב - מרכז העיר":            90,
	"תל אביב - מזרח":                 90,
	"תל אביב - דרום העיר ויפו":       90,
	"תל אביב - עבר הירקון":           90,
	"רמת גן - מזרח":                  90,
	"רמת גן - מערב":                  90,
	"ראש העין":                       90,
	"נס ציונה":                       90,
	"ראשון לציון - מערב":             90,
	"ראשון לציון - מזרח":             90,
	"בת-ים":                          90,
	"חולון":                          90,
	"גבעתיים":                        90,
	"פתח תקווה":                      90,
	"קריית מוצקין":                   60,
	"קריית ים":                       60,
	"קריית ביאליק":                   60,
	"עכו - אזור תעשייה":              30,
	"עכו":                            30,
	"חיפה - כרמל ועיר תחתית":         60,
	"חיפה - מערב":                    60,
	"חיפה - נווה שאנן ורמות כרמל":    60,
	"חיפה - מפרץ והקריות":            60,
}
