package sources

import (
	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
	"testing"
	"time"
)

func Test_processMessage(t *testing.T) {
	districts := district.GetDistricts()
	s := &SourceTelegram{
		Bot: &bot.Bot{},
	}
	s.Bot.Register()
	go s.Bot.AwaitMessage()

	tests := []struct {
		name    string
		now     time.Time
		text    string
		wantErr bool
	}{
		{
			name: "Valid message",
			now:  time.Date(2024, 10, 10, 8, 19, 0, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים [10/10/2024] 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: false,
		},
		{
			name: "Expired message",
			now:  time.Date(2024, 10, 10, 8, 22, 0, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים [10/10/2024] 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := processMessage(tt.text, districts, tt.now, s.Bot)
			if (err != nil) != tt.wantErr {
				t.Errorf("processMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
