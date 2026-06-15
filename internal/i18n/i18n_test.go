package i18n

import "testing"

func TestText(t *testing.T) {
	cases := []struct {
		id   string
		lang Language
		want string
	}{
		{"mention_prefix", "he", "צבעאדום"},
		{"mention_prefix", "en", "OrefAlarm"},
		{"message.rockets", "he", "ירי רקטות וטילים"},
		{"message.rockets", "en", "Rocket and missile fire"},
		{"message.all_clear", "en", "The event is over; you may leave the protected space"},
		{"message.pre_alert", "ru", "В ближайшие минуты в вашем районе ожидаются тревоги; приготовьтесь зайти в убежище"},
		{"subdivision.industrial_zone", "ar", "المنطقة الصناعية"},
	}
	for _, c := range cases {
		if got := Text(c.id, c.lang); got != c.want {
			t.Errorf("Text(%q,%q) = %q, want %q", c.id, c.lang, got, c.want)
		}
	}
}

func TestTextOrMissing(t *testing.T) {
	if got := TextOr("does.not.exist", "en", "fallback"); got != "fallback" {
		t.Errorf("TextOr missing = %q, want fallback", got)
	}
	if got := TextOr("message.rockets", "en", "fallback"); got != "Rocket and missile fire" {
		t.Errorf("TextOr present = %q, want the real string", got)
	}
}

func TestAllLanguagesHaveCoreKeys(t *testing.T) {
	keys := []string{
		"mention_prefix", "message.instructions", "message.lockdown",
		"message.uav_instructions", "message.pre_alert", "message.all_clear",
		"message.rockets", "message.uav", "message.infiltration",
		"message.secondsPrefix", "message.secondsSuffix", "message.immediate",
	}
	for _, lang := range Languages {
		for _, k := range keys {
			if got := TextOr(k, lang, ""); got == "" {
				t.Errorf("missing key %q for lang %q", k, lang)
			}
		}
	}
}
