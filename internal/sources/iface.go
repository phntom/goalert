package sources

import (
	"github.com/phntom/goalert/internal/bot"
)

type AlertSource interface {
	Register()
	Fetch() []byte
	Parse(content []byte) []*bot.Message
	Run()
}
