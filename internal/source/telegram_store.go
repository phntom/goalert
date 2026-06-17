package source

import (
	"bytes"
	"context"
	"strings"
	"sync"

	"github.com/go-faster/errors"
	"github.com/gotd/td/session"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// mattermostSession persists the Telegram (gotd) session as a JSON post in a
// reserved Mattermost "config" channel, so the bot needs no local volume.
type mattermostSession struct {
	mu           sync.RWMutex
	client       *model.Client4
	channelID    string
	deletePostID string
}

// LoadSession implements gotd session.Storage.
func (s *mattermostSession) LoadSession(ctx context.Context) ([]byte, error) {
	if s == nil || s.channelID == "" {
		return nil, session.ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	postList, r, err := s.client.GetPostsForChannel(ctx, s.channelID, 0, 100, "", false, false)
	if err != nil {
		mlog.Error("telegram session load failed", mlog.Any("response", r), mlog.Err(err))
		return nil, session.ErrNotFound
	}
	postList.SortByCreateAt()
	for _, post := range postList.ToSlice() {
		if strings.HasPrefix(post.Message, "{") {
			s.deletePostID = post.Id
			return []byte(post.Message), nil
		}
	}
	return nil, session.ErrNotFound
}

// StoreSession implements gotd session.Storage: it writes the new session as a
// fresh post and deletes the previous one.
func (s *mattermostSession) StoreSession(ctx context.Context, data []byte) error {
	if s == nil {
		return errors.New("StoreSession called on nil store")
	}
	current, err := s.LoadSession(ctx)
	if err != nil && !errors.Is(err, session.ErrNotFound) {
		return err
	}
	if err == nil && bytes.Equal(data, current) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	post := model.Post{Message: string(data), ChannelId: s.channelID}
	post.AddProp("goalert_config", "1")
	if _, _, err := s.client.CreatePost(ctx, &post); err != nil {
		return err
	}
	if s.deletePostID != "" {
		if _, err := s.client.DeletePost(ctx, s.deletePostID); err != nil {
			return err
		}
	}
	return nil
}
