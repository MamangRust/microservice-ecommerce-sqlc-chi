package apps

import (
	"fmt"
	"time"

	pb_merchant "github.com/MamangRust/microservice-ecommerce-grpc-pb/merchant"
	pb_order "github.com/MamangRust/microservice-ecommerce-grpc-pb/order"
	pb_order_item "github.com/MamangRust/microservice-ecommerce-grpc-pb/order_item"
	pb_shipping_address "github.com/MamangRust/microservice-ecommerce-grpc-pb/shipping_address"
	pb_transaction "github.com/MamangRust/microservice-ecommerce-grpc-pb/transaction"
	pb_user "github.com/MamangRust/microservice-ecommerce-grpc-pb/user"
	"github.com/MamangRust/microservice-ecommerce-grpc-transaction/cache"
	db "github.com/MamangRust/microservice-ecommerce-grpc-transaction/database/schema"
	"github.com/MamangRust/microservice-ecommerce-grpc-transaction/handler"
	transactionKafka "github.com/MamangRust/microservice-ecommerce-grpc-transaction/kafka"
	"github.com/MamangRust/microservice-ecommerce-grpc-transaction/repository"
	"github.com/MamangRust/microservice-ecommerce-grpc-transaction/service"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	"github.com/MamangRust/microservice-ecommerce-pkg/kafka"
	"github.com/MamangRust/microservice-ecommerce-pkg/resilience"
	"github.com/MamangRust/microservice-ecommerce-pkg/server"
	"github.com/MamangRust/microservice-ecommerce-shared/observability"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewServer(cfg *server.Config) (*server.GRPCServer, error) {
	srv, err := server.New(cfg)
	if err != nil {
		return nil, err
	}

	queries := db.New(srv.Pool)

	// gRPC Client Connections
	userAddr := viper.GetString("GRPC_USER_ADDR")

	userConn, err := grpc.NewClient(userAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	merchantAddr := viper.GetString("GRPC_MERCHANT_ADDR")

	merchantConn, err := grpc.NewClient(merchantAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to merchant service: %w", err)
	}

	orderAddr := viper.GetString("GRPC_ORDER_ADDR")

	orderConn, err := grpc.NewClient(orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to order service: %w", err)
	}

	orderItemAddr := viper.GetString("GRPC_ORDER_ITEM_ADDR")
	if orderItemAddr == "" {
		orderItemAddr = "order-item:50056"
	}
	orderItemConn, err := grpc.NewClient(orderItemAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to order_item service: %w", err)
	}

	shippingAddr := viper.GetString("GRPC_SHIPPING_ADDRESS_ADDR")
	if shippingAddr == "" {
		shippingAddr = "shipping_address:50063"
	}
	shippingConn, err := grpc.NewClient(shippingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to shipping_address service: %w", err)
	}

	repos := repository.NewRepositories(&repository.Deps{
		DB:                   queries,
		UserQueryClient:      pb_user.NewUserQueryServiceClient(userConn),
		MerchantQueryClient:  pb_merchant.NewMerchantQueryServiceClient(merchantConn),
		OrderQueryClient:     pb_order.NewOrderQueryServiceClient(orderConn),
		OrderItemQueryClient: pb_order_item.NewOrderItemQueryServiceClient(orderItemConn),
		ShippingQueryClient:  pb_shipping_address.NewShippingQueryServiceClient(shippingConn),
		Guards: repository.GuardOptions{
			User: []adapter.GuardOption{
				adapter.WithDependencyGuard(resilience.NewDependencyGuard("user", 5, 30, 100, 3*time.Second, srv.Logger)),
			},
			Merchant: []adapter.GuardOption{
				adapter.WithDependencyGuard(resilience.NewDependencyGuard("merchant", 5, 30, 100, 3*time.Second, srv.Logger)),
			},
			Order: []adapter.GuardOption{
				adapter.WithDependencyGuard(resilience.NewDependencyGuard("order", 5, 30, 100, 3*time.Second, srv.Logger)),
			},
			OrderItem: []adapter.GuardOption{
				adapter.WithDependencyGuard(resilience.NewDependencyGuard("order-item", 5, 30, 100, 3*time.Second, srv.Logger)),
			},
			Shipping: []adapter.GuardOption{
				adapter.WithDependencyGuard(resilience.NewDependencyGuard("shipping-address", 5, 30, 100, 3*time.Second, srv.Logger)),
			},
		},
	})
	myKafka := kafka.NewKafka(srv.Logger, []string{viper.GetString("KAFKA_BROKERS")})
	obs, _ := observability.NewObservability("transaction-server", srv.Logger)
	cache := cache.NewMencache(srv.CacheStore)

	svc := service.NewService(&service.Deps{
		Kafka:         myKafka,
		Pool:          srv.Pool,
		Cache:         cache,
		Logger:        srv.Logger,
		Repositories:  repos,
		Observability: obs,
	})

	h := handler.NewHandler(&handler.Deps{Service: svc, Logger: srv.Logger})

	// Start the outbox relay so events committed after the transaction insert are
	// published to Kafka with durable retry and dead-letter semantics.
	go svc.Outbox.Start(srv.Ctx, service.OutboxRelayInterval, service.OutboxRelayBatchSize)

	if err := myKafka.StartConsumersWithContext(srv.Ctx, []string{"transaction-service-topic-merchant-status-event"}, "transaction-service-group", transactionKafka.NewMerchantStatusConsumer(srv.Ctx, cache.TransactionCommandCache, srv.Logger)); err != nil {
		return nil, fmt.Errorf("failed to start merchant status consumer: %w", err)
	}

	srv.RegisterServices = func(gs *grpc.Server) {
		pb_transaction.RegisterTransactionQueryServiceServer(gs, h.TransactionQuery)
		pb_transaction.RegisterTransactionCommandServiceServer(gs, h.TransactionCommand)
	}

	return srv, nil
}
