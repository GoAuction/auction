package app

import (
	"auction/pkg/aws"
	"auction/pkg/config"
	"auction/pkg/events"
	"auction/pkg/httperror"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UploadItemImageHandler struct {
	repository     Repository
	eventPublisher events.Publisher
}

func NewUploadItemImageHandler(repository Repository, eventPublisher events.Publisher) *UploadItemImageHandler {
	return &UploadItemImageHandler{
		repository:     repository,
		eventPublisher: eventPublisher,
	}
}

type UploadItemImageRequest struct {
	ItemID string `params:"itemId"`
}

type UploadItemImageResponse struct {
	ItemID   string `json:"item_id"`
	ImageID  string `json:"image_id"`
	ImageUrl string `json:"image_url"`
}

func (h *UploadItemImageHandler) Handle(ctx context.Context, req *UploadItemImageRequest) (*UploadItemImageResponse, error) {
	fiberCtx := ctx.Value("fiber")
	if fiberCtx == nil {
		return nil, httperror.InternalServerError("upload.no_context", "Fiber context not found", nil)
	}

	c, ok := fiberCtx.(*fiber.Ctx)
	if !ok {
		return nil, httperror.InternalServerError("upload.invalid_context", "Invalid Fiber context", nil)
	}

	userId := ctx.Value("UserID").(string)

	item, err := h.repository.GetItem(ctx, req.ItemID)
	if err != nil {
		return nil, httperror.NotFound("upload_item_image.not_found", "Item not found.", nil)
	}
	if item.SellerID != userId {
		return nil, httperror.Forbidden("upload_item_image.forbidden", "You are not authorized to upload images for this item.", nil)
	}

	file, err := c.FormFile("image")
	if err != nil {
		return nil, httperror.BadRequest("upload.missing_file", "Image file is required (use 'image' field)", fiber.Map{"error": err.Error()})
	}

	// Validate file size (max 5MB)
	const maxFileSize = 5 * 1024 * 1024
	if file.Size > maxFileSize {
		return nil, httperror.BadRequest("upload.file_too_large", "File size must not exceed 5MB",
			fiber.Map{
				"size_mb": float64(file.Size) / 1024 / 1024,
				"max_mb":  5,
			})
	}

	// Get content type
	contentType := file.Header.Get("Content-Type")

	// Validate content type
	allowedTypes := map[string]bool{
		"image/png":  true,
		"image/jpeg": true,
		"image/jpg":  true,
	}
	if !allowedTypes[contentType] {
		return nil, httperror.BadRequest("upload.invalid_content_type", "Only PNG, JPEG/JPG images are allowed",
			fiber.Map{
				"received": contentType,
				"allowed":  []string{"image/png", "image/jpeg", "image/jpg"},
			})
	}

	fileReader, err := file.Open()
	if err != nil {
		return nil, httperror.InternalServerError("upload.file_open_error", "Failed to open uploaded file", err.Error())
	}
	defer fileReader.Close()

	fileBytes, err := io.ReadAll(fileReader)
	if err != nil {
		return nil, httperror.InternalServerError("upload.file_read_error", "Failed to read file content", err.Error())
	}

	return h.processUpload(ctx, req.ItemID, fileBytes, contentType, file.Filename)
}

func (h *UploadItemImageHandler) processUpload(ctx context.Context, itemID string, imageData []byte, contentType, fileName string) (*UploadItemImageResponse, error) {
	extension := getExtensionFromContentType(contentType)

	key := fmt.Sprintf("items/%s/%s%s", itemID, uuid.New().String(), extension)

	bucket := aws.NewS3Bucket()

	err := bucket.Upload(key, imageData)
	if err != nil {
		return nil, httperror.InternalServerError("upload_item.upload.failed", "Failed to upload image to storage", err.Error())
	}

	imageURL := constructImageURL(key)

	savedImage, err := h.repository.SaveImage(ctx, itemID, imageURL)
	if err != nil {
		_ = bucket.Delete(key)
		return nil, httperror.InternalServerError("upload_item.store.failed", "Failed to save image metadata", err.Error())
	}

	if h.eventPublisher != nil {
		eventPayload := events.ItemImageUploadedPayload{
			ID:        savedImage.ID,
			ItemID:    itemID,
			ImageURL:  imageURL,
			CreatedAt: time.Now(),
		}

		headers := events.Headers{
			TraceID:       events.GenerateTraceID(),
			CorrelationID: events.GenerateCorrelationID(),
			Service:       "auction",
		}

		event := events.NewEvent(
			events.ItemImageUploadedEvent,
			events.EventVersionV1,
			eventPayload,
			headers,
		)

		if err := h.eventPublisher.Publish(ctx, events.ItemExchange, event, headers); err != nil {
			zap.L().Error("Failed to publish item.image.uploaded event",
				zap.String("imageID", savedImage.ID),
				zap.Error(err),
			)
		}
	}

	return &UploadItemImageResponse{
		ItemID:   itemID,
		ImageID:  savedImage.ID,
		ImageUrl: savedImage.ImageURL,
	}, nil
}

func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/svg+xml":
		return ".svg"
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func constructImageURL(key string) string {
	// Get AWS config
	cfg := config.Read()

	// For MinIO/S3, construct the public URL
	// Format: http(s)://endpoint/bucket/key
	if cfg.AWSEndpoint != "" {
		return fmt.Sprintf("%s/%s/%s", cfg.AWSEndpoint, cfg.AWSBucket, key)
	}

	// For AWS S3, use standard URL format
	if cfg.AWSDefaultRegion != "" {
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.AWSBucket, cfg.AWSDefaultRegion, key)
	}

	// Fallback
	return key
}
