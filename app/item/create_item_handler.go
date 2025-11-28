package item

import (
	"auction/domain"
	"auction/pkg/httperror"
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

type CreateItemHandler struct {
	repository Repository
}

type CreateItemRequest struct {
	Name         string           `json:"name" validate:"required" db:"name"`
	Description  *string          `json:"description" db:"description"`
	CurrencyCode string           `json:"currencyCode" validate:"required,iso4217" db:"currency_code"`
	SellerID     string           `json:"sellerID,omitempty" db:"seller_id"`
	StartPrice   decimal.Decimal  `json:"startPrice" validate:"required" db:"start_price"`
	BidIncrement *decimal.Decimal `json:"bidIncrement" validate:"required" db:"bid_increment"`
	ReservePrice *decimal.Decimal `json:"reservePrice,omitempty" db:"reserve_price"`
	BuyoutPrice  *decimal.Decimal `json:"buyoutPrice,omitempty" db:"buyout_price"`
	EndPrice     *decimal.Decimal `json:"endPrice,omitempty" db:"end_price"`
	StartDate    time.Time        `json:"startDate" validate:"required" db:"start_date"`
	EndDate      time.Time        `json:"endDate" validate:"required,gtfield=StartDate" db:"end_date"`
	Status       string           `json:"status,omitempty" validate:"required,oneof=draft active sold cancelled" db:"status"`
}

type CreateItemResponse struct {
	Item domain.Item `json:"item"`
}

func NewCreateItemHandler(repository Repository) *CreateItemHandler {
	return &CreateItemHandler{
		repository: repository,
	}
}

func (e CreateItemHandler) Handle(ctx context.Context, req *CreateItemRequest) (*CreateItemResponse, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	if err := validate.Struct(req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			return nil, httperror.BadRequest(
				"item.create.validation_failed",
				"Validation failed for the request",
				ve.Error(),
			)
		}

		return nil, httperror.InternalServerError(
			"item.create.validation_error",
			"An unexpected validation error occurred",
			nil,
		)
	}

	userID := ctx.Value("UserID").(string)
	req.SellerID = userID

	item, err := e.repository.Create(ctx, req)
	if err != nil {
		return nil, httperror.InternalServerError(
			"item.create.create_failed",
			"An error occurred while creating the item",
			nil,
		)
	}

	return &CreateItemResponse{
		Item: item,
	}, nil
}
