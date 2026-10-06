package main

import (
	"context"
	"fmt"

	db "github.com/MamangRust/microservice-ecommerce-grpc-banner/database/schema"
	"github.com/MamangRust/microservice-ecommerce-grpc-banner/seeder"
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

func main() {
	logger, err := logger.NewLogger("seeder", nil)
	if err != nil {
		logger.Fatal("Failed to initialize logger", zap.Error(err))
	}

	if err := dotenv.Viper(); err != nil {
		logger.Fatal("Failed to load .env file", zap.Error(err))
	}

	ctx := context.Background()

	q, closeFn, err := open(logger, database.ExperienceCluster)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer closeFn()

	s := seeder.NewBannerSeeder(q, ctx, logger)
	if err := s.Seed(); err != nil {
		logger.Fatal("Failed to seed banners", zap.Error(err))
	}

	logger.Info("banners seeded successfully")
}
