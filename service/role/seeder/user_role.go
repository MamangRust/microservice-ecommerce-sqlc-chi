package seeder

import (
	"context"
	"fmt"
	"time"

	db "github.com/MamangRust/microservice-ecommerce-grpc-role/database/schema"
	userdb "github.com/MamangRust/microservice-ecommerce-grpc-user/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/exp/rand"
)

// userRoleSeeder assigns seeded users to seeded roles. users, roles and
// user_roles all live in the identity context database, so it holds a
// connection per table owner.
type userRoleSeeder struct {
	userDB *userdb.Queries
	roleDB *db.Queries
	ctx    context.Context
	logger logger.LoggerInterface
}

func NewUserRoleSeeder(userDB *userdb.Queries, roleDB *db.Queries, ctx context.Context, logger logger.LoggerInterface) *userRoleSeeder {
	return &userRoleSeeder{
		userDB: userDB,
		roleDB: roleDB,
		ctx:    ctx,
		logger: logger,
	}
}

func (r *userRoleSeeder) Seed() error {
	users, err := r.userDB.GetUsers(r.ctx, userdb.GetUsersParams{
		Column1: "",
		Limit:   int32(20),
		Offset:  0,
	})
	if err != nil {
		r.logger.Error("failed to fetch users", zap.Error(err))
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	roles, err := r.roleDB.GetRoles(r.ctx, db.GetRolesParams{
		Column1: "",
		Limit:   4,
		Offset:  0,
	})
	if err != nil {
		r.logger.Error("failed to fetch roles", zap.Error(err))
		return fmt.Errorf("failed to fetch roles: %w", err)
	}

	if len(users) == 0 || len(roles) == 0 {
		r.logger.Debug("no users or roles available for seeding")
		return nil
	}

	// Idempotency: skip when every seeded user already has a role assigned.
	assigned, err := r.roleDB.GetUserRoles(r.ctx, users[0].UserID)
	if err == nil && len(assigned) > 0 {
		r.logger.Debug("user roles already seeded, skipping")
		return nil
	}

	rand.Seed(uint64(time.Now().UnixNano()))

	for _, user := range users {
		role := roles[rand.Intn(len(roles))]

		_, err := r.roleDB.AssignRoleToUser(r.ctx, db.AssignRoleToUserParams{
			UserID: user.UserID,
			RoleID: role.RoleID,
		})
		if err != nil {
			r.logger.Error("failed to assign role to user", zap.String("user", user.Email), zap.String("role", role.RoleName), zap.Error(err))
			return fmt.Errorf("failed to assign role %s to user %s: %w", role.RoleName, user.Email, err)
		}
	}

	r.logger.Info("user roles assigned successfully")
	return nil
}
