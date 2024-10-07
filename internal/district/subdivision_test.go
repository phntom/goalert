package district

import (
	"github.com/phntom/goalert/internal/config"
	"strings"
	"testing"
)

func TestGetCity(t *testing.T) {
	type args struct {
		id   ID
		lang config.Language
	}
	GetDistricts()
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		{
			name: "פרדס חנה-כרכור",
			args: args{
				id:   "1067",
				lang: "he",
			},
			want:  "פרדס חנה-כרכור",
			want1: "",
		},
		{
			name: "תל אביב - מרכז העיר",
			args: args{
				id:   "6031",
				lang: "he",
			},
			want:  "תל אביב",
			want1: "מרכז העיר",
		},
		{
			name: "באר שבע - מזרח",
			args: args{
				id:   "6005",
				lang: "he",
			},
			want:  "באר שבע",
			want1: "מזרח",
		},
		{
			name: "Acco - Industrial Zone",
			args: args{
				id:   "1017",
				lang: "en",
			},
			want:  "Acre",
			want1: "Industrial Zone",
		},
		{
			name: "עכו - אזור תעשייה",
			args: args{
				id:   "1017",
				lang: "he",
			},
			want:  "עכו",
			want1: "אזור תעשייה",
		},
		{
			name: "Haifa - Ramot HaCarmel and Neveh Sha'anan",
			args: args{
				id:   "6017",
				lang: "en",
			},
			want:  "Haifa",
			want1: "Ramot HaCarmel and Neveh Sha'anan",
		},
		{
			name: "אזור תעשייה צמח",
			args: args{
				id:   "67",
				lang: "he",
			},
			want:  "צמח",
			want1: "אזור תעשייה",
		},
		{
			name: "Bar Lev Industrial Zone",
			args: args{
				id:   "46",
				lang: "en",
			},
			want:  "Bar Lev",
			want1: "Industrial Zone",
		},
		{
			name: "Промзона Бар-Лев",
			args: args{
				id:   "46",
				lang: "ru",
			},
			want:  "Бар-Лев",
			want1: "Промышленная зона",
		},
		{
			name: "מועצה אזורית מבואות חרמון",
			args: args{
				id:   "1355",
				lang: "he",
			},
			want:  "מבואות חרמון",
			want1: "מרכז אזורי",
		},
		{
			name: "סינמה סיטי גלילות",
			args: args{
				id:   "1379",
				lang: "he",
			},
			want:  "פי גלילות",
			want1: "סינמה סיטי",
		},
		{
			name: "אזור תעשייה דימונה",
			args: args{
				id:   "49",
				lang: "he",
			},
			want:  "דימונה",
			want1: "אזור תעשייה",
		},
		{
			name: "בית ספר אורט בנימינה",
			args: args{
				id:   "216",
				lang: "he",
			},
			want:  "בנימינה",
			want1: "בית ספר אורט",
		},
		{
			name: "גבים, מכללת ספיר",
			args: args{
				id:   "283",
				lang: "he",
			},
			want:  "גבים",
			want1: "גבים!מכללת ספיר",
		},
		{
			name: "חוף גולן, צאלון",
			args: args{
				id:   "1419",
				lang: "he",
			},
			want:  "חוף גולן",
			want1: "חוף גולן!צאלון",
		},
		{
			name: "מלונות ים המלח מרכז",
			args: args{
				id:   "751",
				lang: "he",
			},
			want:  "ים המלח",
			want1: "מלונות מרכז",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetCity(tt.args.id, tt.args.lang)
			got2 := strings.Join(got1, "!")
			if got != tt.want {
				t.Errorf("GetCity() got = %v, want %v", got, tt.want)
			}
			if got2 != tt.want1 {
				t.Errorf("GetCity() got1 = %v, want %v", got2, tt.want1)
			}
		})
	}
}
