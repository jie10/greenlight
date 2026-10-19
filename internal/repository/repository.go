package repository

import (
	"context"

	"github.com/jie10/greenlight/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user string) error
	Delete(ctx context.Context, id string) error
}
