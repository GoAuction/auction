package item

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

type UpdateItemHandler struct {
	repository Repository
}

type UpdateItemRequest struct {
	ItemID       string           `params:"id" validate:"required,uuid"`
	Name         *string          `json:"name,omitempty"`
	Description  *string          `json:"description,omitempty"`
	CurrencyCode *string          `json:"currencyCode,omitempty" validate:"omitempty,iso4217"`
	BidIncrement *decimal.Decimal `json:"bidIncrement,omitempty"`
	ReservePrice *decimal.Decimal `json:"reservePrice,omitempty"`
	BuyoutPrice  *decimal.Decimal `json:"buyoutPrice,omitempty"`
	EndPrice     *decimal.Decimal `json:"endPrice,omitempty"`
	EndDate      *time.Time       `json:"endDate,omitempty"`
	Status       *string          `json:"status,omitempty" validate:"omitempty,oneof=draft active sold cancelled"`
}

type UpdateItemResponse struct {
	Item domain.Item `json:"item"`
}

func NewUpdateItemHandler(repository Repository) *UpdateItemHandler {
	return &UpdateItemHandler{
		repository: repository,
	}
}

func (e UpdateItemHandler) Handle(ctx context.Context, req *UpdateItemRequest) (*UpdateItemResponse, error) {
	userID := ctx.Value("userID").(string)

	validate := validator.New(validator.WithRequiredStructEnabled())

	if err := validate.Struct(req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			return nil, httperror.BadRequest(
				"item.update.validation_failed",
				"Validation failed for the request",
				ve.Error(),
			)
		}

		return nil, httperror.InternalServerError(
			"item.update.validation_error",
			"An unexpected validation error occurred",
			nil,
		)
	}

	item, err := e.repository.GetItem(ctx, req.ItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, httperror.NotFound(
				"item.update.not_found",
				"Item not found",
				nil,
			)
		}

		return nil, httperror.InternalServerError(
			"item.update.failed",
			"Failed to get item",
			nil,
		)
	}

	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Description != nil {
		item.Description = req.Description
	}
	if req.CurrencyCode != nil {
		item.CurrencyCode = *req.CurrencyCode
	}
	if req.BidIncrement != nil {
		item.BidIncrement = req.BidIncrement
	}
	if req.ReservePrice != nil {
		item.ReservePrice = req.ReservePrice
	}
	if req.BuyoutPrice != nil {
		item.BuyoutPrice = req.BuyoutPrice
	}
	if req.EndPrice != nil {
		item.EndPrice = req.EndPrice
	}
	if req.EndDate != nil {
		item.EndDate = *req.EndDate
	}
	if req.Status != nil {
		item.Status = *req.Status
	}

	err = e.repository.Update(ctx, item, userID)
	if err != nil {
		return nil, httperror.InternalServerError(
			"item.update.update_failed",
			"An error occurred while updating the item",
			nil,
		)
	}

	return &UpdateItemResponse{
		Item: item,
	}, nil
}
