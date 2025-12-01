package app

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
)

type GetCategoryHandler struct {
	repository Repository
}

func NewGetCategoryHandler(repository Repository) *GetCategoryHandler {
	return &GetCategoryHandler{
		repository: repository,
	}
}

type GetCategoryRequest struct {
	ID string `query:"id"`
}

type GetCategoryResponse struct {
	Category domain.Category `json:"category"`
}

func (h GetCategoryHandler) Handle(ctx context.Context, req *GetCategoryRequest) (*GetCategoryResponse, error) {
	category, err := h.repository.GetCategoryByID(ctx, req.ID)
	if err != nil {
		return nil, httperror.InternalServerError(
			"category.index.failed",
			"Failed to retrieve categories",
			nil,
		)
	}

	return &GetCategoryResponse{
		Category: category,
	}, nil
}
