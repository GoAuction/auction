package app

import (
	"auction/domain"
	"auction/pkg/aws"
	"auction/pkg/config"
	"auction/pkg/events"
	"auction/pkg/httperror"
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

var appConfig = config.Read()

type DeleteItemImageHandler struct {
	repository     Repository
	eventPublisher events.Publisher
}

func NewDeleteItemImageHandler(repository Repository, eventPublisher events.Publisher) *DeleteItemImageHandler {
	return &DeleteItemImageHandler{
		repository:     repository,
		eventPublisher: eventPublisher,
	}
}

type DeleteItemImageRequest struct {
	ItemID  string `params:"itemId"`
	ImageID string `params:"imageId"`
}

type DeleteItemImageResponse struct {
}

func (h *DeleteItemImageHandler) Handle(ctx context.Context, req *DeleteItemImageRequest) (*DeleteItemImageResponse, error) {
	userId := ctx.Value("UserID").(string)

	item, err := h.repository.GetItem(ctx, req.ItemID)
	if err != nil {
		return nil, httperror.NotFound("delete_item_image.destroy.not_found", "Item not found.", nil)
	}
	if item.SellerID != userId {
		return nil, httperror.Forbidden("delete_item_image.destroy.forbidden", "You are not authorized to delete this item.", nil)
	}

	image, err := h.repository.GetItemImage(ctx, req.ItemID, req.ImageID)
	if err != nil {
		return nil, httperror.NotFound("delete_item_image.destroy.not_found", "Image not found.", nil)
	}

	bucket := aws.NewS3Bucket()
	key := extractImageKey(image.ImageURL)
	err = bucket.Delete(key)
	if err != nil {
		return nil, httperror.InternalServerError("delete_item_image.destroy.failed", "Failed to delete image.", err)
	}

	err = h.repository.DeleteItemImage(ctx, req.ItemID, req.ImageID)
	if err != nil {
		return nil, httperror.InternalServerError("delete_item_image.destroy.failed", "Failed to delete image.", err)
	}

	h.publishEvent(ctx, image)

	return &DeleteItemImageResponse{}, httperror.NoContent("delete_item_image.destroy.success", "Image deleted successfully.", nil)
}

func extractImageKey(imageUrl string) string {
	url := fmt.Sprintf("%s/%s/", appConfig.AWSEndpoint, appConfig.AWSBucket)
	return strings.Replace(imageUrl, url, "", 1)
}

func (e DeleteItemImageHandler) publishEvent(ctx context.Context, image domain.ItemImage) {
	eventPayload := events.ItemImageDeletedPayload{
		ID:        image.ID,
		ItemID:    image.ItemID,
		ImageURL:  image.ImageURL,
		DeletedAt: time.Now(),
	}

	headers := events.Headers{
		TraceID:       events.GenerateTraceID(),
		CorrelationID: events.GenerateCorrelationID(),
		Service:       "auction",
	}

	event := events.NewEvent(
		events.ItemImageDeletedEvent,
		events.EventVersionV1,
		eventPayload,
		headers,
	)

	if err := e.eventPublisher.Publish(ctx, events.ItemExchange, event, headers); err != nil {
		zap.L().Error("Failed to publish item.image.deleted event",
			zap.String("imageID", image.ID),
			zap.Error(err),
		)
	}
}
