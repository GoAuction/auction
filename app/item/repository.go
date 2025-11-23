package item

import (
	"auction/domain"
	"context"
)

type Repository interface {
	FindByID(id string) (*domain.User, error)
	FindByEmail(email string) (*domain.User, error)
	Create(ctx context.Context, email string, password string, name string) (string, error)
	Update(ctx context.Context, id string, email string, name string) error
	EnableTwoFactor(ctx context.Context, id string, twoFactorSecret string) error
	DisableTwoFactor(ctx context.Context, id string) error
	MarkTwoFactorVerified(ctx context.Context, id string) error
	SetRecoveryCodes(ctx context.Context, id string, recoveryCodes string) error
}
