package sources

import "testing"

func Test_calculatePubTime(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"10:07",
			args{
				id: "133449412450000000",
			},
			"10:07",
		},
		{
			"19:56",
			args{
				id: "133448902120000000",
			},
			"19:56",
		},
		{
			"16:02",
			args{
				id: "133467769470000000",
			},
			"16:02",
		},
		{
			"15:20",
			args{
				id: "133467744260000000",
			},
			"15:20",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculatePubTime(tt.args.id); got != tt.want {
				t.Errorf("calculatePubTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
