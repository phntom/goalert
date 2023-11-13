package district

import "testing"

func TestCityNameCleanFull(t *testing.T) {
	type args struct {
		title string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "tel aviv south",
			args: args{
				title: "תל אביב - דרום העיר ויפו",
			},
			want: "תל_אביב_דרום_העיר_ויפו",
		},
		{
			name: "goosh halav",
			args: args{
				title: "ג'ש - גוש חלב",
			},
			want: "גש_גוש_חלב",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CityNameCleanFull(tt.args.title); got != tt.want {
				t.Errorf("CityNameCleanFull() = %v, want %v", got, tt.want)
			}
		})
	}
}
