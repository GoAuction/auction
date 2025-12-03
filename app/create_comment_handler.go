package app

import (
	"auction/domain"
	"auction/pkg/events"
	"auction/pkg/httperror"
	"context"
	"database/sql"

	"github.com/go-playground/validator/v10"
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

	return &CreateCommentResponse{
		Comment: comment,
	}, nil
}
