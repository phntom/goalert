package bot

import (
	"github.com/phntom/goalert/internal/district"
)

func NewRocketIDsPresent(message *Message, prevMsg *Message) bool {
	if prevMsg.RocketIDs != nil && message.RocketIDs != nil {
		for rocketID := range message.RocketIDs {
			if !prevMsg.RocketIDs[rocketID] {
				return true
			}
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
	defer b.dedupMutex.Unlock()

	for _, city := range message.Cities {
		prevMsg, ok := b.dedup[city]
		if !ok || prevMsg.IsExpired() || NewRocketIDsPresent(message, prevMsg) ||
			(prevMsg.Category != "" && message.Category != "" && prevMsg.Category != message.Category) {
			continue
		}
		delete(citiesNotFound, city)
		prevMsgs[city] = prevMsg
	}

	for city := range citiesNotFound {
		b.dedup[city] = message
	}

	return prevMsgs, citiesNotFound
}
