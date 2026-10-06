package seeder

import (
	"context"

	db "github.com/MamangRust/microservice-ecommerce-grpc-user/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/hash"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"

	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type userSeeder struct {
	db     *db.Queries
	hash   hash.HashPassword
	ctx    context.Context
	logger logger.LoggerInterface
}

func NewUserSeeder(db *db.Queries, hash hash.HashPassword, ctx context.Context, logger logger.LoggerInterface) *userSeeder {
	return &userSeeder{
		db:     db,
		hash:   hash,
		ctx:    ctx,
		logger: logger,
	}
}

func (r *userSeeder) Seed() error {
	// Idempotency: skip when users already exist.
	existing, err := r.db.GetUsers(r.ctx, db.GetUsersParams{
		Column1: "",
		Limit:   1,
		Offset:  0,
	})
	if err == nil && len(existing) > 0 {
		r.logger.Debug("users already seeded, skipping")
		return nil
	}

	for i := 1; i <= 10; i++ {
		email := fmt.Sprintf("user_%s@example.com", uuid.NewString())
		rawPassword := fmt.Sprintf("password%d", i)

		hashedPassword, err := r.hash.HashPassword(rawPassword)
		if err != nil {
			r.logger.Error("failed to hash password", zap.Int("user", i), zap.Error(err))
			return fmt.Errorf("failed to hash password for user %d: %w", i, err)
		}

		user := db.CreateUserParams{
			Firstname: fmt.Sprintf("User%d", i),
			Lastname:  fmt.Sprintf("Last%d", i),
			Email:     email,
			Password:  hashedPassword,
		}

		createdUser, err := r.db.CreateUser(r.ctx, user)
		if err != nil {
			r.logger.Error("failed to seed user", zap.Int("user", i), zap.Error(err))
			return fmt.Errorf("failed to seed user %d: %w", i, err)
		}

		if i > 5 {
			_, err = r.db.TrashUser(r.ctx, createdUser.UserID)
			if err != nil {
				r.logger.Error("failed to trash user", zap.Int("user", i), zap.Error(err))
				return fmt.Errorf("failed to trash user %d: %w", i, err)
			}
		}
	}

	r.logger.Info("User seeding completed successfully")
	return nil
}
