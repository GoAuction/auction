package app

import (
	"auction/domain"
	"auction/pkg/events"
	"auction/pkg/httperror"
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"
)

type DeleteItemHandler struct {
	repository     Repository
	eventPublisher events.Publisher
}

func NewDeleteItemHandler(repository Repository, eventPublisher events.Publisher) *DeleteItemHandler {
	return &DeleteItemHandler{
		repository:     repository,
		eventPublisher: eventPublisher,
	}
}

type DeleteItemRequest struct {
	ItemID string `params:"id" validate:"required,uuid"`
}

type DeleteItemResponse struct {
}

func (h DeleteItemHandler) Handle(ctx context.Context, req *DeleteItemRequest) (*DeleteItemResponse, error) {
	userID := ctx.Value("UserID").(string)

	item, err := h.repository.GetUserItem(ctx, req.ItemID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperror.NotFound(
				"item.destroy.notfound",
				"Item not found",
				nil,
			)
		}
		return nil, httperror.InternalServerError(
			"item.destroy.failed",
			"Failed to retrieve items",
			nil,
		)
	}

	err = h.repository.DeleteItem(ctx, req.ItemID, userID)
	if err != nil {
		return nil, httperror.InternalServerError(
			"item.destroy.failed",
			"Failed to retrieve items",
			nil,
		)
	}

	h.publishEvent(ctx, item)

	return nil, httperror.NoContent(
		"item.destroy.success",
		"Item deleted successfully",
		nil,
	)
}

func (h DeleteItemHandler) publishEvent(ctx context.Context, item domain.Item) {
	if h.eventPublisher != nil {
		eventPayload := events.ItemDeletedPayload{
			ID:        item.ID,
			SellerID:  item.SellerID,
			DeletedAt: time.Now().UTC(),
		}

		headers := events.Headers{
			TraceID:       events.GenerateTraceID(),
			CorrelationID: events.GenerateCorrelationID(),
			Service:       "auction",
		}

		event := events.NewEvent(
			events.ItemDeletedEvent,
			events.EventVersionV1,
			eventPayload,
			headers,
		)

		if err := h.eventPublisher.Publish(ctx, events.ItemExchange, event, headers); err != nil {
			zap.L().Error("Failed to publish item.deleted event",
				zap.String("itemId", item.ID),
				zap.Error(err),
			)
		}
	}
}
