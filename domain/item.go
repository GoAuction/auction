package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Item struct {
	ID           string  `db:"id" json:"id"`
	Name         string  `db:"name" json:"name"`
	Description  *string `db:"description" json:"description"`
	SellerID     string  `db:"seller_id" json:"sellerID"`
	CurrencyCode string  `db:"currency_code" json:"currencyCode"`

	StartPrice   decimal.Decimal  `db:"start_price" json:"startPrice"`
	CurrentPrice decimal.Decimal  `db:"current_price" json:"currentPrice"`
	BidIncrement *decimal.Decimal `db:"bid_increment" json:"bidIncrement"`

	ReservePrice *decimal.Decimal `db:"reserve_price" json:"reservePrice"`
	BuyoutPrice  *decimal.Decimal `db:"buyout_price" json:"buyoutPrice"`
	EndPrice     *decimal.Decimal `db:"end_price" json:"endPrice"`

	StartDate time.Time `db:"start_date" json:"startDate"`
	EndDate   time.Time `db:"end_date" json:"endDate"`

	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}
