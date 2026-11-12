package sources

const (
	YnetURL      = "https://alerts.ynet.co.il/alertsRss/YnetPicodeHaorefAlertFiles.js?callback=jsonCallback"
	YnetReferrer = "https://www.ynet.co.il/"
	OrefURL      = "https://www.oref.org.il/WarningMessages/alert/alerts.json"
	OrefReferrer = "https://www.oref.org.il//12481-he/Pakar.aspx"
)

// instructionsFor returns the locale key of the safety instructions for an alert category.
func instructionsFor(category string) string {
	switch category {
	case "infiltration", "radiological", "biohazard":
		return "lockdown"
	case "uav":
		return "uav_instructions"
	default:
		return "instructions"
	}
}
