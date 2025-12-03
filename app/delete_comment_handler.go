package app

import (
	"auction/pkg/events"
	"auction/pkg/httperror"
	"context"
)

type DeleteCommentHandler struct {
	repository     Repository
	eventPublisher events.Publisher
}

type DeleteCommentRequest struct {
	ItemID    string `params:"itemId" validate:"required,uuid"`
	CommentID string `params:"commentId" validate:"required,uuid"`
}

type DeleteCommentResponse struct {
}

func NewDeleteCommentHandler(repository Repository, eventPublisher events.Publisher) *DeleteCommentHandler {
	return &DeleteCommentHandler{
		repository:     repository,
		eventPublisher: eventPublisher,
	}
}

func (h *DeleteCommentHandler) Handle(ctx context.Context, req *DeleteCommentRequest) (*DeleteCommentResponse, error) {
	comment, err := h.repository.GetCommentByID(ctx, req.CommentID)
	if err != nil {
		return nil, err
	}

	userID := ctx.Value("UserID").(string)

	if comment.UserID != userID {
		return nil, httperror.Forbidden("comment.destroy", "Cannot delete comment with parent", nil)
	}

	err = h.repository.DeleteComment(ctx, req.CommentID)
	if err != nil {
		return nil, err
	}

	return nil, httperror.NoContent("comment.destroy.success", "Comment deleted", nil)
}
