package sources

import "testing"

func Test_instructionsFor(t *testing.T) {
	tests := []struct {
		category string
		want     string
	}{
		{"rockets", "instructions"},
		{"infiltration", "lockdown"},
		{"radiological", "lockdown"},
		{"biohazard", "lockdown"},
		{"uav", "uav_instructions"},
		{"", "instructions"},
	}
	for _, tt := range tests {
		if got := instructionsFor(tt.category); got != tt.want {
			t.Errorf("instructionsFor(%q) = %q, want %q", tt.category, got, tt.want)
		}
	}
}
