package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/gotd/td/session"
)

func TestStorageMattermost_NilReceiver(t *testing.T) {
	var s *StorageMattermost
	if _, err := s.LoadSession(context.Background()); !errors.Is(err, session.ErrNotFound) {
		t.Errorf("LoadSession on nil receiver: err = %v, want session.ErrNotFound", err)
	}
	if err := s.StoreSession(context.Background(), nil); err == nil {
		t.Error("StoreSession on nil receiver: want error, got nil")
	}
}

func TestStorageMattermost_NilConfigChannel(t *testing.T) {
	s := &StorageMattermost{}
	if _, err := s.LoadSession(context.Background()); !errors.Is(err, errNoConfigChannel) {
		t.Errorf("LoadSession without config channel: err = %v, want errNoConfigChannel", err)
	}
	if err := s.StoreSession(context.Background(), []byte("{}")); !errors.Is(err, errNoConfigChannel) {
		t.Errorf("StoreSession without config channel: err = %v, want errNoConfigChannel", err)
	}
}
