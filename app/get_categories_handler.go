package app

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
)

type GetCategoriesHandler struct {
	repository Repository
}

func NewGetCategoriesHandler(repository Repository) *GetCategoriesHandler {
	return &GetCategoriesHandler{
		repository: repository,
	}
}

type GetCategoriesRequest struct {
	Page     int `query:"page"`
	PageSize int `query:"pageSize"`
}

type GetCategoriesResponse struct {
	Categories []domain.Category `json:"categories"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	TotalItems int               `json:"totalItems"`
	TotalPages int               `json:"totalPages"`
}

func (h GetCategoriesHandler) Handle(ctx context.Context, req *GetCategoriesRequest) (*GetCategoriesResponse, error) {
	page := max(req.Page, 1)
	pageSize := max(req.PageSize, 10)

	offset := (page - 1) * pageSize

	categories, err := h.repository.GetCategories(ctx, pageSize, offset)
	if err != nil {
		return nil, httperror.InternalServerError(
			"category.index.failed",
			"Failed to retrieve categories",
			nil,
		)
	}

	totalItems, err := h.repository.CountCategories(ctx)
	if err != nil {
		return nil, httperror.InternalServerError(
			"category.count_categories.failed",
			"Failed to count categories",
			nil,
		)
	}

	totalPages := (totalItems + pageSize - 1) / pageSize

	return &GetCategoriesResponse{
		Categories: categories,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}
