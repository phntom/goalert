package alert

import "testing"

func TestExtractOrigin(t *testing.T) {
	cases := map[string]string{
		"שוגרו טילים מתימן לעבר ישראל": "yemen",
		"ירי רקטות מרצועת עזה":         "gaza",
		"זוהה שיגור מאיראן":            "iran",
		"ירי מלבנון":                   "lebanon",
		"חיזבאללה שיגר לעבר הצפון":     "lebanon",
		"ירי רקטות וטילים":             "",
	}
	for txt, want := range cases {
		if got := ExtractOrigin(txt); got != want {
			t.Errorf("ExtractOrigin(%q) = %q want %q", txt, got, want)
		}
	}
}
