package app

import (
	"auction/domain"
	"auction/pkg/events"
	"auction/pkg/httperror"
	"context"
	"database/sql"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type CreateCommentHandler struct {
	repository     Repository
	eventPublisher events.Publisher
}

func NewCreateCommentHandler(repository Repository, eventPublisher events.Publisher) *CreateCommentHandler {
	return &CreateCommentHandler{
		repository:     repository,
		eventPublisher: eventPublisher,
	}
}

type CreateCommentRequest struct {
	ItemID   string  `params:"id" validate:"required,uuid"`
	Comment  string  `json:"content" validate:"required"`
	ParentID *string `json:"parentId,omitempty" validate:"omitempty,uuid"`
}

type CreateCommentResponse struct {
	Comment domain.ItemComment `json:"comment"`
}

func (c *CreateCommentHandler) Handle(ctx context.Context, req *CreateCommentRequest) (*CreateCommentResponse, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	if err := validate.Struct(req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			return nil, httperror.BadRequest(
				"comments.create.validation_failed",
				"Validation failed for the request",
				ve.Error(),
			)
		}

		return nil, httperror.InternalServerError(
			"comments.create.validation_error",
			"An unexpected validation error occurred",
			nil,
		)
	}

	item, err := c.repository.GetItem(ctx, req.ItemID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperror.NotFound("comments.create.not_found", "Item not found", nil)
		}

		return nil, httperror.InternalServerError("comments.create.internal_error", "Failed to get item", err)
	}

	userID := ctx.Value("UserID").(string)

	comment, err := c.repository.CreateComment(ctx, item.ID, req.Comment, userID, req.ParentID)
	if err != nil {
		return nil, httperror.InternalServerError("comments.create.internal_error", "Failed to create comment", err)
	}

	c.publishEvent(ctx, comment)

	return &CreateCommentResponse{
		Comment: comment,
	}, nil
}

func (e CreateCommentHandler) publishEvent(ctx context.Context, comment domain.ItemComment) {
	eventPayload := events.ItemCommentCreatedPayload{
		ID:        comment.ID,
		ItemID:    comment.ItemID,
		AuthorID:  comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
	}

	headers := events.Headers{
		TraceID:       events.GenerateTraceID(),
		CorrelationID: events.GenerateCorrelationID(),
		Service:       "auction",
	}

	event := events.NewEvent(
		events.ItemCommentCreatedEvent,
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
