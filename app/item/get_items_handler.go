package item

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
)

type GetItemsHandler struct {
	repository Repository
}

func NewGetItemsHandler(repository Repository) *GetItemsHandler {
	return &GetItemsHandler{
		repository: repository,
	}
}

type GetItemsRequest struct {
	Page     int `query:"page"`
	PageSize int `query:"pageSize"`
}

type GetItemsResponse struct {
	Items      []domain.Item `json:"items"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalItems int           `json:"totalItems"`
	TotalPages int           `json:"totalPages"`
}

func (h GetItemsHandler) Handle(ctx context.Context, req *GetItemsRequest) (*GetItemsResponse, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}

	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	items, err := h.repository.GetItems(ctx, pageSize, offset)
	if err != nil {
		return nil, httperror.InternalServerError(
			"item.index.failed",
			"Failed to retrieve items",
			nil,
		)
	}

	totalItems, err := h.repository.CountItems(ctx)
	if err != nil {
		return nil, httperror.InternalServerError(
			"item.count_items.failed",
			"Failed to count items",
			nil,
		)
	}

	totalPages := (totalItems + pageSize - 1) / pageSize

	return &GetItemsResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}
