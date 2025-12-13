package app

import (
	"auction/domain"
	"auction/pkg/events"
	"auction/pkg/httperror"
	"context"
	"time"

	"go.uber.org/zap"
)

type CreateItemAttributesHandler struct {
	repository Repository
	publisher  events.Publisher
}

type AttributeKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CreateItemAttributesRequest struct {
	ItemID     string              `params:"itemId"`
	Attributes []AttributeKeyValue `json:"attributes"`
}

type CreateItemAttributesResponse struct {
	Attributes []domain.ItemAttribute `json:"attributes"`
}

func NewCreateItemAttributesHandler(repository Repository, publisher events.Publisher) *CreateItemAttributesHandler {
	return &CreateItemAttributesHandler{
		repository: repository,
		publisher:  publisher,
	}
}

func (r *CreateItemAttributesHandler) Handle(ctx context.Context, req *CreateItemAttributesRequest) (*CreateItemAttributesResponse, error) {
	item, err := r.repository.GetItem(ctx, req.ItemID)
	if err != nil {
		return nil, httperror.NotFound("create_item.store.not_found", "Item not found", nil)
	}

	attributes := make([]domain.ItemAttribute, len(req.Attributes))
	for i, attr := range req.Attributes {
		attributes[i] = domain.ItemAttribute{
			ItemID: item.ID,
			Key:    attr.Key,
			Value:  attr.Value,
		}
	}

	createdAttributes, err := r.repository.CreateItemAttributes(ctx, attributes)
	if err != nil {
		return nil, httperror.InternalServerError("create_item.store.create_failed", "Failed to create item attributes", nil)
	}

	r.publishEvent(ctx, createdAttributes)

	return &CreateItemAttributesResponse{
		Attributes: createdAttributes,
	}, nil
}

func (r CreateItemAttributesHandler) publishEvent(ctx context.Context, attributes []domain.ItemAttribute) error {
	for _, attribute := range attributes {
		go func() {
			eventPayload := events.ItemAttributeCreatedPayload{
				ID:        attribute.ID,
				ItemID:    attribute.ItemID,
				Key:       attribute.Key,
				Value:     attribute.Value,
				CreatedAt: attribute.CreatedAt,
			}

			headers := events.Headers{
				TraceID:       events.GenerateTraceID(),
				CorrelationID: events.GenerateCorrelationID(),
				Service:       "auction",
			}

			event := events.NewEvent(
				events.ItemAttributeCreatedEvent,
				events.EventVersionV1,
				eventPayload,
				headers,
			)

			publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := r.publisher.Publish(publishCtx, events.ItemExchange, event, headers); err != nil {
				zap.L().Error(
					"Failed to publish item.attribute.created event",
					zap.String("attributeID", attribute.ID),
					zap.Error(err),
				)
			}
		}()
	}

	return nil
}
