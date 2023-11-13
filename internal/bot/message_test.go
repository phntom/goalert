package bot

//import (
//	"bytes"
//	"encoding/json"
//	"fmt"
//	"github.com/phntom/goalert/internal/district"
//	"net/http"
//	"strings"
//)
//
//type MattermostWebhookPayload struct {
//	Text        string         `json:"text,omitempty"`
//	Channel     string         `json:"channel,omitempty"`
//	Username    string         `json:"username,omitempty"`
//	IconURL     string         `json:"icon_url,omitempty"`
//	IconEmoji   string         `json:"icon_emoji,omitempty"`
//	Attachments []Attachment   `json:"attachments,omitempty"`
//	Type        string         `json:"type,omitempty"`
//	Props       map[string]any `json:"props,omitempty"`
//}
//
//// Attachment structure for Mattermost message attachments
//type Attachment struct {
//	Fallback   string            `json:"fallback"`
//	Color      string            `json:"color"`
//	Pretext    string            `json:"pretext"`
//	Text       string            `json:"text"`
//	AuthorName string            `json:"author_name"`
//	AuthorIcon string            `json:"author_icon"`
//	AuthorLink string            `json:"author_link"`
//	Title      string            `json:"title"`
//	TitleLink  string            `json:"title_link"`
//	Fields     []AttachmentField `json:"fields"`
//	ImageURL   string            `json:"image_url"`
//}
//
//// AttachmentField structure for individual fields in an attachment
//type AttachmentField struct {
//	Short bool   `json:"short"`
//	Title string `json:"title"`
//	Value string `json:"value"`
//}
//
//// SendMattermostWebhook sends a webhook to a Mattermost server
//func SendMattermostWebhook(webhookURL string, payload MattermostWebhookPayload) error {
//	// Marshal the payload into JSON
//	jsonPayload, err := json.Marshal(payload)
//	if err != nil {
//		return fmt.Errorf("error marshaling payload to JSON: %v", err)
//	}
//
//	// Create a new HTTP request
//	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonPayload))
//	if err != nil {
//		return fmt.Errorf("error creating HTTP request: %v", err)
//	}
//
//	// Set the content type to JSON
//	req.Header.Set("Content-Type", "application/json")
//
//	// Execute the HTTP request
//	client := &http.Client{}
//	resp, err := client.Do(req)
//	if err != nil {
//		return fmt.Errorf("error sending HTTP request: %v", err)
//	}
//	defer resp.Body.Close()
//
//	// Check if the HTTP request was successful
//	if resp.StatusCode != http.StatusOK {
//		return fmt.Errorf("error response from Mattermost: %s", resp.Status)
//	}
//
//	fmt.Println("Webhook sent successfully.")
//	return nil
//}
//
//func Test1() {
//	lang := district.Language("he")
//
//	district.GetDistricts()
//	var ids []district.DistrictID
//	for i, _ := range district.SubdivisionsSet {
//		ids = append(ids, i)
//	}
//	cities := IDsToCities(lang, ids)
//	fields := CitiesToFields(cities)
//
//	username := "ירי רקטות וטילים"
//	card := "This is a [card](https://www.example.com) :poop:"
//	fallback := " פאנטום ניסיון2 צבעאדוםביתחנן #נטעים #נס_ציונה"
//	payload := MattermostWebhookPayload{
//		//Text:      "",
//		IconEmoji: "missile-alert",
//		Username:  username,
//		Props: map[string]any{
//			"webhook_display_name": username,
//			"card":                 card,
//			"metadata": map[string]any{
//				"priority": map[string]any{
//					"priority": "urgent",
//				},
//			},
//		},
//		Attachments: []Attachment{
//			{
//				Text:     "היכנסו למרחב מוגן תוך 90 שניות",
//				Fallback: fallback,
//				Pretext:  fallback,
//				Color:    "#CF1434",
//				Fields:   fields,
//			},
//		},
//	}
//
//	// Mattermost webhook URL
//	webhookURL := "https://kix.co.il/hooks/oopw9b9z43d7tp1sttf7778iho"
//
//	// Send the webhook
//	err := SendMattermostWebhook(webhookURL, payload)
//	if err != nil {
//		fmt.Printf("Failed to send webhook: %s\n", err)
//	}
//}
//
//func IDsToCities(lang district.Language, ids []district.DistrictID) map[string][]string {
//	var cities map[string][]string
//	for _, id := range ids {
//		n1, n2 := district.GetCity(id, lang)
//		cities[n1] = append(cities[n1], n2)
//	}
//	return cities
//}
//
//func CitiesToFields(cities map[string][]string) []AttachmentField {
//	var fields []AttachmentField
//	for n1, n2 := range cities {
//		value := strings.Join(n2, "\n")
//		if len(n2) == 1 && district.SubdivisionsSet[district.DistrictID(n2[0])] == "" {
//			value = ""
//		}
//		fields = append(fields, AttachmentField{
//			Title: n1,
//			Value: value,
//			Short: true,
//		})
//	}
//	return fields
//}
