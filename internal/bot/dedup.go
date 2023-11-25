package bot

import (
	"github.com/phntom/goalert/internal/district"
)

func NewRocketIDsPresent(message *Message, prevMsg *Message) bool {
	if prevMsg.RocketIDs != nil && message.RocketIDs != nil {
		for rocketID := range message.RocketIDs {
			if prevMsg.RocketIDs[rocketID] {
				continue
			}
			return true
		}
	}
	return false
}

func (b *Bot) GetPrevMsgs(message *Message) (map[district.ID]*Message, map[district.ID]bool) {
	citiesNotFound := make(map[district.ID]bool, len(message.Cities))
	for _, city := range message.Cities {
		citiesNotFound[city] = true
	}

	prevMsgs := make(map[district.ID]*Message, len(message.Cities))
	b.dedupMutex.Lock()
	for _, city := range message.Cities {
		prevMsg, ok := b.dedup[city]
		if !ok || prevMsg.IsExpired() {
			// city not found or message with city is expired
			continue
		}
		if NewRocketIDsPresent(message, prevMsg) {
			// message about city has new rocket ids
			continue
		}
		if prevMsg.Category != "" && message.Category != "" && prevMsg.Category != message.Category {
			// message about city is from a different category
			continue
		}
		if citiesNotFound[city] {
			delete(citiesNotFound, city)
		}
		prevMsgs[city] = prevMsg
	}
	for city := range citiesNotFound {
		b.dedup[city] = message
	}
	b.dedupMutex.Unlock()
	return prevMsgs, citiesNotFound
}
