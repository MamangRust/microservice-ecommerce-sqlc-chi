package main

import (
	"context"
	"fmt"

	orderitemdb "github.com/MamangRust/microservice-ecommerce-grpc-order-item/database/schema"
	db "github.com/MamangRust/microservice-ecommerce-grpc-order/database/schema"
	"github.com/MamangRust/microservice-ecommerce-grpc-order/seeder"
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

// openItem connects to the order_item tables, which live in the same sales
// context as orders (one PostgreSQL instance per bounded context).
func openItem(logger logger.LoggerInterface, prefix string) (*orderitemdb.Queries, func(), error) {
	conn, err := database.NewClientWithPrefix(logger, prefix)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to %s: %w", prefix, err)
	}
	closeFn := func() { conn.Close() }
	return orderitemdb.New(conn), closeFn, nil
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

	orderDB, closeOrder, err := open(logger, database.SalesCluster)
	if err != nil {
		logger.Fatal("Failed to connect to order database", zap.Error(err))
	}
	defer closeOrder()

	orderItemDB, closeItem, err := openItem(logger, database.SalesCluster)
	if err != nil {
		logger.Fatal("Failed to connect to order_item database", zap.Error(err))
	}
	defer closeItem()

	s := seeder.NewOrderSeeder(orderDB, orderItemDB, ctx, logger)
	if err := s.Seed(); err != nil {
		logger.Fatal("Failed to seed orders", zap.Error(err))
	}

	logger.Info("orders and order_items seeded successfully")
}
