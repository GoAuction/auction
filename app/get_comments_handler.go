package app

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
)

type GetCommentsHandler struct {
	repository Repository
}

func NewGetCommentsHandler(repository Repository) *GetCommentsHandler {
	return &GetCommentsHandler{
		repository: repository,
	}
}

type GetCommentsRequest struct {
	ID       string `params:"id"`
	Page     int    `query:"page"`
	PageSize int    `query:"limit"`
}

type GetCommentsResponse struct {
	Comments   []domain.ItemComment `json:"comments"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
	TotalItems int                  `json:"totalItems"`
	TotalPages int                  `json:"totalPages"`
}

func (h *GetCommentsHandler) Handle(ctx context.Context, req *GetCommentsRequest) (*GetCommentsResponse, error) {
	page := max(req.Page, 1)
	pageSize := max(req.PageSize, 10)

	comments, err := h.repository.GetItemCommentsByItemID(ctx, req.ID, page, pageSize)
	if err != nil {
		return nil, httperror.InternalServerError(
			"comments.index.failed",
			"Comments repository failed to retrieve comments",
			nil,
		)
	}

	totalItems, err := h.repository.CountItemComments(ctx, req.ID)
	if err != nil {
		return nil, httperror.InternalServerError(
			"comments.count_comments.failed",
			"Failed to count comments",
			nil,
		)
	}

	totalPages := (totalItems + pageSize - 1) / pageSize

	return &GetCommentsResponse{
		Comments:   comments,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}
