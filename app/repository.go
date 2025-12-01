package app

import (
	"auction/domain"
	"context"
)

type Repository interface {
	Close() error
	GetItems(ctx context.Context, limit, offset int) ([]domain.Item, error)
	GetCategories(ctx context.Context, limit, offset int) ([]domain.Category, error)
	GetItem(ctx context.Context, id string) (domain.Item, error)
	GetUserItem(ctx context.Context, id string, userID string) (domain.Item, error)
	DeleteItem(ctx context.Context, id string, userID string) error
	CountItems(ctx context.Context) (int, error)
	CountCategories(ctx context.Context) (int, error)
	Create(ctx context.Context, req *CreateItemRequest) (domain.Item, error)
	UpdateUserItem(ctx context.Context, item domain.Item, userID string) error
	Update(ctx context.Context, item domain.Item) error
	GetCategoryByID(ctx context.Context, id string) (domain.Category, error)
	GetCategoriesByItemID(ctx context.Context, itemID string) ([]domain.Category, error)
}
