package seeder

import (
	"context"
	"fmt"

	db "github.com/MamangRust/microservice-ecommerce-grpc-role/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"

	"go.uber.org/zap"
)

type roleSeeder struct {
	db     *db.Queries
	ctx    context.Context
	logger logger.LoggerInterface
}

func NewRoleSeeder(db *db.Queries, ctx context.Context, logger logger.LoggerInterface) *roleSeeder {
	return &roleSeeder{
		db:     db,
		ctx:    ctx,
		logger: logger,
	}
}

func (r *roleSeeder) Seed() error {
	// Idempotency: skip only when one of our seeded roles already exists.
	// (The e2e reset pre-inserts ROLE_ADMIN/ROLE_USER, so "any role exists"
	// must not suppress the Cashier/Manager/Admin/Supplier seeding.)
	existing, err := r.db.GetRoles(r.ctx, db.GetRolesParams{
		Column1: "Cashier",
		Limit:   1,
		Offset:  0,
	})
	if err == nil && len(existing) > 0 {
		r.logger.Debug("roles already seeded, skipping")
		return nil
	}

	randomRoles := []string{"Cashier", "Manager", "Admin", "Supplier"}

	totalRoles := len(randomRoles)

	for i, roleName := range randomRoles {
		_, err := r.db.CreateRole(r.ctx, roleName)
		if err != nil {
			r.logger.Error("failed to seed role", zap.Int("role", i+1), zap.String("roleName", roleName), zap.Error(err))
			return fmt.Errorf("failed to seed role %d (%s): %w", i+1, roleName, err)
		}
	}

	r.logger.Debug("role seeded successfully", zap.Int("totalRoles", totalRoles))
	return nil
}
