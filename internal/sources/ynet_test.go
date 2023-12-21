package sources

import (
	"encoding/json"
	"fmt"
	"github.com/go-test/deep"
	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/district"
	"reflect"
	"testing"
	"time"
)

//func TestSourceYnet_Added(t *testing.T) {
//	type fields struct {
//		seen map[string][]district.ID
//	}
//	type args struct {
//		parsed map[string][]district.ID
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   map[string][]district.ID
//	}{
//		{
//			name:   "empty",
//			fields: fields{seen: make(map[string][]district.ID)},
//			args:   args{parsed: make(map[string][]district.ID)},
//			want:   make(map[string][]district.ID),
//		},
//		{
//			name:   "TestEmptyInitialData",
//			fields: fields{seen: make(map[string][]district.ID)},
//			args:   args{parsed: map[string][]district.ID{"instruction1": {"1", "2", "3"}}},
//			want:   map[string][]district.ID{"instruction1": {"1", "2", "3"}},
//		},
//		{
//			name:   "TestNoNewData",
//			fields: fields{seen: map[string][]district.ID{"instruction1": {"1", "2", "3"}}},
//			args:   args{parsed: map[string][]district.ID{"instruction1": {"1", "2", "3"}}},
//			want:   make(map[string][]district.ID),
//		},
//		{
//			name:   "TestNewDataAdded",
//			fields: fields{seen: map[string][]district.ID{"instruction1": {"1", "2"}}},
//			args:   args{parsed: map[string][]district.ID{"instruction1": {"2", "3", "4"}}},
//			want:   map[string][]district.ID{"instruction1": {"3", "4"}},
//		},
//		{
//			name:   "TestMixedData",
//			fields: fields{seen: map[string][]district.ID{"instruction1": {"1", "2"}, "instruction2": {"5"}}},
//			args:   args{parsed: map[string][]district.ID{"instruction1": {"2", "3"}, "instruction2": {"4", "5"}}},
//			want:   map[string][]district.ID{"instruction1": {"3"}, "instruction2": {"4"}},
//		},
//		{
//			name:   "TestNoOverlap",
//			fields: fields{seen: map[string][]district.ID{"instruction1": {"1", "2"}}},
//			args:   args{parsed: map[string][]district.ID{"instruction2": {"3", "4"}}},
//			want:   map[string][]district.ID{"instruction2": {"3", "4"}},
//		},
//		{
//			name:   "TestEmptyParsed",
//			fields: fields{seen: map[string][]district.ID{"instruction1": {"1", "2", "3"}}},
//			args:   args{parsed: make(map[string][]district.ID)},
//			want:   make(map[string][]district.ID),
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &SourceYnet{
//				seen: tt.fields.seen,
//			}
//			if got := s.Added(tt.args.parsed); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("Added() = %v, want %v", got, tt.want)
//			}
//			if !reflect.DeepEqual(s.seen, tt.args.parsed) {
//				t.Errorf("Added() state not updated")
//			}
//		})
//	}
//}

func GenerateContent(items []YnetMessageItem) []byte {
	j, _ := json.Marshal(YnetMessage{Alerts: YnetMessageItems{Items: items}})
	result := fmt.Sprintf("jsonCallback(%s);", j)
	return []byte(result)
}

func TestSourceYnet_Parse(t *testing.T) {
	district.GetDistricts()
	type args struct {
		content []byte
	}
	tests := []struct {
		name string
		args args
		want []bot.Message
	}{
		{
			name: "empty1",
			args: args{
				content: []byte(""),
			},
			want: nil,
		},
		{
			name: "empty2",
			args: args{
				content: []byte(" []"),
			},
			want: nil,
		},
		{
			name: "empty3",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": []}});"),
			},
			want: nil,
		},
		{
			name: "empty4",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": ]}});"),
			},
			want: nil,
		},
		{
			name: "empty5",
			args: args{
				content: []byte("[]"),
			},
			want: nil,
		},
		{
			name: "sanity",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "שדה ניצן",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תלמי אליהו",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תקומה",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "מבטחים, עמיעוז, ישע",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 15,
					Cities: []district.ID{
						"1211",
						"1289",
						"1349",
						"697",
					},
					RocketIDs: map[string]bool{
						"6c38fbbd-d8c0-40e4-bfe0-a17b1657203e": true,
					},
				},
			},
		},
		{
			name: "important city solo",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "11111111-1111-1111-1111-111111111111",
							Time:        "11:11",
							Title:       "תל אביב - מרכז העיר",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 90,
					Cities: []district.ID{
						"6031",
					},
					RocketIDs: map[string]bool{
						"11111111-1111-1111-1111-111111111111": true,
					},
				},
			},
		},
		{
			name: "only an exercise",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "11111111-1111-1111-1111-111111111111",
							Time:        "11:11",
							Title:       "תל אביב - מרכז העיר",
							Description: "ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: nil,
		},
		{
			name: "exercise and real rocket 1",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "11111111-1111-1111-1111-111111111111",
							Time:        "11:11",
							Title:       "תל אביב - מרכז העיר",
							Description: "ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "22222222-2222-2222-2222-222222222222",
							Time:        "11:12",
							Title:       "באר שבע - מערב",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 60,
					Cities: []district.ID{
						"6006",
					},
					RocketIDs: map[string]bool{
						"22222222-2222-2222-2222-222222222222": true,
					},
				},
			},
		},
		{
			name: "exercise and real rocket 2",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"22222222-1111-1111-1111-111111111111\",\"pubdate\": \"11:12\",\"title\": \"באר שבע - מערב\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"11111111-1111-1111-1111-111111111111\",\"pubdate\": \"11:11\",\"title\": \"תל אביב - מרכז העיר\",\"description\": \"ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 60,
					Cities: []district.ID{
						"6006",
					},
					RocketIDs: map[string]bool{
						"22222222-1111-1111-1111-111111111111": true,
					},
				},
			},
		},
		{
			name: "multiple cities same guid",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"אביבים\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"ברעם\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"דוב''ב\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"יראון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"מתת\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"סאסא\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 0,
					Cities: []district.ID{
						"7",
						"262",
						"364",
						"540",
						"838",
						"936",
					},
					RocketIDs: map[string]bool{
						"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe": true,
					},
				},
			},
		},
		{
			name: "real prerender panic",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"2408a61c-3139-492c-8a86-6f61431a25c4\",\"pubdate\": \"13:27\",\"title\": \"אביגדור\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"2408a61c-3139-492c-8a86-6f61431a25c4\",\"pubdate\": \"13:27\",\"title\": \"באר טוביה\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"2408a61c-3139-492c-8a86-6f61431a25c4\",\"pubdate\": \"13:27\",\"title\": \"כפר ורבורג\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"2408a61c-3139-492c-8a86-6f61431a25c4\",\"pubdate\": \"13:27\",\"title\": \"קריית מלאכי\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"2408a61c-3139-492c-8a86-6f61431a25c4\",\"pubdate\": \"13:27\",\"title\": \"מרכז שפירא\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"2408a61c-3139-492c-8a86-6f61431a25c4\",\"pubdate\": \"13:27\",\"title\": \"עין צורים\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"2408a61c-3139-492c-8a86-6f61431a25c4\",\"pubdate\": \"13:27\",\"title\": \"שפיר\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"958f145c-4ecf-494e-bfc5-7f2b76c9061a\",\"pubdate\": \"13:27\",\"title\": \"בני ראם\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"958f145c-4ecf-494e-bfc5-7f2b76c9061a\",\"pubdate\": \"13:27\",\"title\": \"גני טל\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"958f145c-4ecf-494e-bfc5-7f2b76c9061a\",\"pubdate\": \"13:27\",\"title\": \"כפר הרי''ף וצומת ראם\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"958f145c-4ecf-494e-bfc5-7f2b76c9061a\",\"pubdate\": \"13:27\",\"title\": \"תלמי יחיאל\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"958f145c-4ecf-494e-bfc5-7f2b76c9061a\",\"pubdate\": \"13:27\",\"title\": \"אזור תעשייה כנות\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"958f145c-4ecf-494e-bfc5-7f2b76c9061a\",\"pubdate\": \"13:27\",\"title\": \"בני עי''ש\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"958f145c-4ecf-494e-bfc5-7f2b76c9061a\",\"pubdate\": \"13:27\",\"title\": \"חצב\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"958f145c-4ecf-494e-bfc5-7f2b76c9061a\",\"pubdate\": \"13:27\",\"title\": \"כנות\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"958f145c-4ecf-494e-bfc5-7f2b76c9061a\",\"pubdate\": \"13:27\",\"title\": \"פארק תעשייה ראם\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"6b2c9cb7-c972-48ea-af6d-8624c147635d\",\"pubdate\": \"13:27\",\"title\": \"חצור\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"a05886c0-c773-4195-8d1b-3fe89ebc9abc\",\"pubdate\": \"13:26\",\"title\": \"בית חנן\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"a05886c0-c773-4195-8d1b-3fe89ebc9abc\",\"pubdate\": \"13:26\",\"title\": \"בית עובד\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"a05886c0-c773-4195-8d1b-3fe89ebc9abc\",\"pubdate\": \"13:26\",\"title\": \"גן שורק\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"a05886c0-c773-4195-8d1b-3fe89ebc9abc\",\"pubdate\": \"13:26\",\"title\": \"נטעים\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"a05886c0-c773-4195-8d1b-3fe89ebc9abc\",\"pubdate\": \"13:26\",\"title\": \"נס ציונה\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"a05886c0-c773-4195-8d1b-3fe89ebc9abc\",\"pubdate\": \"13:26\",\"title\": \"עיינות\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"88ab2a44-3c5c-4a19-bd73-17637ff5ca11\",\"pubdate\": \"13:26\",\"title\": \"גבעת וושינגטון\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"88ab2a44-3c5c-4a19-bd73-17637ff5ca11\",\"pubdate\": \"13:26\",\"title\": \"משגב דב\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"88ab2a44-3c5c-4a19-bd73-17637ff5ca11\",\"pubdate\": \"13:26\",\"title\": \"שדמה\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"9c6f1963-e1ee-4712-a1bc-e7e098da917c\",\"pubdate\": \"13:26\",\"title\": \"בית גמליאל\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"9c6f1963-e1ee-4712-a1bc-e7e098da917c\",\"pubdate\": \"13:26\",\"title\": \"בניה\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"9c6f1963-e1ee-4712-a1bc-e7e098da917c\",\"pubdate\": \"13:26\",\"title\": \"כפר מרדכי\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"9c6f1963-e1ee-4712-a1bc-e7e098da917c\",\"pubdate\": \"13:26\",\"title\": \"מישר\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"9c6f1963-e1ee-4712-a1bc-e7e098da917c\",\"pubdate\": \"13:26\",\"title\": \"עשרת\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"890c9e7c-b9d8-455b-b29e-fc05aadc0238\",\"pubdate\": \"13:26\",\"title\": \"גדרה\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"890c9e7c-b9d8-455b-b29e-fc05aadc0238\",\"pubdate\": \"13:26\",\"title\": \"קדרון\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"f479c8c5-f74f-4e19-a9cc-e472f7a16872\",\"pubdate\": \"13:26\",\"title\": \"נווה מבטח\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"24598faa-832e-482d-9909-b9698d447c5b\",\"pubdate\": \"13:26\",\"title\": \"אשדוד - ח,ט,י,יג,יד,טז\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}},{\"item\": {\"guid\": \"d8953844-8901-4292-94db-1ee58e37b700\",\"pubdate\": \"13:26\",\"title\": \"יבנה\",\"description\": \"היכנסו למרחב המוגן ושהו בו 10 דקות\",\"link\": \"\"}}]}});"),
			},
			want: []bot.Message{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SourceYnet{}
			s.Register()
			got1 := s.Parse(tt.args.content)
			var got []bot.Message
			for i, message := range got1 {
				message.Expire = tt.want[i].Expire
				got = append(got, *message)
			}
			diff := deep.Equal(got, tt.want)
			if diff != nil {
				t.Errorf("Render() = %v, want %v, diff: %v", got, tt.want, diff)
			}

		})
	}
}

func TestSourceYnet_Parse_Persistence(t *testing.T) {
	district.GetDistricts()
	type args struct {
		content []byte
	}
	tests := []struct {
		name string
		args args
		want []bot.Message
	}{
		{
			name: "sanity",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "שדה ניצן",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "7c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תלמי אליהו",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "8c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תקומה",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "9c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "מבטחים, עמיעוז, ישע",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 15,
					Cities: []district.ID{
						"1211",
					},
					RocketIDs: map[string]bool{
						"6c38fbbd-d8c0-40e4-bfe0-a17b1657203e": true,
					},
				},
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 15,
					Cities: []district.ID{
						"1289",
					},
					RocketIDs: map[string]bool{
						"7c38fbbd-d8c0-40e4-bfe0-a17b1657203e": true,
					},
				},
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 15,
					Cities: []district.ID{
						"1349",
					},
					RocketIDs: map[string]bool{
						"8c38fbbd-d8c0-40e4-bfe0-a17b1657203e": true,
					},
				},
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 15,
					Cities: []district.ID{
						"697",
					},
					RocketIDs: map[string]bool{
						"9c38fbbd-d8c0-40e4-bfe0-a17b1657203e": true,
					},
				},
			},
		},
		{
			name: "repeat",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "שדה ניצן",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "7c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תלמי אליהו",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "8c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תקומה",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "9c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "מבטחים, עמיעוז, ישע",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: nil,
		},
		{
			name: "empty1",
			args: args{
				content: GenerateContent([]YnetMessageItem{}),
			},
			want: nil,
		},
		{
			name: "first",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "שדה ניצן",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 15,
					Cities: []district.ID{
						"1211",
					},
					RocketIDs: map[string]bool{
						"6c38fbbd-d8c0-40e4-bfe0-a17b1657203e": true,
					},
				},
			},
		},
		{
			name: "first and second",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "שדה ניצן",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "7c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תלמי אליהו",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 15,
					Cities: []district.ID{
						"1289",
					},
					RocketIDs: map[string]bool{
						"7c38fbbd-d8c0-40e4-bfe0-a17b1657203e": true,
					},
				},
			},
		},
		{
			name: "only second",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "7c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תלמי אליהו",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: nil,
		},
		{
			name: "first and second again",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "שדה ניצן",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "7c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תלמי אליהו",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: nil,
		},
		{
			name: "first second and third",
			args: args{
				content: GenerateContent([]YnetMessageItem{
					{
						Item: YnetMessageItemConcrete{
							Guid:        "6c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "שדה ניצן",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "7c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תלמי אליהו",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
					{
						Item: YnetMessageItemConcrete{
							Guid:        "8c38fbbd-d8c0-40e4-bfe0-a17b1657203e",
							Time:        "20:53",
							Title:       "תקומה",
							Description: "היכנסו למרחב המוגן",
							Link:        "",
						},
					},
				}),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 15,
					Cities: []district.ID{
						"1349",
					},
					RocketIDs: map[string]bool{
						"8c38fbbd-d8c0-40e4-bfe0-a17b1657203e": true,
					},
				},
			},
		},
		{
			name: "real 1",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"27b246ea-a678-4e96-81bf-5ab905992cdc\",\"pubdate\": \"11:40\",\"title\": \"אזור תעשייה צפוני אשקלון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"27b246ea-a678-4e96-81bf-5ab905992cdc\",\"pubdate\": \"11:40\",\"title\": \"בת הדר\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - דרום\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - צפון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 30,
					Cities: []district.ID{
						"68",
						"267",
					},
					RocketIDs: map[string]bool{
						"27b246ea-a678-4e96-81bf-5ab905992cdc": true,
					},
				},
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 30,
					Cities: []district.ID{
						"6037",
						"6039",
					},
					RocketIDs: map[string]bool{
						"c5ebbc91-51e0-46dd-80de-13618a459987": true,
					},
				},
			},
		},
		{
			name: "real 2",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - דרום\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - צפון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: nil,
		},
		{
			name: "real 3",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"f038657b-99e1-48ec-b5c2-3e49c409b3bb\",\"pubdate\": \"11:40\",\"title\": \"בית שקמה\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"f038657b-99e1-48ec-b5c2-3e49c409b3bb\",\"pubdate\": \"11:40\",\"title\": \"גיאה\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"אזור תעשייה הדרומי אשקלון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"זיקים\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"יד מרדכי\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"כרמיה\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"מבקיעים\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"27b246ea-a678-4e96-81bf-5ab905992cdc\",\"pubdate\": \"11:40\",\"title\": \"אזור תעשייה צפוני אשקלון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"27b246ea-a678-4e96-81bf-5ab905992cdc\",\"pubdate\": \"11:40\",\"title\": \"בת הדר\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - דרום\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - צפון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: []bot.Message{
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 30,
					Cities: []district.ID{
						"230",
						"321",
					},
					RocketIDs: map[string]bool{
						"f038657b-99e1-48ec-b5c2-3e49c409b3bb": true,
					},
				},
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 30,
					Cities: []district.ID{
						"50",
						"698",
					},
					RocketIDs: map[string]bool{
						"ce68fed6-5c8a-4795-8bce-5f20dcd42d58": true,
					},
				},
				{
					Instructions:  "instructions",
					Category:      "",
					SafetySeconds: 15,
					Cities: []district.ID{
						"414",
						"505",
						"666",
					},
					RocketIDs: map[string]bool{
						"ce68fed6-5c8a-4795-8bce-5f20dcd42d58": true,
					},
				},
			},
		},
	}
	s := &SourceYnet{}
	s.Register()
	now := time.Now()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.Parse(tt.args.content)
			gotSet := make(map[string]*bot.Message)
			for _, message := range got {
				message.Expire = now
				gotSet[message.GetHash()] = message
			}
			wantSet := make(map[string]bot.Message)
			for _, message := range tt.want {
				message.Expire = now
				wantSet[message.GetHash()] = message
			}
			if !reflect.DeepEqual(gotSet, wantSet) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
