package sources

import (
	"encoding/json"
	"fmt"
	"github.com/phntom/goalert/internal/district"
	"reflect"
	"testing"
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
		want map[string][]district.ID
	}{
		{
			name: "empty1",
			args: args{
				content: []byte(""),
			},
			want: make(map[string][]district.ID),
		},
		{
			name: "empty2",
			args: args{
				content: []byte(" []"),
			},
			want: make(map[string][]district.ID),
		},
		{
			name: "empty3",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": []}});"),
			},
			want: make(map[string][]district.ID),
		},
		{
			name: "empty4",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": ]}});"),
			},
			want: make(map[string][]district.ID),
		},
		{
			name: "empty5",
			args: args{
				content: []byte("[]"),
			},
			want: make(map[string][]district.ID),
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
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"1211",
					"1289",
					"1349",
					"697",
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
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"6031",
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
			want: map[string][]district.ID{
				"ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן": {
					"6031",
				},
			},
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
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"6006",
				},
				"ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן": {
					"6031",
				},
			},
		},
		{
			name: "exercise and real rocket 2",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"22222222-1111-1111-1111-111111111111\",\"pubdate\": \"11:12\",\"title\": \"באר שבע - מערב\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"11111111-1111-1111-1111-111111111111\",\"pubdate\": \"11:11\",\"title\": \"תל אביב - מרכז העיר\",\"description\": \"ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"6006",
				},
				"ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן": {
					"6031",
				},
			},
		},
		{
			name: "multiple cities same guid",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"אביבים\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"ברעם\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"דוב''ב\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"יראון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"מתת\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"61d68de5-f4f3-4d53-a67d-d2ce24ac9efe\",\"pubdate\": \"22:34\",\"title\": \"סאסא\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"7",
					"262",
					"364",
					"540",
					"838",
					"936",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SourceYnet{}
			s.Register()
			if got := s.Parse(tt.args.content); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
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
		want map[string][]district.ID
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
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"1211",
					"1289",
					"1349",
					"697",
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
			want: map[string][]district.ID{},
		},
		{
			name: "empty1",
			args: args{
				content: GenerateContent([]YnetMessageItem{}),
			},
			want: map[string][]district.ID{},
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
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"1211",
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
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"1289",
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
			want: map[string][]district.ID{},
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
			want: map[string][]district.ID{},
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
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"1349",
				},
			},
		},
		{
			name: "real 1",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"27b246ea-a678-4e96-81bf-5ab905992cdc\",\"pubdate\": \"11:40\",\"title\": \"אזור תעשייה צפוני אשקלון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"27b246ea-a678-4e96-81bf-5ab905992cdc\",\"pubdate\": \"11:40\",\"title\": \"בת הדר\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - דרום\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - צפון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"68",
					"267",
					"6037",
					"6039",
				},
			},
		},
		{
			name: "real 2",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - דרום\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - צפון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: map[string][]district.ID{},
		},
		{
			name: "real 3",
			args: args{
				content: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"f038657b-99e1-48ec-b5c2-3e49c409b3bb\",\"pubdate\": \"11:40\",\"title\": \"בית שקמה\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"f038657b-99e1-48ec-b5c2-3e49c409b3bb\",\"pubdate\": \"11:40\",\"title\": \"גיאה\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"אזור תעשייה הדרומי אשקלון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"זיקים\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"יד מרדכי\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"כרמיה\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"ce68fed6-5c8a-4795-8bce-5f20dcd42d58\",\"pubdate\": \"11:40\",\"title\": \"מבקיעים\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"27b246ea-a678-4e96-81bf-5ab905992cdc\",\"pubdate\": \"11:40\",\"title\": \"אזור תעשייה צפוני אשקלון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"27b246ea-a678-4e96-81bf-5ab905992cdc\",\"pubdate\": \"11:40\",\"title\": \"בת הדר\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - דרום\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"c5ebbc91-51e0-46dd-80de-13618a459987\",\"pubdate\": \"11:40\",\"title\": \"אשקלון - צפון\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
			},
			want: map[string][]district.ID{
				"היכנסו למרחב המוגן": {
					"230",
					"321",
					"50",
					"414",
					"505",
					"666",
					"698",
				},
			},
		},
	}
	s := &SourceYnet{}
	s.Register()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Parse(tt.args.content); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
