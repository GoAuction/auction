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
	BuyerID      *string `db:"buyer_id" json:"buyerID"`
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

	Categories []Category `db:"categories" json:"categories"`

	ExtensionThresholdMinutes *int `db:"extension_threshold_minutes" json:"extensionThresholdMinutes,omitempty"`
	ExtensionDurationMinutes  *int `db:"extension_duration_minutes" json:"extensionDurationMinutes,omitempty"`

	Version int `db:"version" json:"version"`
}

const (
	DefaultExtensionThresholdMinutes = 10
	DefaultExtensionDurationMinutes  = 5
)

func (i *Item) GetExtensionThreshold() time.Duration {
	if i.ExtensionThresholdMinutes != nil && *i.ExtensionThresholdMinutes > 0 {
		return time.Duration(*i.ExtensionThresholdMinutes) * time.Minute
	}

	return DefaultExtensionThresholdMinutes * time.Minute
}

func (i *Item) GetExtensionDuration() time.Duration {
	if i.ExtensionDurationMinutes != nil && *i.ExtensionDurationMinutes > 0 {
		return time.Duration(*i.ExtensionDurationMinutes) * time.Minute
	}

	return DefaultExtensionDurationMinutes * time.Minute
}

func (i *Item) ShouldExtendForBid(bidTime time.Time) bool {
	timeUntilEnd := i.EndDate.Sub(bidTime)
	threshold := i.GetExtensionThreshold()

	return timeUntilEnd > 0 && timeUntilEnd <= threshold
}

func (i *Item) CalculateNewEndDate() time.Time {
	return i.EndDate.Add(i.GetExtensionDuration())
}
