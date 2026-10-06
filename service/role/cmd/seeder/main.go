package main

import (
	"context"
	"fmt"

	db "github.com/MamangRust/microservice-ecommerce-grpc-role/database/schema"
	"github.com/MamangRust/microservice-ecommerce-grpc-role/seeder"
	userdb "github.com/MamangRust/microservice-ecommerce-grpc-user/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	"github.com/MamangRust/microservice-ecommerce-pkg/dotenv"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"
	"go.uber.org/zap"
)

// open connects to the database configured via the given DBCluster prefix.
func open(logger logger.LoggerInterface, prefix string) (*db.Queries, func(), error) {
	conn, err := database.NewClientWithPrefix(logger, prefix)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to %s: %w", prefix, err)
	}
	closeFn := func() { conn.Close() }
	return db.New(conn), closeFn, nil
}

// openUser connects to the user tables, which live in the same identity
// context as roles (one PostgreSQL instance per bounded context).
func openUser(logger logger.LoggerInterface, prefix string) (*userdb.Queries, func(), error) {
	conn, err := database.NewClientWithPrefix(logger, prefix)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to %s: %w", prefix, err)
	}
	closeFn := func() { conn.Close() }
	return userdb.New(conn), closeFn, nil
}

func main() {
	logger, err := logger.NewLogger("seeder", nil)
	if err != nil {
		logger.Fatal("Failed to initialize logger", zap.Error(err))
	}

	if err := dotenv.Viper(); err != nil {
		logger.Fatal("Failed to load .env file", zap.Error(err))
	}

	ctx := context.Background()

	roleDB, closeRole, err := open(logger, database.IdentityCluster)
	if err != nil {
		logger.Fatal("Failed to connect to role database", zap.Error(err))
	}
	defer closeRole()

	userDB, closeUser, err := openUser(logger, database.IdentityCluster)
	if err != nil {
		logger.Fatal("Failed to connect to user database", zap.Error(err))
	}
	defer closeUser()

	role := seeder.NewRoleSeeder(roleDB, ctx, logger)
	if err := role.Seed(); err != nil {
		logger.Fatal("Failed to seed roles", zap.Error(err))
	}

	userRole := seeder.NewUserRoleSeeder(userDB, roleDB, ctx, logger)
	if err := userRole.Seed(); err != nil {
		logger.Fatal("Failed to seed user_roles", zap.Error(err))
	}

	logger.Info("roles and user_roles seeded successfully")
}
