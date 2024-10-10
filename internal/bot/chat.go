package bot

import (
	"context"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func executeSubmitPost(b *Bot, post *model.Post, message *Message, channel *model.Channel) (*model.Post, error) {
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	defer cancel()
	result, response, err := b.Client.CreatePost(ctx, post)
	if err != nil {
		mlog.Error("failed creating post",
			mlog.Err(err),
			mlog.Any("post", post),
			mlog.Any("response", response),
		)
		return nil, err
	}
	message.PostMutex.Lock()
	message.PostIDs = append(message.PostIDs, result.Id)
	message.ChannelsPosted = append(message.ChannelsPosted, channel)
	message.PostMutex.Unlock()
	b.Monitoring.SuccessfulPosts.Inc()
	return result, err
}

func executePatchPost(b *Bot, post *model.Post, postID string) {
	//goland:noinspection GoDeprecation
	patch := model.PostPatch{
		Message: model.NewString(""),
		Props:   &post.Props,
	}
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	_, response, err := b.Client.PatchPost(ctx, postID, &patch)
	cancel()
	if err != nil {
		mlog.Error("failed patching post",
			mlog.Err(err),
			mlog.Any("postID", postID),
			mlog.Any("patch", patch),
			mlog.Any("response", response),
		)
		b.Monitoring.FailedPatches.Inc()
	} else {
		b.Monitoring.SuccessfulPatches.Inc()
	}
}

func executeAddReaction(b *Bot, post *model.Post, emoji string) {
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	defer cancel()
	_, _, err := b.Client.SaveReaction(ctx, &model.Reaction{
		UserId:    b.userId,
		PostId:    post.Id,
		EmojiName: emoji,
	})
	if err != nil {
		mlog.Error("failed adding reaction",
			mlog.Err(err),
			mlog.Any("emoji", emoji),
			mlog.Any("postID", post),
		)
	}
}
