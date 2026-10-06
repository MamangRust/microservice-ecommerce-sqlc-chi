package cart_test

import (
	"context"
	"testing"

	cart_cache "github.com/MamangRust/microservice-ecommerce-grpc-cart/cache"
	db "github.com/MamangRust/microservice-ecommerce-grpc-cart/database/schema"
	cart_handler "github.com/MamangRust/microservice-ecommerce-grpc-cart/handler"
	cart_repo "github.com/MamangRust/microservice-ecommerce-grpc-cart/repository"
	cart_service "github.com/MamangRust/microservice-ecommerce-grpc-cart/service"
	pb_cart "github.com/MamangRust/microservice-ecommerce-grpc-pb/cart"
	pb_product "github.com/MamangRust/microservice-ecommerce-grpc-pb/product"
	pb_user "github.com/MamangRust/microservice-ecommerce-grpc-pb/user"
	"github.com/MamangRust/microservice-ecommerce-shared/cache"
	"github.com/MamangRust/microservice-ecommerce-shared/observability"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type CartGapiTestSuite struct {
	tests.BaseTestSuite
	queryClient   pb_cart.CartQueryServiceClient
	commandClient pb_cart.CartCommandServiceClient
}

func (s *CartGapiTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	// Setup dependencies
	s.SetupUserService()
	s.SetupCategoryService()
	s.SetupMerchantService()
	s.SetupProductService()

	// Infrastructure
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.RedisClient(), s.Log, cacheMetrics)
	queries := db.New(s.DBPool())

	// Cart dependencies
	mencache := cart_cache.NewMencache(cacheStore)
	repos := cart_repo.NewRepositories(
		queries,
		pb_user.NewUserQueryServiceClient(s.Conns["user"]),
		pb_product.NewProductQueryServiceClient(s.Conns["product"]),
		cart_repo.GuardOptions{},
	)
	svc := cart_service.NewService(&cart_service.Deps{
		Cache:         mencache,
		Repositories:  repos,
		Logger:        s.Log,
		Observability: s.Obs,
	})

	// Handler
	handler := cart_handler.NewHandler(&cart_handler.Deps{
		Service: svc,
		Logger:  s.Log,
	})

	// Server
	server := grpc.NewServer()
	pb_cart.RegisterCartQueryServiceServer(server, handler.CartQuery)
	pb_cart.RegisterCartCommandServiceServer(server, handler.CartCommand)

	addr := s.RegisterServer(server)
	conn := s.GetConnection(addr)

	s.queryClient = pb_cart.NewCartQueryServiceClient(conn)
	s.commandClient = pb_cart.NewCartCommandServiceClient(conn)
}

func (s *CartGapiTestSuite) TestGapiLifecycle() {
	ctx := context.Background()

	// Seed dependencies
	userID := s.SeedUser(ctx)
	categoryID := s.SeedCategory(ctx)
	merchantID := s.SeedMerchant(ctx, userID)
	prodID := s.SeedProduct(ctx, merchantID, categoryID)

	// Add to Cart
	createRes, err := s.commandClient.Create(ctx, &pb_cart.CreateCartRequest{
		UserId:    int32(userID),
		ProductId: int32(prodID),
		Quantity:  5,
	})
	s.Require().NoError(err)
	s.NotNil(createRes)

	// Get
	listRes, err := s.queryClient.FindAll(ctx, &pb_cart.FindAllCartRequest{UserId: int32(userID)})
	s.Require().NoError(err)
	s.NotEmpty(listRes.Data)
}

func TestCartGapiSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(CartGapiTestSuite))
}
