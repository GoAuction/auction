package app

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
	"database/sql"
)

type GetItemAttributeHandler struct {
	repository Repository
}

type GetItemAttributeRequest struct {
	ItemID      string `params:"itemId"`
	AttributeID string `params:"attributeId"`
}

type GetItemAttributeResponse struct {
	Attribute domain.ItemAttribute `json:"attribute"`
}

func NewGetItemAttributeHandler(repository Repository) *GetItemAttributeHandler {
	return &GetItemAttributeHandler{
		repository: repository,
	}
}

func (r *GetItemAttributeHandler) Handle(ctx context.Context, req *GetItemAttributeRequest) (*GetItemAttributeResponse, error) {
	attribute, err := r.repository.GetItemAttribute(ctx, req.ItemID, req.AttributeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperror.NotFound("get_item_attribute.show.not_found", "Item attribute not found", nil)
		}
		return nil, httperror.InternalServerError("get_item_attribute.show.internal_server_error", "Internal server error", err)
	}

	return &GetItemAttributeResponse{
		Attribute: attribute,
	}, nil
}
