package merchant_award_test

import (
	"context"
	"testing"

	award_cache "github.com/MamangRust/microservice-ecommerce-grpc-merchant_award/cache"
	db "github.com/MamangRust/microservice-ecommerce-grpc-merchant_award/database/schema"
	award_handler "github.com/MamangRust/microservice-ecommerce-grpc-merchant_award/handler"
	award_repo "github.com/MamangRust/microservice-ecommerce-grpc-merchant_award/repository"
	award_service "github.com/MamangRust/microservice-ecommerce-grpc-merchant_award/service"
	pb_merchant "github.com/MamangRust/microservice-ecommerce-grpc-pb/merchant"
	pb_merchant_award "github.com/MamangRust/microservice-ecommerce-grpc-pb/merchant_award"
	"github.com/MamangRust/microservice-ecommerce-shared/cache"
	"github.com/MamangRust/microservice-ecommerce-shared/observability"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MerchantAwardGapiTestSuite struct {
	tests.BaseTestSuite
	queryClient   pb_merchant_award.MerchantAwardQueryServiceClient
	commandClient pb_merchant_award.MerchantAwardCommandServiceClient
}

func (s *MerchantAwardGapiTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	// Setup dependencies
	s.SetupUserService()
	s.SetupMerchantService()

	// Infrastructure
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.RedisClient(), s.Log, cacheMetrics)
	queries := db.New(s.DBPool())

	// Award dependencies
	mencache := award_cache.NewMencache(cacheStore)
	repos := award_repo.NewRepositories(
		queries,
		pb_merchant.NewMerchantQueryServiceClient(s.Conns["merchant"]),
		award_repo.GuardOptions{},
	)
	svc := award_service.NewService(&award_service.Deps{
		Cache:         mencache,
		Repository:    repos,
		Logger:        s.Log,
		Observability: s.Obs,
	})

	// Handler
	handler := award_handler.NewHandler(&award_handler.Deps{
		Service: svc,
		Logger:  s.Log,
	})

	// Server
	server := grpc.NewServer()
	pb_merchant_award.RegisterMerchantAwardQueryServiceServer(server, handler.MerchantAwardQuery)
	pb_merchant_award.RegisterMerchantAwardCommandServiceServer(server, handler.MerchantAwardCommand)

	addr := s.RegisterServer(server)
	conn := s.GetConnection(addr)

	s.queryClient = pb_merchant_award.NewMerchantAwardQueryServiceClient(conn)
	s.commandClient = pb_merchant_award.NewMerchantAwardCommandServiceClient(conn)
}

func (s *MerchantAwardGapiTestSuite) TestMerchantAwardGapiLifecycle() {
	ctx := context.Background()

	// 1. Seed dependencies
	userID := s.SeedUser(ctx)
	merchID := int32(s.SeedMerchant(ctx, userID))

	// 2. Create
	createRes, err := s.commandClient.Create(ctx, &pb_merchant_award.CreateMerchantAwardRequest{
		MerchantId:  merchID,
		Title:       "GAPI Achievement",
		Description: "Detailed description of the achievement.",
		IssuedBy:    "Board",
		IssueDate:   "2024-06-01",
	})
	s.NoError(err)
	s.NotNil(createRes)
	awardID := createRes.Data.Id

	// 3. FindById
	getRes, err := s.queryClient.FindById(ctx, &pb_merchant_award.FindByIdMerchantAwardRequest{Id: awardID})
	s.NoError(err)
	s.Equal("GAPI Achievement", getRes.Data.Title)

	// 4. FindAll
	allRes, err := s.queryClient.FindAll(ctx, &pb_merchant.FindAllMerchantRequest{Page: 1, PageSize: 10})
	s.NoError(err)
	s.NotEmpty(allRes.Data)

	// 5. FindByActive
	activeRes, err := s.queryClient.FindByActive(ctx, &pb_merchant.FindAllMerchantRequest{Page: 1, PageSize: 10})
	s.NoError(err)
	s.NotEmpty(activeRes.Data)

	// 6. Update
	updateRes, err := s.commandClient.Update(ctx, &pb_merchant_award.UpdateMerchantAwardRequest{
		MerchantCertificationId: awardID,
		Title:                   "GAPI Achievement Updated",
		Description:             "Updated description.",
		IssuedBy:                "New Board",
		IssueDate:               "2024-06-01",
	})
	s.NoError(err)
	s.Equal("GAPI Achievement Updated", updateRes.Data.Title)

	// 7. Trash
	_, err = s.commandClient.TrashedMerchantAward(ctx, &pb_merchant_award.FindByIdMerchantAwardRequest{Id: awardID})
	s.NoError(err)

	// 8. FindByTrashed
	trashedRes, err := s.queryClient.FindByTrashed(ctx, &pb_merchant.FindAllMerchantRequest{Page: 1, PageSize: 10})
	s.NoError(err)
	s.NotEmpty(trashedRes.Data)

	// 9. Restore
	_, err = s.commandClient.RestoreMerchantAward(ctx, &pb_merchant_award.FindByIdMerchantAwardRequest{Id: awardID})
	s.NoError(err)

	// 10. DeletePermanent
	_, _ = s.commandClient.TrashedMerchantAward(ctx, &pb_merchant_award.FindByIdMerchantAwardRequest{Id: awardID})
	_, err = s.commandClient.DeleteMerchantAwardPermanent(ctx, &pb_merchant_award.FindByIdMerchantAwardRequest{Id: awardID})
	s.NoError(err)

	// 11. RestoreAll
	_, err = s.commandClient.RestoreAllMerchantAward(ctx, &emptypb.Empty{})
	s.NoError(err)

	// 12. DeleteAll
	_, err = s.commandClient.DeleteAllMerchantAwardPermanent(ctx, &emptypb.Empty{})
	s.NoError(err)
}

func TestMerchantAwardGapiSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantAwardGapiTestSuite))
}
