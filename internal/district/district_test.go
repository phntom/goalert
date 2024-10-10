package district

import (
	"github.com/phntom/goalert/internal/config"
	"sync"
	"testing"
)

func TestInitDistricts(t *testing.T) {
	// Reset the once variable to allow reinitialization
	once = sync.Once{}
	initDistricts()

	if len(districts) == 0 {
		t.Errorf("Expected districts to be initialized, but it was empty")
	}

	for _, lang := range config.Languages {
		if _, ok := districts[lang]; !ok {
			t.Errorf("Expected districts to contain language %s, but it was missing", lang)
		}
	}
}

func TestGetDistricts(t *testing.T) {
	// Reset the once variable to allow reinitialization
	once = sync.Once{}
	districts := GetDistricts()

	if len(districts) == 0 {
		t.Errorf("Expected districts to be initialized, but it was empty")
	}

	for _, lang := range config.Languages {
		if _, ok := districts[lang]; !ok {
			t.Errorf("Expected districts to contain language %s, but it was missing", lang)
		}
	}
}

func TestGetDistrictByCity(t *testing.T) {
	// Reset the once variable to allow reinitialization
	once = sync.Once{}
	initDistricts()

	tests := []struct {
		city string
		want ID
	}{
		{"Tel Aviv - City Center", "6031"},
		{"Haifa - West", "6015"},
		{"Givat Ze'ev", "296"},
	}

	for _, tt := range tests {
		t.Run(tt.city, func(t *testing.T) {
			got := GetDistrictByCity(tt.city)
			if got != tt.want {
				t.Errorf("GetDistrictByCity(%s) = %v, want %v", tt.city, got, tt.want)
			}
		})
	}
}
