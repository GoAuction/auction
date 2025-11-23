package item

import (
	"context"
)

type CreateItemHandler struct {
	repository Repository
}

type CreateItemRequest struct {
}

type CreateItemResponse struct {
	TotpUrl string `json:"totp_url"`
}

func NewCreateItemHandler(repository Repository) *CreateItemHandler {
	return &CreateItemHandler{
		repository: repository,
	}
}

func (e CreateItemHandler) Handle(ctx context.Context, _ *CreateItemRequest) (*CreateItemResponse, error) {
	return &CreateItemResponse{

	}, nil
}