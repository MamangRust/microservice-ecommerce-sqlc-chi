package repository

import (
	"context"

	db "github.com/MamangRust/microservice-ecommerce-auth/database/schema"
	"github.com/MamangRust/microservice-ecommerce-shared/domain/requests"
	"github.com/jackc/pgx/v5"
)

// RefreshTokenRepository is auth's own refresh-token store.
//
//go:generate mockgen -source=interfaces.go -destination=mocks/mock.go
type RefreshTokenRepository interface {
	FindByToken(ctx context.Context, token string) (*db.RefreshToken, error)

	FindByUserId(ctx context.Context, user_id int) (*db.RefreshToken, error)

	CreateRefreshToken(ctx context.Context, req *requests.CreateRefreshToken) (*db.RefreshToken, error)

	UpdateRefreshToken(ctx context.Context, req *requests.UpdateRefreshToken) (*db.RefreshToken, error)

	DeleteRefreshToken(ctx context.Context, token string) error

	DeleteRefreshTokenByUserId(ctx context.Context, user_id int) error
}

// ResetTokenRepository is auth's own password-reset-token store.
type ResetTokenRepository interface {
	FindByToken(ctx context.Context, code string) (*db.ResetToken, error)

	CreateResetToken(ctx context.Context, req *requests.CreateResetTokenRequest) (*db.ResetToken, error)

	CreateResetTokenInTx(ctx context.Context, tx pgx.Tx, req *requests.CreateResetTokenRequest) (*db.ResetToken, error)

	DeleteResetToken(ctx context.Context, user_id int) error
}
