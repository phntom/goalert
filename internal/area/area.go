// Package area holds the Pikud HaOref alert-zone dataset and the lookups that
// turn a source's raw (and slightly inconsistent) area name into a canonical
// Area carrying display names, shelter time, region and Tzeva Adom id.
package area

import (
	_ "embed"
	"encoding/json"
	"sync"

	"github.com/phntom/goalert/internal/i18n"
)

// Area is a single alert zone (a settlement or a labelled sub-area of a city).
type Area struct {
	ID        string                   `json:"id"`
	Names     map[i18n.Language]string `json:"names"`
	MigunTime int                      `json:"migun_time"`
	Region    string                   `json:"region"`
	TzevaID   int                      `json:"tzeva_id,omitempty"`
	Lat       float64                  `json:"lat,omitempty"`
	Lng       float64                  `json:"lng,omitempty"`
	Aliases   []string                 `json:"aliases,omitempty"`
}

// Name returns the display name in lang, falling back to Hebrew then any name.
func (a *Area) Name(lang i18n.Language) string {
	if n := a.Names[lang]; n != "" {
		return n
	}
	if n := a.Names["he"]; n != "" {
		return n
	}
	for _, n := range a.Names {
		if n != "" {
			return n
		}
	}
	return a.ID
}

// Set is an indexed, read-only collection of areas.
type Set struct {
	all     []*Area
	byID    map[string]*Area
	byName  map[string]*Area
	byTzeva map[int]*Area
}

//go:embed data.gen.json
var dataJSON []byte

var defaultSet = sync.OnceValue(func() *Set {
	var areas []*Area
	if err := json.Unmarshal(dataJSON, &areas); err != nil {
		panic("area: parsing embedded data.gen.json: " + err.Error())
	}
	return NewSet(areas)
})

// Default returns the embedded dataset, parsed once.
func Default() *Set { return defaultSet() }

// NewSet builds the lookup indexes over areas.
func NewSet(areas []*Area) *Set {
	s := &Set{
		all:     areas,
		byID:    make(map[string]*Area, len(areas)),
		byName:  make(map[string]*Area, len(areas)*3),
		byTzeva: make(map[int]*Area, len(areas)),
	}
	for _, a := range areas {
		s.byID[a.ID] = a
		if a.TzevaID != 0 {
			s.byTzeva[a.TzevaID] = a
		}
		for _, n := range a.Names {
			s.index(n, a)
		}
		for _, al := range a.Aliases {
			s.index(al, a)
		}
	}
	return s
}

// index records a normalized name->area mapping without clobbering an existing
// one (first writer wins, so a primary name beats a later alias collision).
func (s *Set) index(name string, a *Area) {
	key := Normalize(name)
	if key == "" {
		return
	}
	if _, exists := s.byName[key]; !exists {
		s.byName[key] = a
	}
}

// All returns every area.
func (s *Set) All() []*Area { return s.all }

// ByID looks up an area by its oref id.
func (s *Set) ByID(id string) (*Area, bool) { a, ok := s.byID[id]; return a, ok }

// ByTzevaID looks up an area by its Tzeva Adom numeric id.
func (s *Set) ByTzevaID(id int) (*Area, bool) { a, ok := s.byTzeva[id]; return a, ok }

// ByName looks up an area by any of its names/aliases after normalization.
func (s *Set) ByName(raw string) (*Area, bool) {
	a, ok := s.byName[Normalize(raw)]
	return a, ok
}
