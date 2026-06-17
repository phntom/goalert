package publish

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/i18n"
	"github.com/phntom/goalert/internal/metrics"
)

const postTimeout = 10 * time.Second

// Mattermost is the live ChatClient backed by the Mattermost REST API.
type Mattermost struct {
	client    *model.Client4
	metrics   *metrics.Metrics
	userID    string
	channels  []Channel
	configChn string // a "config" channel reserved for Telegram session storage
}

// NewMattermost builds a client for the given server domain and bot token.
func NewMattermost(domain, token string, m *metrics.Metrics) *Mattermost {
	c := model.NewAPIv4Client(domain)
	c.SetToken(token)
	return &Mattermost{client: c, metrics: m}
}

// Connect verifies the server is reachable and the token is valid.
func (mm *Mattermost) Connect(ctx context.Context) error {
	props, _, err := mm.client.GetOldClientConfig(ctx, "")
	if err != nil {
		return fmt.Errorf("ping mattermost: %w", err)
	}
	user, _, err := mm.client.GetMe(ctx, "")
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	mm.userID = user.Id
	mlog.Info("connected to mattermost",
		mlog.String("user", user.Username), mlog.String("version", props["Version"]))
	return nil
}

// FindChannels discovers the channels the bot may post to (excluding direct,
// group, off-topic and town-square channels) and assigns each a language.
func (mm *Mattermost) FindChannels(ctx context.Context) error {
	teams, _, err := mm.client.GetTeamsForUser(ctx, mm.userID, "")
	if err != nil {
		return fmt.Errorf("get teams: %w", err)
	}
	var chans []Channel
	for _, team := range teams {
		if team == nil {
			continue
		}
		cs, _, err := mm.client.GetChannelsForTeamForUser(ctx, team.Id, mm.userID, false, "")
		if err != nil {
			return fmt.Errorf("get channels for team %s: %w", team.Id, err)
		}
		for _, ch := range cs {
			if ch == nil || ch.IsGroupOrDirect() || ch.Name == "off-topic" || ch.Name == "town-square" {
				continue
			}
			if ch.Name == "config" {
				mm.configChn = ch.Id // reserved for Telegram session storage, not an alert target
				continue
			}
			chans = append(chans, Channel{ID: ch.Id, Name: ch.Name, Lang: LanguageOf(ch.DisplayName)})
		}
	}
	if len(chans) == 0 {
		return errors.New("bot is not a member of any channel; invite it and retry")
	}
	mm.channels = chans
	mlog.Info("joined channels", mlog.Int("count", len(chans)))
	return nil
}

// Channels implements ChatClient.
func (mm *Mattermost) Channels() []Channel { return mm.channels }

// Client exposes the underlying API client (for Telegram session storage).
func (mm *Mattermost) Client() *model.Client4 { return mm.client }

// ConfigChannelID returns the reserved "config" channel id, or "" if none.
func (mm *Mattermost) ConfigChannelID() string { return mm.configChn }

// Broadcast posts plain text to every channel of the given language.
func (mm *Mattermost) Broadcast(lang i18n.Language, text string) error {
	var firstErr error
	for _, ch := range mm.channels {
		if ch.Lang != lang {
			continue
		}
		if _, err := mm.CreatePost(ch.ID, &model.Post{Message: text}); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// PostToNamed posts plain text to the channel with the given name.
func (mm *Mattermost) PostToNamed(name, text string) error {
	for _, ch := range mm.channels {
		if ch.Name == name {
			_, err := mm.CreatePost(ch.ID, &model.Post{Message: text})
			return err
		}
	}
	return fmt.Errorf("channel %q not found", name)
}

// CreatePost implements ChatClient.
func (mm *Mattermost) CreatePost(channelID string, post *model.Post) (string, error) {
	post.ChannelId = channelID
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	defer cancel()
	res, _, err := mm.client.CreatePost(ctx, post)
	if err != nil {
		return "", err
	}
	mm.metrics.PostsOK.Inc()
	return res.Id, nil
}

// PatchPost implements ChatClient: it blanks the trigger text and replaces the
// post body with the card props.
func (mm *Mattermost) PatchPost(postID string, props map[string]any) error {
	si := model.StringInterface(props)
	patch := &model.PostPatch{Message: model.NewString(""), Props: &si}
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	defer cancel()
	if _, _, err := mm.client.PatchPost(ctx, postID, patch); err != nil {
		mm.metrics.PatchFail.Inc()
		return err
	}
	mm.metrics.PatchesOK.Inc()
	return nil
}

// AddReaction implements ChatClient.
func (mm *Mattermost) AddReaction(postID, emoji string) error {
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	defer cancel()
	_, _, err := mm.client.SaveReaction(ctx, &model.Reaction{
		UserId: mm.userID, PostId: postID, EmojiName: emoji,
	})
	return err
}
