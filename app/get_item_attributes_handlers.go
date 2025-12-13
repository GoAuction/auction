package app

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
)

type GetItemAttributesHandler struct {
	repository Repository
}

type GetItemAttributesRequest struct {
	ItemID string `params:"itemId"`
}

type GetItemAttributesResponse struct {
	Attributes []domain.ItemAttribute `json:"attributes"`
}

func NewGetItemAttributesHandler(repository Repository) *GetItemAttributesHandler {
	return &GetItemAttributesHandler{
		repository: repository,
	}
}

func (r *GetItemAttributesHandler) Handle(ctx context.Context, req *GetItemAttributesRequest) (*GetItemAttributesResponse, error) {
	item, err := r.repository.GetItem(ctx, req.ItemID)
	if err != nil {
		return nil, httperror.NotFound("get_item_attributes.index.not_found", "Item not found", nil)
	}

	attributes, err := r.repository.GetItemAttributes(ctx, item.ID)
	if err != nil {
		return nil, httperror.InternalServerError("get_item_attributes.index.internal_server_error", "Failed to get item attributes", err)
	}

	return &GetItemAttributesResponse{
		Attributes: attributes,
	}, nil
}
