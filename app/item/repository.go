package item

import (
	"auction/domain"
	"context"
)

type Repository interface {
	Close() error
	GetItems(ctx context.Context, limit, offset int) ([]domain.Item, error)
	GetItem(ctx context.Context, id string) (domain.Item, error)
	GetUserItem(ctx context.Context, id string, userID string) (domain.Item, error)
	DeleteItem(ctx context.Context, id string, userID string) error
	CountItems(ctx context.Context) (int, error)
	Create(ctx context.Context, req *CreateItemRequest) (domain.Item, error)
	Update(ctx context.Context, item domain.Item, userID string) error
}
