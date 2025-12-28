package consumers

import (
	"auction/app"
	"auction/pkg/events"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type BidEventHandler struct {
	repository app.Repository
	logger     *zap.Logger
}

func NewBidEventHandler(repository app.Repository, logger *zap.Logger) *BidEventHandler {
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
		zap.Any("payload", event.Payload),
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

	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("malformed payload - unmarshal failed: %w", err)
	}

	itemID, ok := payload["ItemID"].(string)
	if !ok || itemID == "" {
		return fmt.Errorf("malformed payload - itemId missing or invalid")
	}

	amountStr, ok := payload["Amount"].(string)
	if !ok || amountStr == "" {
		return fmt.Errorf("malformed payload - amount missing or invalid")
	}

	bidTime := event.Timestamp
	if bidTimeStr, ok := payload["Timestamp"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, bidTimeStr); err == nil {
			bidTime = parsedTime
		}
	}

	zap.L().Info("Processing bid.placed event",
		zap.String("itemId", itemID),
		zap.String("amount", amountStr),
		zap.Time("bidTime", bidTime),
		zap.String("traceId", event.TraceID),
	)

	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		item, err := h.repository.GetItem(ctx, itemID)
		if err != nil {
			return fmt.Errorf("failed to get item: %w", err)
		}

		item.CurrentPrice, err = decimal.NewFromString(amountStr)
		if err != nil {
			return fmt.Errorf("malformed payload - invalid amount format: %w", err)
		}

		originalEndDate := item.EndDate
		if item.ShouldExtendForBid(bidTime) {
			item.EndDate = item.CalculateNewEndDate()

			zap.L().Info("Extending auction end time",
				zap.String("itemId", itemID),
				zap.Time("originalEndDate", originalEndDate),
				zap.Time("newEndDate", item.EndDate),
				zap.Duration("extensionDuration", item.GetExtensionDuration()),
				zap.String("traceId", event.TraceID),
			)
		}

		item.UpdatedAt = time.Now()

		if err := h.repository.Update(ctx, item); err != nil {
			if strings.Contains(err.Error(), "optimistic lock failed") {
				if attempt < maxRetries {
					zap.L().Warn("Optimistic lock conflict, retrying",
						zap.String("itemId", itemID),
						zap.Int("attempt", attempt),
						zap.Int("maxRetries", maxRetries),
					)
					time.Sleep(time.Duration(10*attempt) * time.Millisecond)
					continue
				}
				return fmt.Errorf("failed to update item after %d retries due to concurrent modifications", maxRetries)
			}
			return fmt.Errorf("failed to update item: %w", err)
		}

		if !item.EndDate.Equal(originalEndDate) {
			zap.L().Info("Auction successfully extended",
				zap.String("itemId", itemID),
				zap.Time("newEndDate", item.EndDate),
				zap.Int("attempt", attempt),
			)
		}

		return nil
	}

	return fmt.Errorf("unexpected error: max retries reached")
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
	itemID, ok := payload["ItemID"].(string)
	if !ok || itemID == "" {
		return fmt.Errorf("malformed payload - itemId missing or invalid")
	}

	buyerID, ok := payload["BuyerID"].(string)
	if !ok || buyerID == "" {
		return fmt.Errorf("malformed payload - buyerId missing or invalid")
	}

	finalAmount, ok := payload["FinalAmount"].(string)
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
	item.BuyerID = &buyerID

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
