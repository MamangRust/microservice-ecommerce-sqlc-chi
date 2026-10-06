package main

import (
	"context"
	"fmt"

	reviewdetaildb "github.com/MamangRust/microservice-ecommerce-grpc-review-detail/database/schema"
	db "github.com/MamangRust/microservice-ecommerce-grpc-review/database/schema"
	"github.com/MamangRust/microservice-ecommerce-grpc-review/seeder"
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

// openDetail connects to the review_detail tables, which live in the same
// experience context as reviews (one PostgreSQL instance per bounded context).
func openDetail(logger logger.LoggerInterface, prefix string) (*reviewdetaildb.Queries, func(), error) {
	conn, err := database.NewClientWithPrefix(logger, prefix)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to %s: %w", prefix, err)
	}
	closeFn := func() { conn.Close() }
	return reviewdetaildb.New(conn), closeFn, nil
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

	reviewDB, closeReview, err := open(logger, database.ExperienceCluster)
	if err != nil {
		logger.Fatal("Failed to connect to review database", zap.Error(err))
	}
	defer closeReview()

	reviewDetailDB, closeDetail, err := openDetail(logger, database.ExperienceCluster)
	if err != nil {
		logger.Fatal("Failed to connect to review_detail database", zap.Error(err))
	}
	defer closeDetail()

	s := seeder.NewReviewSeeder(reviewDB, reviewDetailDB, ctx, logger)
	if err := s.Seed(); err != nil {
		logger.Fatal("Failed to seed reviews", zap.Error(err))
	}

	logger.Info("reviews and review_details seeded successfully")
}
