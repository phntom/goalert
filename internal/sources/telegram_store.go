package sources

import (
	"bytes"
	"context"
	"github.com/gotd/td/session"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"strings"
	"sync"

	"github.com/go-faster/errors"
)

type StorageMattermost struct {
	mux           sync.RWMutex
	client        *model.Client4
	configChannel *model.Channel
	deletePostID  string
}

// LoadSession loads session from memory.
func (s *StorageMattermost) LoadSession(ctx context.Context) ([]byte, error) {
	if s == nil {
		return nil, session.ErrNotFound
	}
	s.mux.RLock()
	defer s.mux.RUnlock()

	postList, r, err := s.client.GetPostsForChannel(ctx, s.configChannel.Id, 0, 100, "", false, false)
	if err != nil {
		mlog.Error(
			"failed to fetch posts from configuration channel",
			mlog.Any("configChannel", s.configChannel),
			mlog.Any("response", r),
			mlog.Err(err),
		)
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

// StoreSession stores session to memory.
func (s *StorageMattermost) StoreSession(ctx context.Context, data []byte) error {
	if s == nil {
		return errors.New("StoreSession called on StorageMattermost(nil)")
	}
	currentData, err := s.LoadSession(ctx)
	if err != nil {
		return err
	}
	if bytes.Equal(data, currentData) {
		return nil
	} else {
		s.mux.Lock()
		defer s.mux.Unlock()
		post := model.Post{
			Message:   string(data),
			ChannelId: s.configChannel.Id,
		}
		post.AddProp("goalert_config", "1")
		_, _, err := s.client.CreatePost(ctx, &post)
		if err != nil {
			return err
		}
		if s.deletePostID != "" {
			_, err := s.client.DeletePost(ctx, s.deletePostID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
