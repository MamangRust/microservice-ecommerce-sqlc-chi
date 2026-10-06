package seeder

import (
	"context"

	orderitemdb "github.com/MamangRust/microservice-ecommerce-grpc-order-item/database/schema"
	db "github.com/MamangRust/microservice-ecommerce-grpc-order/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"

	"go.uber.org/zap"
)

// orderSeeder seeds orders and their order_items, which share the sales
// context database, so it holds a connection per table owner.
type orderSeeder struct {
	orderDB     *db.Queries
	orderItemDB *orderitemdb.Queries
	ctx         context.Context
	logger      logger.LoggerInterface
}

func NewOrderSeeder(orderDB *db.Queries, orderItemDB *orderitemdb.Queries, ctx context.Context, logger logger.LoggerInterface) *orderSeeder {
	return &orderSeeder{
		orderDB:     orderDB,
		orderItemDB: orderItemDB,
		ctx:         ctx,
		logger:      logger,
	}
}

func (r *orderSeeder) Seed() error {
	// Idempotency: skip when orders already exist.
	existing, err := r.orderDB.GetOrders(r.ctx, db.GetOrdersParams{
		Column1: "",
		Limit:   1,
		Offset:  0,
	})
	if err == nil && len(existing) > 0 {
		r.logger.Debug("orders already seeded, skipping")
		return nil
	}

	for i := 1; i <= 8; i++ {
		order, err := r.orderDB.CreateOrder(r.ctx, db.CreateOrderParams{
			MerchantID: int32(i),
			UserID:     int32(i),
			TotalPrice: int32(10000 * i),
		})
		if err != nil {
			r.logger.Error("failed to create order", zap.Error(err))
			return err
		}

		_, err = r.orderItemDB.CreateOrderItem(r.ctx, orderitemdb.CreateOrderItemParams{
			OrderID:   order.OrderID,
			ProductID: int32(i),
			Quantity:  int32(i),
			Price:     int32(10000),
		})
		if err != nil {
			r.logger.Error("failed to create order item", zap.Error(err))
			return err
		}
	}

	r.logger.Info("order & order-item successfully seeded")

	return nil
}
