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
			name: "Original Valid message (updated format)",
			// Message time: (10/10/2024) 11:19 IDT. For UTC, assuming IDT is UTC+3, this is 08:19 UTC.
			// "now" is 30 seconds after message time.
			now: time.Date(2024, 10, 10, 8, 19, 30, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: false,
		},
		{
			name: "Original Expired message (updated format)",
			// Message time: (10/10/2024) 11:19 IDT (08:19 UTC)
			// "now" is 121 seconds after message time (08:21:01 UTC)
			now: time.Date(2024, 10, 10, 8, 21, 1, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: true,
		},
		{
			name: "New Message 1 - Non-expired",
			// Message time: (29/05/2025) 21:23 IDT. Assuming IDT is UTC+3, this is 18:23 UTC.
			// "now" is 30 seconds after message time.
			now: time.Date(2025, 5, 29, 18, 23, 30, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:23

אזור שומרון
דולב, טלמון, נריה (דקה וחצי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: false,
		},
		{
			name: "New Message 1 - Expired",
			// Message time: (29/05/2025) 21:23 IDT (18:23 UTC)
			// "now" is 120 seconds after message time (18:25:00 UTC)
			now: time.Date(2025, 5, 29, 18, 25, 0, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:23

אזור שומרון
דולב, טלמון, נריה (דקה וחצי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: true,
		},
		{
			name: "New Message 2 - Non-expired",
			// Message time: (29/05/2025) 21:22 IDT (18:22 UTC)
			// "now" is 30 seconds after message time.
			now: time.Date(2025, 5, 29, 18, 22, 30, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:22

אזור השפלה
גאליה (דקה)

אזור לכיש
בית אלעזרי, בית גמליאל, יבנה, כפר הנגיד, מעון צופיה, מפעל אגריגדה  (דקה)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
			wantErr: false,
		},
		{
			name: "New Message 2 - Expired",
			// Message time: (29/05/2025) 21:22 IDT (18:22 UTC)
			// "now" is 120 seconds after message time (18:24:00 UTC)
			now: time.Date(2025, 5, 29, 18, 24, 0, 0, time.UTC),
			text: `🚨 ירי רקטות וטילים (29/5/2025) 21:22

אזור השפלה
גאליה (דקה)

אזור לכיש
בית אלעזרי, בית גמליאל, יבנה, כפר הנגיד, מעון צופיה, מפעל אגריגדה  (דקה)

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
