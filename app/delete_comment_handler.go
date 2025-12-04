package app

import (
	"auction/domain"
	"auction/pkg/events"
	"auction/pkg/httperror"
	"context"
	"time"

	"go.uber.org/zap"
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

	h.publishEvent(ctx, comment)

	return nil, httperror.NoContent("comment.destroy.success", "Comment deleted", nil)
}

func (e DeleteCommentHandler) publishEvent(ctx context.Context, comment domain.ItemComment) {
	eventPayload := events.ItemCommentDeletedPayload{
		ID:        comment.ID,
		ItemID:    comment.ItemID,
		AuthorID:  comment.UserID,
		DeletedAt: time.Now(),
	}

	headers := events.Headers{
		TraceID:       events.GenerateTraceID(),
		CorrelationID: events.GenerateCorrelationID(),
		Service:       "auction",
	}

	event := events.NewEvent(
		events.ItemCommentDeletedEvent,
		events.EventVersionV1,
		eventPayload,
		headers,
	)

	if err := e.eventPublisher.Publish(ctx, events.ItemExchange, event, headers); err != nil {
		zap.L().Error("Failed to publish item.comment.created event",
			zap.String("commentID", comment.ID),
			zap.Error(err),
		)
	}
}
