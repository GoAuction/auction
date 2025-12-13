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

type DeleteItemAttributeHandler struct {
	repository Repository
	publisher  events.Publisher
}

type DeleteItemAttributeRequest struct {
	ItemID      string `params:"itemId"`
	AttributeID string `params:"attributeId"`
}

type DeleteItemAttributeResponse struct {
}

func NewDeleteItemAttributeHandler(repository Repository, publisher events.Publisher) *DeleteItemAttributeHandler {
	return &DeleteItemAttributeHandler{
		repository: repository,
		publisher:  publisher,
	}
}

func (r *DeleteItemAttributeHandler) Handle(ctx context.Context, req *DeleteItemAttributeRequest) (*DeleteItemAttributeResponse, error) {
	userId := ctx.Value("UserID").(string)

	item, err := r.repository.GetItem(ctx, req.ItemID)
	if err == sql.ErrNoRows {
		return nil, httperror.NotFound("delete_item.destroy.not_found", "Item not found", nil)
	}

	if item.SellerID != userId {
		return nil, httperror.Forbidden("delete_item.destroy.forbidden", "You are not authorized to delete this item", nil)
	}

	err = r.repository.DeleteItemAttribute(ctx, req.ItemID, req.AttributeID)
	if err != nil {
		return nil, httperror.InternalServerError("delete_item.destroy.server_errror", "Internal server error", nil)
	}

	return &DeleteItemAttributeResponse{}, httperror.NoContent("delete_item.destroy.no_content", "Item deleted", nil)
}

func (r *DeleteItemAttributeHandler) publishEvent(ctx context.Context, item domain.Item, attributeID string) error {
	eventPayload := events.ItemAttributeDeletedPayload{
		ID:        attributeID,
		ItemID:    item.ID,
		DeletedAt: time.Now(),
	}

	headers := events.Headers{
		TraceID:       events.GenerateTraceID(),
		CorrelationID: events.GenerateCorrelationID(),
		Service:       "auction",
	}

	event := events.NewEvent(
		events.ItemAttributeDeletedEvent,
		events.EventVersionV1,
		eventPayload,
		headers,
	)

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := r.publisher.Publish(publishCtx, events.ItemExchange, event, headers); err != nil {
		zap.L().Error(
			"Failed to publish item.attribute.deleted event",
			zap.String("attributeID", attributeID),
			zap.Error(err),
		)
	}

	return nil
}
