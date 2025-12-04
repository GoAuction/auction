package app

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
	"database/sql"
)

type GetItemImagesHandler struct {
	repository Repository
}

func NewGetItemImagesHandler(repository Repository) *GetItemImagesHandler {
	return &GetItemImagesHandler{
		repository: repository,
	}
}

type GetItemImagesRequest struct {
	ItemID   string `params:"id" validate:"required,uuid"`
	Page     int    `query:"page"`
	PageSize int    `query:"limit"`
}

type GetItemImagesResponse struct {
	Images     []domain.ItemImage `json:"images"`
	Page       int                `json:"page"`
	PageSize   int                `json:"pageSize"`
	TotalItems int                `json:"totalItems"`
	TotalPages int                `json:"totalPages"`
}

func (h *GetItemImagesHandler) Handle(ctx context.Context, req *GetItemImagesRequest) (*GetItemImagesResponse, error) {
	page := max(req.Page, 1)
	pageSize := max(req.PageSize, 10)

	_, err := h.repository.GetItem(ctx, req.ItemID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperror.NotFound("item_images.index.not_found", "Item not found", nil)
		}

		return nil, httperror.InternalServerError("item_images.index.internal_error", "Internal server error", nil)
	}

	images, err := h.repository.GetItemImages(ctx, req.ItemID, page, pageSize)
	if err != nil {
		return nil, httperror.InternalServerError("item_images.index.internal_error", "Internal server error", []string{
			err.Error(),
		})
	}

	totalItems, err := h.repository.CountItemImages(ctx, req.ItemID)
	if err != nil {
		return nil, httperror.InternalServerError("item_images.index.internal_error", "Internal server error", []string{
			err.Error(),
		})
	}

	totalPages := (totalItems + pageSize - 1) / pageSize

	return &GetItemImagesResponse{
		Images:     images,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}
