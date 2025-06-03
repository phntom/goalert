package sources

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/assert"
)

// MockBot for capturing submitted messages
type MockBot struct {
	bot.Bot
	SubmittedMessages []*bot.Message
	Client            *model.Client4   // Added to satisfy Bot struct if it has it
	Channels          []*model.Channel // Added to satisfy Bot struct if it has it
}

func (m *MockBot) SubmitMessage(msg *bot.Message) {
	m.SubmittedMessages = append(m.SubmittedMessages, msg)
}

func (m *MockBot) Register()               {}
func (m *MockBot) AwaitMessage()           {}
func (m *MockBot) DirectMessage(post *model.Post, language string) {}

var jerusalem = time.FixedZone("IST", 2*60*60)

var tests = []struct {
	name             string
	now              time.Time
	text             string
	overrideCategory string
	wantErr          bool
	expectedCategory string
	expectedCities   []string
}{
	{
		name: "Regular rocket alert - Valid",
		now:  time.Date(2024, 10, 10, 11, 19, 30, 0, jerusalem),
		text: `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
		overrideCategory: "",
		wantErr:          false,
		expectedCategory: "rockets",
		expectedCities:   []string{"מטולה"},
	},
	// --- Regular Alerts (90s expiration) ---
	{
		name: "Regular Alert - Valid (under 90s)",
		now:  time.Date(2024, 10, 10, 11, 20, 29, 0, jerusalem), // PubDate 11:19, 89s diff
		text: `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.`,
		overrideCategory: "",
		wantErr:          false,
		expectedCategory: "rockets",
		expectedCities:   []string{"מטולה"},
	},
	{
		name: "Regular Alert - Expired (just over 90s)",
		now:  time.Date(2024, 10, 10, 11, 20, 31, 0, jerusalem), // PubDate 11:19, 91s diff
		text: `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.`,
		overrideCategory: "",
		wantErr:          true, // Should be expired by 90s rule
	},
	{
		name: "Regular Alert - Expired (121s diff, previously NOT expired by 300s blanket rule)",
		now:  time.Date(2024, 10, 10, 11, 21, 1, 0, jerusalem), // PubDate 11:19, 121s diff
		text: `🚨 ירי רקטות וטילים (10/10/2024) 11:19

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
		overrideCategory: "",
		wantErr:          true, // Now expired by 90s rule
	},
	{
		name: "Regular Alert - Expired (299s diff, previously NOT expired by 300s blanket rule)",
		now:  time.Date(2024, 10, 10, 11, 4, 59, 0, jerusalem), // PubDate 11:00, 299s diff
		text: `🚨 ירי רקטות וטילים (10/10/2024) 11:00

אזור קו העימות
מטולה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
		overrideCategory: "",
		wantErr:          true, // Now expired by 90s rule
	},
	// --- Early Alerts (300s expiration) ---
	{
		name: "Early Alert - Valid (under 90s)", // Still valid under 300s
		now:  time.Date(2024, 10, 10, 12, 0, 30, 0, jerusalem), // PubDate 12:00, 30s diff
		text: `(10/10/2024) 12:00 התרעה מוקדמת: בדקות הקרובות צפויות להתקבל התרעות באזורך.`,
		overrideCategory: "rockets",
		wantErr:          false,
		expectedCategory: "rockets",
		expectedCities:   []string{},
	},
	{
		name: "Early Alert - Valid (over 90s, under 300s)",
		now:  time.Date(2024, 10, 10, 12, 2, 30, 0, jerusalem), // PubDate 12:00, 150s diff
		text: `(10/10/2024) 12:00 התרעה מוקדמת: בדקות הקרובות צפויות להתקבל התרעות באזורך.`,
		overrideCategory: "rockets",
		wantErr:          false, // Should NOT be expired by 300s rule
		expectedCategory: "rockets",
		expectedCities:   []string{},
	},
	{
		name: "Early Alert - Not Expired (just under 300s)",
		now:  time.Date(2024, 10, 10, 12, 4, 59, 0, jerusalem), // PubDate 12:00, 299s diff
		text: `(10/10/2024) 12:00 התרעה מוקדמת: בדקות הקרובות צפויות להתקבל התרעות באזורך.`,
		overrideCategory: "rockets",
		wantErr:          false,
		expectedCategory: "rockets",
		expectedCities:   []string{},
	},
	{
		name: "Early Alert - Expired (just over 300s)",
		now:  time.Date(2024, 10, 10, 12, 5, 1, 0, jerusalem), // PubDate 12:00, 301s diff
		text: `(10/10/2024) 12:00 התרעה מוקדמת: בדקות הקרובות צפויות להתקבל התרעות באזורך.`,
		overrideCategory: "rockets",
		wantErr:          true, // Should be expired by 300s rule
	},
	// --- General cases ---
	{
		name: "UAV alert - Valid (Regular 90s rule, under 90s)",
		now:  time.Date(2024, 10, 10, 13, 0, 30, 0, jerusalem), // PubDate 13:00, 30s diff
		text: `🚨 חדירת כלי טיס עוין (10/10/2024) 13:00

אזור הצפון
קריית שמונה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.`,
		overrideCategory: "", // This is a regular alert
		wantErr:          false,
		expectedCategory: "uav",
		expectedCities:   []string{"קריית שמונה"},
	},
	{
		name: "UAV alert - Expired (Regular 90s rule, over 90s)",
		now:  time.Date(2024, 10, 10, 13, 1, 31, 0, jerusalem), // PubDate 13:00, 91s diff
		text: `🚨 חדירת כלי טיס עוין (10/10/2024) 13:00

אזור הצפון
קריית שמונה (מיידי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.`,
		overrideCategory: "", // This is a regular alert
		wantErr:          true, // Should be expired
	},
	{
		name: "Alert with no cities (Regular, not expired)",
		now:  time.Date(2024, 10, 10, 14, 0, 30, 0, jerusalem), // PubDate 14:00, 30s diff
		text: `🚨 ירי רקטות וטילים (10/10/2024) 14:00

התרעה כללית באזור לא מוגדר.

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.`,
		overrideCategory: "",
		wantErr:          false,
		expectedCategory: "rockets",
		expectedCities:   []string{},
	},
	{
		name: "Regular Alert - Previously 'not expired by 300s rule' (120s diff), now EXPIRED by 90s rule",
		now:  time.Date(2025, 5, 29, 21, 25, 0, 0, jerusalem), // PubDate 21:23, 120s diff
		text: `🚨 ירי רקטות וטילים (29/5/2025) 21:23

אזור שומרון
דולב, טלמון, נריה (דקה וחצי)

היכנסו למרחב המוגן ושהו בו למשך 10 דקות.
להנחיות המלאות - https://www.oref.org.il/heb/life-saving-guidelines/rocket-and-missile-attacks`,
		overrideCategory: "",
		wantErr:          true, // Now expired by 90s rule
	},
}

func Test_processMessage(t *testing.T) {
	districts := district.GetDistricts()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBot := &MockBot{
				SubmittedMessages: make([]*bot.Message, 0),
				Client:            &model.Client4{},
				// Channels:          []*model.Channel{}, // Initialize if needed
			}
			err := processMessage(tt.text, districts, tt.now, &mockBot.Bot, tt.overrideCategory)

			if tt.wantErr {
				assert.Error(t, err, "Test: %s. Expected error, got nil", tt.name)
			} else {
				assert.NoError(t, err, "Test: %s. Expected no error, got %v", tt.name, err)
			}

			if !tt.wantErr && tt.expectedCategory != "" {
				if len(tt.expectedCities) > 0 {
					if assert.NotEmpty(t, mockBot.SubmittedMessages, "Test: %s. Expected messages, but none were submitted.", tt.name) {
						submittedMsg := mockBot.SubmittedMessages[0]
						assert.Equal(t, tt.expectedCategory, submittedMsg.Category, "Test: %s. Category mismatch.", tt.name)
					}
				} else { // len(tt.expectedCities) == 0
					assert.Empty(t, mockBot.SubmittedMessages, "Test: %s. Expected no messages, but some were submitted.", tt.name)
				}
			}
		})
	}
}

// TestParseMessage_EarlyAlert tests that ParseMessage correctly identifies an early alert
// and calls processMessage with the "rockets" category.
func TestParseMessage_EarlyAlert(t *testing.T) {
	mockBot := &MockBot{
		SubmittedMessages: make([]*bot.Message, 0),
		Client:            &model.Client4{},
	}
	source := &SourceTelegram{
		Bot: &mockBot.Bot,
	}

	alertTime := time.Now().In(jerusalem).Add(-15 * time.Second) // Recent to avoid expiration
	alertDateStr := alertTime.Format("02/01/2006")               // DD/MM/YYYY
	alertTimeStr := alertTime.Format("15:04")                   // HH:MM
	earlyAlertText := fmt.Sprintf("(%s) %s התרעה מוקדמת: בדקות הקרובות צפויות להתקבל התרעות באזורך.", alertDateStr, alertTimeStr)

	update := &tg.UpdateNewChannelMessage{
		Message: &tg.Message{
			Out:     false,
			PeerID:  &tg.PeerChannel{ChannelID: 1441886157}, // pikudhaoref_all
			Message: earlyAlertText,
			Date:    int(time.Now().Unix()),
		},
	}

	err := source.ParseMessage(context.Background(), tg.Entities{}, update)
	assert.NoError(t, err, "ParseMessage failed for early alert.")
	assert.Empty(t, mockBot.SubmittedMessages, "Expected no messages submitted for early alert text with no cities.")
}

func TestParseMessage_Channel2335255539_KeywordFound(t *testing.T) {
	var capturedPostByHook *model.Post
	hookCalled := false
	CreatePostTestHook = func(postToCreate *model.Post) bool {
		capturedPostByHook = postToCreate
		hookCalled = true
		return true
	}
	defer func() { CreatePostTestHook = nil }()

	telegramChannelID := int64(2335255539)
	expectedMattermostChannelID := "mock_mattermost_channel_id_123"
	expectedMattermostChannelName := fmt.Sprintf("telegram-%d", telegramChannelID)

	testBot := &MockBot{ // Use MockBot
		Client: &model.Client4{},
		Channels: []*model.Channel{
			{Id: expectedMattermostChannelID, Name: expectedMattermostChannelName, Type: model.ChannelTypeOpen},
			{Id: "other_channel_id", Name: "some-other-channel", Type: model.ChannelTypeOpen},
		},
	}
	source := &SourceTelegram{
		Bot: &testBot.Bot, // Pass the embedded *bot.Bot
	}

	keyword := "ירוט"
	messageText := fmt.Sprintf("This message contains a %s keyword.", keyword)
	expectedPrefixedMessage := "חדשות ישראל בטלגרם: " + messageText
	updateMessage := &tg.Message{
		Out:     false,
		PeerID:  &tg.PeerChannel{ChannelID: telegramChannelID},
		Message: messageText,
		Date:    int(time.Now().Unix()),
	}
	update := &tg.UpdateNewChannelMessage{Message: updateMessage}

	err := source.ParseMessage(context.Background(), tg.Entities{}, update)
	assert.NoError(t, err)
	assert.True(t, hookCalled, "CreatePostTestHook was not called.")
	if capturedPostByHook != nil { // Check to prevent nil dereference if hookCalled is false
		assert.Equal(t, expectedPrefixedMessage, capturedPostByHook.Message)
		assert.Equal(t, expectedMattermostChannelID, capturedPostByHook.ChannelId)
	} else if hookCalled {
		t.Fatal("Hook was called but capturedPostByHook is nil")
	}
}

func TestParseMessage_Channel2335255539_ChannelNotFound(t *testing.T) {
	hookCalled := false
	CreatePostTestHook = func(postToCreate *model.Post) bool {
		hookCalled = true
		return true
	}
	defer func() { CreatePostTestHook = nil }()

	telegramChannelID := int64(2335255539)
	testBot := &MockBot{ // Use MockBot
		Client:   &model.Client4{},
		Channels: []*model.Channel{{Id: "some_other_id", Name: "another-channel-name", Type: model.ChannelTypeOpen}},
	}
	source := &SourceTelegram{
		Bot: &testBot.Bot, // Pass the embedded *bot.Bot
	}

	keyword := "אזעק"
	messageText := fmt.Sprintf("Alert: %s in the area!", keyword)
	updateMessage := &tg.Message{
		PeerID:  &tg.PeerChannel{ChannelID: telegramChannelID},
		Message: messageText,
		Date:    int(time.Now().Unix()),
	}
	update := &tg.UpdateNewChannelMessage{Message: updateMessage}

	err := source.ParseMessage(context.Background(), tg.Entities{}, update)
	assert.NoError(t, err)
	assert.False(t, hookCalled, "CreatePostTestHook was called, but should not have been.")
}

func TestParseMessage_Channel2335255539_KeywordNotFound(t *testing.T) {
	hookCalled := false
	CreatePostTestHook = func(postToCreate *model.Post) bool {
		hookCalled = true
		return true
	}
	defer func() { CreatePostTestHook = nil }()

	telegramChannelID := int64(2335255539)
	expectedMattermostChannelID := "mock_mattermost_channel_id_123"
	expectedMattermostChannelName := fmt.Sprintf("telegram-%d", telegramChannelID)
	testBot := &MockBot{ // Use MockBot
		Client:   &model.Client4{},
		Channels: []*model.Channel{{Id: expectedMattermostChannelID, Name: expectedMattermostChannelName, Type: model.ChannelTypeOpen}},
	}
	source := &SourceTelegram{
		Bot: &testBot.Bot, // Pass the embedded *bot.Bot
	}

	messageText := "This is a regular message with no special keywords."
	updateMessage := &tg.Message{
		PeerID:  &tg.PeerChannel{ChannelID: telegramChannelID},
		Message: messageText,
		Date:    int(time.Now().Unix()),
	}
	update := &tg.UpdateNewChannelMessage{Message: updateMessage}

	err := source.ParseMessage(context.Background(), tg.Entities{}, update)
	assert.NoError(t, err)
	assert.False(t, hookCalled, "CreatePostTestHook was called, but should not have been.")

}
