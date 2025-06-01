package sources

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
	"github.com/gotd/td/tg"
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

func TestParseMessage_Channel2335255539_KeywordFound(t *testing.T) {
	// 1. Setup Hook
	var capturedPostByHook *model.Post
	hookCalled := false
	CreatePostTestHook = func(postToCreate *model.Post) bool {
		capturedPostByHook = postToCreate
		hookCalled = true
		return true // Indicate hook handled the call, so original CreatePost is skipped
	}
	defer func() { CreatePostTestHook = nil }() // Cleanup hook

	// 2. Setup SourceTelegram and Mocks
	telegramChannelID := int64(2335255539)
	expectedMattermostChannelID := "mock_mattermost_channel_id_123"
	expectedMattermostChannelName := fmt.Sprintf("telegram-%d", telegramChannelID)

	// Assuming bot.Bot has a public 'Channels' field or a setter for testing.
	// If bot.Bot.channels is private and has no setter, this part of the test
	// (verifying channel lookup) cannot be done without changing bot.go.
	// For this test, we proceed assuming 'Channels' is accessible.
	testBot := &bot.Bot{
		Client: &model.Client4{}, // Must be non-nil
		Channels: []*model.Channel{
			{
				Id:   expectedMattermostChannelID,
				Name: expectedMattermostChannelName,
				Type: model.ChannelTypeOpen, // Type can be relevant for channel filtering
			},
			{
				Id:   "other_channel_id",
				Name: "some-other-channel",
				Type: model.ChannelTypeOpen,
			},
		},
	}

	source := &SourceTelegram{
		Bot: testBot,
	}

	// 3. Prepare Input
	keyword := "ירוט" // One of the relevant keywords
	messageText := fmt.Sprintf("This message contains a %s keyword.", keyword)
	expectedPrefixedMessage := "חדשות ישראל בטלגרם: " + messageText

	// Constructing tg.UpdateNewChannelMessage
	// The Message field should be of type tg.MessageClass, which *tg.Message implements.
	// Ensure PeerID is *tg.PeerChannel.
	// Date is a required field for a message to be considered "NotEmpty".
	update := &tg.UpdateNewChannelMessage{
		Message: &tg.Message{
			Out:     false, // Incoming message
			PeerID:  &tg.PeerChannel{ChannelID: telegramChannelID},
			Message: messageText,
			Date:    int(time.Now().Unix()), // Unix timestamp for "now"
		},
	}

	// 4. Call ParseMessage
	// Note: The `e tg.Entities` argument is not used by the part of ParseMessage we're testing.
	err := source.ParseMessage(context.Background(), tg.Entities{}, update)

	// 5. Assertions
	if err != nil {
		t.Errorf("ParseMessage() returned error = %v, wantErr nil", err)
	}

	if !hookCalled {
		t.Fatalf("CreatePostTestHook was not called, message might not have been processed as expected or hook failed.")
	}

	if capturedPostByHook == nil {
		// This check is redundant if hookCalled is true and hook assigns to capturedPostByHook,
		// but good for safety.
		t.Fatalf("CreatePostTestHook was called, but capturedPostByHook is nil.")
	}

	if capturedPostByHook.Message != expectedPrefixedMessage {
		t.Errorf("Captured post message: got '%s', want '%s'", capturedPostByHook.Message, expectedPrefixedMessage)
	}

	if capturedPostByHook.ChannelId != expectedMattermostChannelID {
		t.Errorf("Captured post ChannelId: got '%s', want '%s'", capturedPostByHook.ChannelId, expectedMattermostChannelID)
	}
}

// Test case for when the specific channel (2335255539) is not found in Bot's channels list
func TestParseMessage_Channel2335255539_ChannelNotFound(t *testing.T) {
	var hookCalled bool = false
	CreatePostTestHook = func(postToCreate *model.Post) bool {
		hookCalled = true
		return true
	}
	defer func() { CreatePostTestHook = nil }()

	telegramChannelID := int64(2335255539)
	// Bot's channels list does NOT contain the target channel name "telegram-2335255539"
	testBot := &bot.Bot{
		Client: &model.Client4{},
		Channels: []*model.Channel{
			{
				Id:   "some_other_id",
				Name: "another-channel-name", // Does not match expected name convention
				Type: model.ChannelTypeOpen,
			},
		},
	}

	source := &SourceTelegram{
		Bot: testBot,
	}

	keyword := "אזעק"
	messageText := fmt.Sprintf("Alert: %s in the area!", keyword)

	update := &tg.UpdateNewChannelMessage{
		Message: &tg.Message{
			PeerID:  &tg.PeerChannel{ChannelID: telegramChannelID},
			Message: messageText,
			Date:    int(time.Now().Unix()),
		},
	}

	err := source.ParseMessage(context.Background(), tg.Entities{}, update)

	if err != nil {
		t.Errorf("ParseMessage() returned error = %v, wantErr nil", err)
	}

	if hookCalled {
		t.Errorf("CreatePostTestHook was called, but it should not have been (channel not found).")
	}
	// Further assertions could involve checking mlog for the "Could not find corresponding Mattermost channel" warning,
	// but that requires a more complex logging mock. For this test, checking that CreatePost was not called is sufficient.
}

// Test case for when a keyword is NOT found in a message from channel 2335255539
func TestParseMessage_Channel2335255539_KeywordNotFound(t *testing.T) {
	var hookCalled bool = false
	CreatePostTestHook = func(postToCreate *model.Post) bool {
		hookCalled = true
		return true
	}
	defer func() { CreatePostTestHook = nil }()

	telegramChannelID := int64(2335255539)
	expectedMattermostChannelID := "mock_mattermost_channel_id_123"
	expectedMattermostChannelName := fmt.Sprintf("telegram-%d", telegramChannelID)

	testBot := &bot.Bot{
		Client: &model.Client4{},
		Channels: []*model.Channel{ // Channel exists
			{
				Id:   expectedMattermostChannelID,
				Name: expectedMattermostChannelName,
				Type: model.ChannelTypeOpen,
			},
		},
	}
	source := &SourceTelegram{
		Bot: testBot,
	}

	messageText := "This is a regular message with no special keywords." // No relevant keywords

	update := &tg.UpdateNewChannelMessage{
		Message: &tg.Message{
			PeerID:  &tg.PeerChannel{ChannelID: telegramChannelID},
			Message: messageText,
			Date:    int(time.Now().Unix()),
		},
	}

	err := source.ParseMessage(context.Background(), tg.Entities{}, update)

	if err != nil {
		t.Errorf("ParseMessage() returned error = %v, wantErr nil", err)
	}

	if hookCalled {
		t.Errorf("CreatePostTestHook was called, but it should not have been (no keyword found).")
	}
}
