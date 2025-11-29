package events

import (
	"time"

	"github.com/shopspring/decimal"
)

// Domain constants
const (
	ItemDomain = "item"
	ItemExchange = "auction.item"
)

// Event names
const (
	ItemCreatedEvent = "item.created"
	ItemUpdatedEvent = "item.updated"
	ItemDeletedEvent = "item.deleted"
)

// Event versions
const (
	EventVersionV1 = "v1"
)

// ItemCreatedPayload represents the payload for item.created event
type ItemCreatedPayload struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  *string          `json:"description"`
	SellerID     string           `json:"sellerId"`
	CurrencyCode string           `json:"currencyCode"`
	StartPrice   decimal.Decimal  `json:"startPrice"`
	CurrentPrice decimal.Decimal  `json:"currentPrice"`
	BidIncrement *decimal.Decimal `json:"bidIncrement"`
	ReservePrice *decimal.Decimal `json:"reservePrice"`
	BuyoutPrice  *decimal.Decimal `json:"buyoutPrice"`
	StartDate    time.Time        `json:"startDate"`
	EndDate      time.Time        `json:"endDate"`
	Status       string           `json:"status"`
	CreatedAt    time.Time        `json:"createdAt"`
}

// ItemUpdatedPayload represents the payload for item.updated event
type ItemUpdatedPayload struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  *string          `json:"description"`
	CurrencyCode string           `json:"currencyCode"`
	StartPrice   decimal.Decimal  `json:"startPrice"`
	CurrentPrice decimal.Decimal  `json:"currentPrice"`
	BidIncrement *decimal.Decimal `json:"bidIncrement"`
	ReservePrice *decimal.Decimal `json:"reservePrice"`
	BuyoutPrice  *decimal.Decimal `json:"buyoutPrice"`
	EndPrice     *decimal.Decimal `json:"endPrice"`
	StartDate    time.Time        `json:"startDate"`
	EndDate      time.Time        `json:"endDate"`
	Status       string           `json:"status"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

// ItemDeletedPayload represents the payload for item.deleted event
type ItemDeletedPayload struct {
	ID        string    `json:"id"`
	SellerID  string    `json:"sellerId"`
	DeletedAt time.Time `json:"deletedAt"`
}
