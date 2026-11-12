package sources

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/go-faster/errors"
	"github.com/gotd/td/session"
	"github.com/mattermost/mattermost/server/public/model"
)

// errNoConfigChannel is returned when the Mattermost channel that stores the
// Telegram session is not available.
var errNoConfigChannel = errors.New("telegram config channel is not available")

type StorageMattermost struct {
	mux           sync.RWMutex
	client        *model.Client4
	configChannel *model.Channel
}

// loadSession fetches the current session bytes from the configuration
// channel, returning them together with the ID of the post they were read
// from.
func (s *StorageMattermost) loadSession(ctx context.Context) (data []byte, postID string, err error) {
	if s.configChannel == nil {
		return nil, "", errNoConfigChannel
	}
	postList, r, err := s.client.GetPostsForChannel(ctx, s.configChannel.Id, 0, 100, "", false, false)
	if err != nil {
		slog.Error(
			"failed to fetch posts from configuration channel",
			"configChannel", s.configChannel,
			"response", r,
			"error", err,
		)
		return nil, "", session.ErrNotFound
	}
	postList.SortByCreateAt()
	for _, post := range postList.ToSlice() {
		if strings.HasPrefix(post.Message, "{") {
			return []byte(post.Message), post.Id, nil
		}
	}

	return nil, "", session.ErrNotFound
}

// LoadSession loads session from memory.
func (s *StorageMattermost) LoadSession(ctx context.Context) ([]byte, error) {
	if s == nil {
		return nil, session.ErrNotFound
	}
	data, _, err := s.loadSession(ctx)
	return data, err
}

// StoreSession stores session to memory.
func (s *StorageMattermost) StoreSession(ctx context.Context, data []byte) error {
	if s == nil {
		return errors.New("StoreSession called on StorageMattermost(nil)")
	}
	currentData, currentPostID, err := s.loadSession(ctx)
	if err != nil {
		return err
	}
	if bytes.Equal(data, currentData) {
		return nil
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	post := model.Post{
		Message:   string(data),
		ChannelId: s.configChannel.Id,
	}
	post.AddProp("goalert_config", "1")
	_, _, err = s.client.CreatePost(ctx, &post)
	if err != nil {
		return err
	}
	if currentPostID != "" {
		_, err = s.client.DeletePost(ctx, currentPostID)
		if err != nil {
			return err
		}
	}
	return nil
}
