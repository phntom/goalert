package bot

import (
	"github.com/phntom/goalert/internal/monitoring"
)

// BotService defines the interface for bot operations that sources might need.
type BotService interface {
	SubmitMessage(m *Message) // m *Message assumes Message is a defined type in package bot
	GetMonitoring() monitoring.Monitoring
}
