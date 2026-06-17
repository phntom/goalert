package area

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"תל אביב  -  יפו": "תל אביב יפו",
		"קריית שמונה":     "קרית שמונה", // doubled yud collapsed
		"כפר סבא (מערב)":  "כפר סבא מערב",
		"  Tel  Aviv  ":   "Tel Aviv",
		"באר-שבע":         "בארשבע",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q want %q", in, got, want)
		}
	}
}

func TestSplitName(t *testing.T) {
	p, s := SplitName("Tel Aviv - East")
	if p != "Tel Aviv" || s != "East" {
		t.Errorf("SplitName = %q,%q", p, s)
	}
	p, s = SplitName("Ashkelon")
	if p != "Ashkelon" || s != "" {
		t.Errorf("SplitName no-sep = %q,%q", p, s)
	}
}

func TestHashtagName(t *testing.T) {
	if got := HashtagName("Tel Aviv - East"); got != "TelAvivEast" {
		t.Errorf("HashtagName = %q", got)
	}
}

func TestDefaultDatasetLoads(t *testing.T) {
	s := Default()
	if len(s.All()) < 1000 {
		t.Fatalf("expected a populated dataset, got %d areas", len(s.All()))
	}
	for _, a := range s.All() {
		if a.ID == "" || a.Names["he"] == "" {
			t.Fatalf("area missing id/he name: %+v", a)
			break
		}
	}
}

func TestLookupsAndReconciliation(t *testing.T) {
	s := Default()
	a, ok := s.ByName("תל אביב - יפו")
	if !ok {
		t.Fatal("expected to resolve Tel Aviv - Yafo")
	}
	if a.Region == "" || a.MigunTime == 0 {
		t.Errorf("area lacks region/migun: %+v", a)
	}
	// Spelling variation (extra spaces) must resolve to the same area.
	if b, ok := s.ByName("תל אביב  -  יפו"); !ok || b.ID != a.ID {
		t.Errorf("spacing variant did not reconcile to same area")
	}
	if a.TzevaID != 0 {
		if c, ok := s.ByTzevaID(a.TzevaID); !ok || c.ID != a.ID {
			t.Errorf("ByTzevaID(%d) did not round-trip", a.TzevaID)
		}
	}
}

func TestResolveDeaggregation(t *testing.T) {
	s := Default()
	if got, ok := s.Resolve("כל הארץ"); !ok || len(got) != 1 || got[0] != Nationwide {
		t.Errorf("nationwide resolve = %v,%v", got, ok)
	}
	if !IsNationwide("ברחבי הארץ") {
		t.Error("expected ברחבי הארץ to be nationwide")
	}
	if _, ok := s.Resolve("בחלק מהאזורים בארץ"); ok {
		t.Error("partial token should not resolve")
	}
	if _, ok := s.Resolve("עיר שלא קיימת בכלל"); ok {
		t.Error("unknown city should not resolve")
	}
}
