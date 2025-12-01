package app

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
	"database/sql"
	"errors"
)

type GetItemHandler struct {
	repository Repository
}

func NewGetItemHandler(repository Repository) *GetItemHandler {
	return &GetItemHandler{
		repository: repository,
	}
}

type GetItemRequest struct {
	ItemID string `params:"id"`
}

type GetItemResponse struct {
	Item domain.Item `json:"item"`
}

func (h GetItemHandler) Handle(ctx context.Context, req *GetItemRequest) (*GetItemResponse, error) {
	item, err := h.repository.GetItem(ctx, req.ItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, httperror.NotFound(
				"item.show.not_found",
				"Item not found",
				nil,
			)
		}

		return nil, httperror.InternalServerError(
			"item.show.failed",
			"Failed to retrieve items",
			nil,
		)
	}

	return &GetItemResponse{
		Item: item,
	}, nil
}
