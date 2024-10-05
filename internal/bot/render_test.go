package bot

import (
	"github.com/go-test/deep"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/phntom/goalert/internal/config"
	"github.com/phntom/goalert/internal/district"
	"testing"
)

func TestRender(t *testing.T) {
	type args struct {
		msg  Message
		lang config.Language
	}
	tests := []struct {
		name string
		args args
		want *model.Post
	}{
		{
			name: "rocket alert hebrew simple",
			args: args{
				msg: Message{
					Instructions:  "instructions",
					Category:      "rockets",
					SafetySeconds: 90,
					Cities: []district.ID{
						"999",
					},
				},
				lang: "he",
			},
			want: &model.Post{
				Message: "עין חרוד מאוחד\nתוך 90 שניות היכנסו למרחב המוגן\nצבעאדוםעיןחרודמאוחד #עין_חרוד_מאוחד",
				Metadata: &model.PostMetadata{
					Priority: &model.PostPriority{
						Priority:     model.NewString("urgent"),
						RequestedAck: model.NewBool(true),
					},
				},
			},
		},
		{
			name: "rocket alert english simple",
			args: args{
				msg: Message{
					Instructions:  "instructions",
					Category:      "rockets",
					SafetySeconds: 90,
					Cities: []district.ID{
						"999",
					},
				},
				lang: "en",
			},
			want: &model.Post{
				Message: "Ein Harod\nYou have 90 seconds to seek shelter\nOrefAlarmEinHarod #Ein_Harod",
				Metadata: &model.PostMetadata{
					Priority: &model.PostPriority{
						Priority:     model.NewString("urgent"),
						RequestedAck: model.NewBool(true),
					},
				},
			},
		},
		{
			name: "rocket alert russian simple",
			args: args{
				msg: Message{
					Instructions:  "instructions",
					Category:      "rockets",
					SafetySeconds: 90,
					Cities: []district.ID{
						"999",
					},
				},
				lang: "ru",
			},
			want: &model.Post{
				Message: "Эйн Харод\nУ вас 90 секунд, чтобы найти убежище\nТревогаТылаЭйнХарод #Эйн_Харод",
				Metadata: &model.PostMetadata{
					Priority: &model.PostPriority{
						Priority:     model.NewString("urgent"),
						RequestedAck: model.NewBool(true),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(&tt.args.msg, tt.args.lang)
			if len(tt.want.Attachments()) == 0 {
				//goland:noinspection GoDeprecation
				got.Props = nil
			}
			diff := deep.Equal(got, tt.want)
			if diff != nil {
				t.Errorf("Render() = %v, want %v, diff: %v", got, tt.want, diff)
			}
		})
	}
}
