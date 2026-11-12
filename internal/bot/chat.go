package bot

import (
	"context"
	"errors"
	"log/slog"

	"github.com/mattermost/mattermost/server/public/model"
)

// errMattermostAPINotInitialized is returned when the Mattermost API seam was
// never initialized, which is only possible in tests that skip Register.
var errMattermostAPINotInitialized = errors.New("mattermost API is not initialized")

func executeSubmitPost(b *Bot, post *model.Post, message *Message, channel *model.Channel) (*model.Post, error) {
	if b.apiSubmitPost == nil {
		slog.Error("mattermost API is not initialized")
		return nil, errMattermostAPINotInitialized
	}
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	defer cancel()
	result, response, err := b.apiSubmitPost(ctx, post)
	if err != nil {
		slog.Error("failed creating post",
			"error", err,
			"post", post,
			"response", response,
		)
		return nil, err
	}
	if message != nil {
		message.PostMutex.Lock()
		message.PostIDs = append(message.PostIDs, result.Id)
		message.ChannelsPosted = append(message.ChannelsPosted, channel)
		message.PostMutex.Unlock()
	}
	b.Monitoring.SuccessfulPosts.Inc()
	return result, err
}

func executePatchPost(b *Bot, post *model.Post, postID string) {
	if b.apiPatchPost == nil {
		slog.Error("mattermost API is not initialized")
		b.Monitoring.FailedPatches.Inc()
		return
	}
	//goland:noinspection GoDeprecation
	patch := model.PostPatch{
		Message: model.NewPointer(""),
		Props:   &post.Props,
	}
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	_, response, err := b.apiPatchPost(ctx, postID, &patch)
	cancel()
	if err != nil {
		slog.Error("failed patching post",
			"error", err,
			"postID", postID,
			"patch", patch,
			"response", response,
		)
		b.Monitoring.FailedPatches.Inc()
	} else {
		b.Monitoring.SuccessfulPatches.Inc()
	}
}

func executeAddReaction(b *Bot, post *model.Post, emoji string) {
	if b.apiSaveReaction == nil {
		slog.Error("mattermost API is not initialized")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	defer cancel()
	_, _, err := b.apiSaveReaction(ctx, &model.Reaction{
		UserId:    b.userId,
		PostId:    post.Id,
		EmojiName: emoji,
	})
	if err != nil {
		slog.Error("failed adding reaction",
			"error", err,
			"emoji", emoji,
			"postID", post,
		)
	}
}
