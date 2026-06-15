package alert

import "testing"

func TestFromTzevaadomThreat(t *testing.T) {
	want := map[int]Category{
		0: CatMissile, 1: CatHazmat, 2: CatTerror, 3: CatEarthquake,
		4: CatTsunami, 5: CatUAV, 6: CatCBRNE, 7: CatNonConventional, 8: CatWarning,
	}
	for threat, exp := range want {
		got, ok := FromTzevaadomThreat(threat)
		if !ok || got != exp {
			t.Errorf("FromTzevaadomThreat(%d) = %d,%v want %d,true", threat, got, ok, exp)
		}
	}
	if _, ok := FromTzevaadomThreat(99); ok {
		t.Error("FromTzevaadomThreat(99) should be unmapped")
	}
}

func TestFromOrefHistory(t *testing.T) {
	cases := map[int]Category{
		1: CatMissile, 13: CatEndAlert, 14: CatPreAlert, 10: CatTerror,
		8: 8, // earthquake variant stays known
		5: CatUnknown, 6: CatUnknown, 99: CatUnknown,
	}
	for in, exp := range cases {
		if got := FromOrefHistory(in); got != exp {
			t.Errorf("FromOrefHistory(%d) = %d want %d", in, got, exp)
		}
	}
}

func TestFromYnetDescription(t *testing.T) {
	cases := []struct {
		desc string
		want Category
	}{
		{"היכנסו למרחב המוגן ושהו בו 10 דקות", CatMissile},
		{"היכנסו למרחב המוגן ושהו בו שעה אלא אם ניתנה התרעה נוספת", CatUAV},
		{"היכנסו למבנה נעלו את הדלתות", CatTerror},
		{"משהו אחר לגמרי", CatUnknown},
	}
	for _, c := range cases {
		if got := FromYnetDescription(c.desc); got != c.want {
			t.Errorf("FromYnetDescription(%q) = %d want %d", c.desc, got, c.want)
		}
	}
}

func TestIsHebrewDrill(t *testing.T) {
	if !IsHebrewDrill("אזעקה במסגרת תרגיל") {
		t.Error("expected drill detection")
	}
	if IsHebrewDrill("ירי רקטות וטילים") {
		t.Error("false positive drill")
	}
}

func TestCategorySemantics(t *testing.T) {
	if !CatMissile.IsAlert() || CatEndAlert.IsAlert() || CatPreAlert.IsAlert() {
		t.Error("IsAlert classification wrong")
	}
	if !CatEndAlert.IsEndAlert() || !CatPreAlert.IsPreAlert() {
		t.Error("end/pre classification wrong")
	}
	if CatTerror.InstructionsKey() != "message.lockdown" {
		t.Errorf("terror instructions = %q", CatTerror.InstructionsKey())
	}
	if CatUAV.InstructionsKey() != "message.uav_instructions" {
		t.Errorf("uav instructions = %q", CatUAV.InstructionsKey())
	}
	if CatMissile.InstructionsKey() != "message.instructions" {
		t.Errorf("missile instructions = %q", CatMissile.InstructionsKey())
	}
	if CatEndAlert.Urgency() != "" || CatTerror.Urgency() != "important" || CatMissile.Urgency() != "urgent" {
		t.Error("urgency mapping wrong")
	}
	if CatMissile.TitleKey() != "message.rockets" {
		t.Errorf("missile title = %q", CatMissile.TitleKey())
	}
	if CatMissile.Emoji() != "🚀" {
		t.Errorf("missile emoji = %q", CatMissile.Emoji())
	}
}

func TestKindOf(t *testing.T) {
	if KindOf(CatEndAlert) != KindEnd || KindOf(CatPreAlert) != KindPreAlert || KindOf(CatMissile) != KindAlert {
		t.Error("KindOf mapping wrong")
	}
}
