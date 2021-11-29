package main

import "testing"

func TestGenerateMessageFromAlert(t *testing.T) {
	type args struct {
		alertContent []byte
		announced    map[string]bool
	}
	persistentAnnounced := make(map[string]bool)
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty1",
			args: args{
				alertContent: []byte(""),
				announced:    make(map[string]bool),
			},
			want: "",
		},
		{
			name: "empty2",
			args: args{
				alertContent: []byte(" []"),
				announced:    make(map[string]bool),
			},
			want: "",
		},
		{
			name: "empty3",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": []}});"),
				announced:    make(map[string]bool),
			},
			want: "",
		},
		{
			name: "empty4",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": ]}});"),
				announced:    make(map[string]bool),
			},
			want: "",
		},
		{
			name: "empty5",
			args: args{
				alertContent: []byte("[]"),
				announced:    make(map[string]bool),
			},
			want: "",
		},
		{
			name: "sanity",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"6c38fbbd-d8c0-40e4-bfe0-a17b1657203e\",\"pubdate\": \"20:53\",\"title\": \"שדה ניצן\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"6c38fbbd-d8c0-40e4-bfe0-a17b1657203e\",\"pubdate\": \"20:53\",\"title\": \"תלמי אליהו\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"8a299260-c12c-4e2e-adc7-671b325474a3\",\"pubdate\": \"20:53\",\"title\": \"צוחר ואוהד\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"56d79011-549d-4862-85f5-58a2240c12a7\",\"pubdate\": \"20:53\",\"title\": \"מבטחים עמיעוז ישע\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    persistentAnnounced,
			},
			want: " שדה ניצן, תלמי אליהו, צוחר ואוהד, מבטחים עמיעוז ישע\n#שדה_ניצן #תלמי_אליהו #צוחר_ואוהד #מבטחים_עמיעוז_ישע היכנסו למרחב המוגן\nצבעאדוםשדהניצן צבעאדוםתלמיאליהו צבעאדוםצוחרואוהד צבעאדוםמבטחיםעמיעוזישע",
		},
		{
			name: "same again",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"6c38fbbd-d8c0-40e4-bfe0-a17b1657203e\",\"pubdate\": \"20:53\",\"title\": \"שדה ניצן\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"6c38fbbd-d8c0-40e4-bfe0-a17b1657203e\",\"pubdate\": \"20:53\",\"title\": \"תלמי אליהו\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"8a299260-c12c-4e2e-adc7-671b325474a3\",\"pubdate\": \"20:53\",\"title\": \"צוחר ואוהד\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"56d79011-549d-4862-85f5-58a2240c12a7\",\"pubdate\": \"20:53\",\"title\": \"מבטחים עמיעוז ישע\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    persistentAnnounced,
			},
			want: "",
		},
		{
			name: "old with new",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"6c38fbbd-d8c0-40e4-bfe0-a17b1657203e\",\"pubdate\": \"20:53\",\"title\": \"שדה ניצן\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"6c38fbbd-d8c0-40e4-bfe0-a17b1657203e\",\"pubdate\": \"20:53\",\"title\": \"תלמי אליהו\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"11111111-c12c-4e2e-adc7-671b325474a3\",\"pubdate\": \"20:53\",\"title\": \"צוחר ואוהד\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    persistentAnnounced,
			},
			want: " צוחר ואוהד\n#צוחר_ואוהד היכנסו למרחב המוגן\nצבעאדוםצוחרואוהד",
		},
		{
			name: "only old",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"11111111-c12c-4e2e-adc7-671b325474a3\",\"pubdate\": \"20:53\",\"title\": \"צוחר ואוהד\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    persistentAnnounced,
			},
			want: "",
		},
		{
			name: "important city solo",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"11111111-1111-1111-1111-111111111111\",\"pubdate\": \"11:11\",\"title\": \"תל אביב - מרכז העיר\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    make(map[string]bool),
			},
			want: "# תל אביב - מרכז העיר\n#תל_אביב_מרכז_העיר היכנסו למרחב המוגן\nצבעאדוםתלאביבמרכזהעיר",
		},
		{
			name: "important city and less important city",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"22222222-1111-1111-1111-111111111111\",\"pubdate\": \"11:12\",\"title\": \"באר שבע - מערב\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"11111111-1111-1111-1111-111111111111\",\"pubdate\": \"11:11\",\"title\": \"תל אביב - מרכז העיר\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    make(map[string]bool),
			},
			want: "# באר שבע - מערב, תל אביב - מרכז העיר\n#באר_שבע_מערב #תל_אביב_מרכז_העיר היכנסו למרחב המוגן\nצבעאדוםבארשבעמערב צבעאדוםתלאביבמרכזהעיר",
		},
		{
			name: "less important city solo",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"22222222-1111-1111-1111-111111111111\",\"pubdate\": \"11:12\",\"title\": \"באר שבע - מערב\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    make(map[string]bool),
			},
			want: "## באר שבע - מערב\n#באר_שבע_מערב היכנסו למרחב המוגן\nצבעאדוםבארשבעמערב",
		},
		{
			name: "only an exercise",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"11111111-1111-1111-1111-111111111111\",\"pubdate\": \"11:11\",\"title\": \"תל אביב - מרכז העיר\",\"description\": \"ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    make(map[string]bool),
			},
			want: " ~~תל אביב - מרכז העיר~~\n ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן\n",
		},
		{
			name: "exercise and real rocket 1",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"11111111-1111-1111-1111-111111111111\",\"pubdate\": \"11:11\",\"title\": \"תל אביב - מרכז העיר\",\"description\": \"ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"22222222-1111-1111-1111-111111111111\",\"pubdate\": \"11:12\",\"title\": \"באר שבע - מערב\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    make(map[string]bool),
			},
			want: "## ~~תל אביב - מרכז העיר~~, באר שבע - מערב\n #באר_שבע_מערב היכנסו למרחב המוגן\n צבעאדוםבארשבעמערב",
		},
		{
			name: "exercise and real rocket 2",
			args: args{
				alertContent: []byte("jsonCallback({\"alerts\": {\"items\": [{\"item\": {\"guid\": \"22222222-1111-1111-1111-111111111111\",\"pubdate\": \"11:12\",\"title\": \"באר שבע - מערב\",\"description\": \"היכנסו למרחב המוגן\",\"link\": \"\"}},{\"item\": {\"guid\": \"11111111-1111-1111-1111-111111111111\",\"pubdate\": \"11:11\",\"title\": \"תל אביב - מרכז העיר\",\"description\": \"ברגעים אלה נשמעת אזעקה במסגרת תרגיל העורף הלאומי תרגלו כניסה למרחב המוגן\",\"link\": \"\"}}]}});"),
				announced:    make(map[string]bool),
			},
			want: "## באר שבע - מערב, ~~תל אביב - מרכז העיר~~\n#באר_שבע_מערב  היכנסו למרחב המוגן\nצבעאדוםבארשבעמערב ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateMessageFromAlert(tt.args.alertContent, tt.args.announced); got != tt.want {
				t.Errorf("GenerateMessageFromAlert() = %v, want %v", got, tt.want)
			}
		})
	}
}
