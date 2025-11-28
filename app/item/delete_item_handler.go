package item

import (
	"auction/pkg/httperror"
	"context"
	"database/sql"
)

type DeleteItemHandler struct {
	repository Repository
}

func NewDeleteItemHandler(repository Repository) *DeleteItemHandler {
	return &DeleteItemHandler{
		repository: repository,
	}
}

type DeleteItemRequest struct {
	ItemID string `params:"id" validate:"required,uuid"`
}

type DeleteItemResponse struct {
}

func (h DeleteItemHandler) Handle(ctx context.Context, req *DeleteItemRequest) (*DeleteItemResponse, error) {
	userID := ctx.Value("UserID").(string)

	_, err := h.repository.GetUserItem(ctx, req.ItemID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperror.NotFound(
				"item.destroy.notfound",
				"Item not found",
				nil,
			)
		}
		return nil, httperror.InternalServerError(
			"item.destroy.failed",
			"Failed to retrieve items",
			nil,
		)
	}

	err = h.repository.DeleteItem(ctx, req.ItemID, userID)
	if err != nil {
		return nil, httperror.InternalServerError(
			"item.destroy.failed",
			"Failed to retrieve items",
			nil,
		)
	}

	return nil, httperror.NoContent(
		"item.destroy.success",
		"Item deleted successfully",
		nil,
	)
}
