// Package alert defines the source-agnostic domain types: an Alert is one
// immutable fact reported by a source (a category affecting some areas at a
// point in time), and Category is the canonical Pikud HaOref category model.
package alert

import "time"

// Kind classifies where an alert sits in the warning lifecycle.
type Kind int

const (
	// KindAlert is an active siren.
	KindAlert Kind = iota
	// KindPreAlert is an early "expect alerts shortly" warning (category 14).
	KindPreAlert
	// KindEnd is an all-clear (category 13).
	KindEnd
)

// Alert is a single fact reported by a source. Areas holds the raw area names
// as the source spelled them (Hebrew); the publisher resolves them to the
// canonical area set, which also handles "all areas" de-aggregation.
type Alert struct {
	Category Category
	Kind     Kind
	Areas    []string
	At       time.Time
	Source   string
	IDs      []string
}

// KindOf derives the lifecycle Kind from a category.
func KindOf(c Category) Kind {
	switch c {
	case CatEndAlert:
		return KindEnd
	case CatPreAlert:
		return KindPreAlert
	default:
		return KindAlert
	}
}
