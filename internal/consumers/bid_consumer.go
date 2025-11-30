package consumers

import (
	"auction/pkg/events"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"auction/app/item"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type BidEventHandler struct {
	repository item.Repository
	logger     *zap.Logger
}

func NewBidEventHandler(repository item.Repository, logger *zap.Logger) *BidEventHandler {
	return &BidEventHandler{
		repository: repository,
		logger:     logger,
	}
}

func (h *BidEventHandler) HandleEvent(ctx context.Context, event *events.Event) error {
	zap.L().Info("Bid event received",
		zap.String("event", event.Event),
		zap.String("version", event.Version),
		zap.String("traceId", event.TraceID),
	)

	switch event.Event {
	case "bid.placed":
		return h.handleBidPlaced(ctx, event)
	case "bid.won":
		return h.handleBidWon(ctx, event)
	default:
		zap.L().Warn("Unknown bid event type", zap.String("event", event.Event))
		return nil
	}
}

func (h *BidEventHandler) handleBidPlaced(ctx context.Context, event *events.Event) error {
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("malformed payload - marshal failed: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("malformed payload - unmarshal failed: %w", err)
	}

	itemID, ok := payload["itemId"].(string)
	if !ok || itemID == "" {
		return fmt.Errorf("malformed payload - itemId missing or invalid")
	}

	amountStr, ok := payload["amount"].(string)
	if !ok || amountStr == "" {
		return fmt.Errorf("malformed payload - amount missing or invalid")
	}

	zap.L().Info("Processing bid.placed event",
		zap.String("itemId", itemID),
		zap.String("amount", amountStr),
		zap.String("traceId", event.TraceID),
	)

	item, err := h.repository.GetItem(ctx, itemID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	item.CurrentPrice, err = decimal.NewFromString(amountStr)
	if err != nil {
		return fmt.Errorf("malformed payload - invalid amount format: %w", err)
	}
	item.UpdatedAt = time.Now()

	if err := h.repository.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	return nil
}

func (h *BidEventHandler) handleBidWon(ctx context.Context, event *events.Event) error {
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("malformed payload - marshal failed: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("malformed payload - unmarshal failed: %w", err)
	}

	// Validate required fields
	itemID, ok := payload["itemId"].(string)
	if !ok || itemID == "" {
		return fmt.Errorf("malformed payload - itemId missing or invalid")
	}

	buyerID, ok := payload["buyerId"].(string)
	if !ok || buyerID == "" {
		return fmt.Errorf("malformed payload - buyerId missing or invalid")
	}

	finalAmount, ok := payload["finalAmount"].(string)
	if !ok || finalAmount == "" {
		return fmt.Errorf("malformed payload - finalAmount missing or invalid")
	}

	zap.L().Info("Processing bid.won event",
		zap.String("itemId", itemID),
		zap.String("buyerId", buyerID),
		zap.String("finalAmount", finalAmount),
		zap.String("traceId", event.TraceID),
	)

	item, err := h.repository.GetItem(ctx, itemID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	item.Status = "sold"
	item.BuyerID = buyerID

	finalPrice, err := decimal.NewFromString(finalAmount)
	if err != nil {
		return fmt.Errorf("malformed payload - invalid finalAmount format: %w", err)
	}
	item.CurrentPrice = finalPrice
	item.EndPrice = &finalPrice
	item.UpdatedAt = time.Now()

	if err := h.repository.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	return nil
}
