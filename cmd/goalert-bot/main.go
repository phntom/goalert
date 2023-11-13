package main

import (
	"github.com/phntom/goalert/internal/district"
	"sort"
)

func main() {
	var collect []string
	for language, districts := range district.GetDistricts() {
		if language != "he" {
			continue
		}
		for id, d := range districts {
			name1, _ := district.GetCity(id, language)
			//name := d.SettlementName
			d = d
			collect = append(collect, name1)
		}
	}
	sort.Slice(collect, func(i, j int) bool {
		return len(collect[i]) > len(collect[j])
	})
	print(collect)
}
