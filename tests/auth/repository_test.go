package auth_test

import (
	"context"
	"testing"
	"time"

	db "github.com/MamangRust/microservice-ecommerce-auth/database/schema"
	"github.com/MamangRust/microservice-ecommerce-auth/repository"
	pb_role "github.com/MamangRust/microservice-ecommerce-grpc-pb/role"
	pb_user_role "github.com/MamangRust/microservice-ecommerce-grpc-pb/user_role"
	pb_user "github.com/MamangRust/microservice-ecommerce-grpc-pb/user"
	roledb "github.com/MamangRust/microservice-ecommerce-grpc-role/database/schema"
	userdb "github.com/MamangRust/microservice-ecommerce-grpc-user/database/schema"
	"github.com/MamangRust/microservice-ecommerce-shared/domain/requests"
	tests "github.com/MamangRust/microservice-ecommerce-test"

	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/MamangRust/microservice-ecommerce-pkg/hash"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"
	"github.com/MamangRust/microservice-ecommerce-shared/cache"
	"github.com/MamangRust/microservice-ecommerce-shared/observability"

	role_cache "github.com/MamangRust/microservice-ecommerce-grpc-role/cache"
	role_handler "github.com/MamangRust/microservice-ecommerce-grpc-role/handler"
	role_repo "github.com/MamangRust/microservice-ecommerce-grpc-role/repository"
	role_service "github.com/MamangRust/microservice-ecommerce-grpc-role/service"
	user_cache "github.com/MamangRust/microservice-ecommerce-grpc-user/cache"
	user_handler "github.com/MamangRust/microservice-ecommerce-grpc-user/handler"
	user_repo "github.com/MamangRust/microservice-ecommerce-grpc-user/repository"
	user_service "github.com/MamangRust/microservice-ecommerce-grpc-user/service"
)

type AuthRepositoryTestSuite struct {
	suite.Suite
	ts          *tests.TestSuite
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
	repo        *repository.Repositories
	userID      int
	email       string
}

func (s *AuthRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	queries := db.New(pool)

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	hasher := hash.NewHashingPassword()
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)
	obs, _ := observability.NewObservability("test", log)

	// 1. Setup Role Service & gRPC Server
	roleMencache := role_cache.NewMencache(cacheStore)
	roleRepos := role_repo.NewRepositories(roledb.New(pool))
	roleSvc := role_service.NewService(&role_service.Deps{
		Repository:    roleRepos,
		Logger:        log,
		Cache:         roleMencache,
		Observability: obs,
	})
	roleGapi := role_handler.NewHandler(&role_handler.Deps{
		Service: roleSvc,
		Logger:  log,
	})
	roleServer := grpc.NewServer()
	pb_role.RegisterRoleQueryServiceServer(roleServer, roleGapi.RoleQuery)
	pb_role.RegisterRoleCommandServiceServer(roleServer, roleGapi.RoleCommand)
	pb_user_role.RegisterUserRoleQueryServiceServer(roleServer, roleGapi.UserRoleQuery)
	pb_user_role.RegisterUserRoleCommandServiceServer(roleServer, roleGapi.UserRoleCommand)
	roleLis, _ := net.Listen("tcp", "localhost:0")
	go roleServer.Serve(roleLis)
	roleConn, _ := grpc.NewClient(roleLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))

	// 2. Setup User Service & gRPC Server
	userMencache := user_cache.NewMencache(cacheStore)
	userRepos := user_repo.NewRepositories(&user_repo.Deps{
		Db:       userdb.New(pool),
		Role:     pb_role.NewRoleQueryServiceClient(roleConn),
		UserRole: pb_user_role.NewUserRoleCommandServiceClient(roleConn),
		Guards:   user_repo.GuardOptions{},
	})
	userSvc := user_service.NewService(&user_service.Deps{
		Repositories:  userRepos,
		Logger:        log,
		Hash:          hasher,
		Cache:         userMencache,
		Observability: obs,
	})
	userGapi := user_handler.NewHandler(&user_handler.Deps{
		Service: userSvc,
		Logger:  log,
	})
	userServer := grpc.NewServer()
	pb_user.RegisterUserQueryServiceServer(userServer, userGapi.UserQuery)
	pb_user.RegisterUserCommandServiceServer(userServer, userGapi.UserCommand)
	userLis, _ := net.Listen("tcp", "localhost:0")
	go userServer.Serve(userLis)
	userConn, _ := grpc.NewClient(userLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))

	// 3. Setup Auth Repository with gRPC clients
	userQueryClient := pb_user.NewUserQueryServiceClient(userConn)
	userCommandClient := pb_user.NewUserCommandServiceClient(userConn)
	roleQueryClient := pb_role.NewRoleQueryServiceClient(roleConn)
	roleCommandClient := pb_user_role.NewUserRoleCommandServiceClient(roleConn)

	s.repo = repository.NewRepositories(&repository.Deps{
		Db:              queries,
		User:            userQueryClient,
		UserCommand:     userCommandClient,
		Role:            roleQueryClient,
		UserRoleCommand: roleCommandClient,
		Guards:          repository.GuardOptions{},
	})
	s.email = "auth.repo.test@example.com"

	// auth no longer owns the users table, so the fixture user has to be created
	// through the user service adapter before the lookups below can resolve it.
	createdUser, err := s.repo.UserCommand.Create(context.Background(), &requests.RegisterRequest{
		FirstName:       "Auth",
		LastName:        "Repo",
		Email:           s.email,
		Password:        "password123",
		ConfirmPassword: "password123",
	})
	s.Require().NoError(err)
	s.Require().NotNil(createdUser)
	s.userID = int(createdUser.UserID)
}

func (s *AuthRepositoryTestSuite) TearDownSuite() {
	if s.redisClient != nil {
		s.redisClient.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	s.ts.Teardown()
}

func (s *AuthRepositoryTestSuite) Test2_FindByEmail() {
	s.Require().NotEmpty(s.email)
	ctx := context.Background()

	found, err := s.repo.User.FindByEmail(ctx, s.email)
	s.NoError(err)
	s.NotNil(found)
	s.Equal(int32(s.userID), found.UserID)
}

func (s *AuthRepositoryTestSuite) Test4_RefreshToken() {
	s.Require().NotZero(s.userID)
	ctx := context.Background()

	token := "test-refresh-token"
	expiresAt := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")

	req := &requests.CreateRefreshToken{
		UserId:    s.userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	res, err := s.repo.RefreshToken.CreateRefreshToken(ctx, req)
	s.NoError(err)
	s.NotNil(res)
	s.Equal(token, res.Token)

	found, err := s.repo.RefreshToken.FindByToken(ctx, token)
	s.NoError(err)
	s.NotNil(found)

	err = s.repo.RefreshToken.DeleteRefreshToken(ctx, token)
	s.NoError(err)
}

func (s *AuthRepositoryTestSuite) Test5_ResetToken() {
	s.Require().NotZero(s.userID)
	ctx := context.Background()

	token := "reset-token-123"
	expiresAt := time.Now().Add(1 * time.Hour).Format("2006-01-02 15:04:05")

	req := &requests.CreateResetTokenRequest{
		UserID:     s.userID,
		ResetToken: token,
		ExpiredAt:  expiresAt,
	}

	res, err := s.repo.ResetToken.CreateResetToken(ctx, req)
	s.NoError(err)
	s.NotNil(res)
	s.Equal(token, res.Token)

	found, err := s.repo.ResetToken.FindByToken(ctx, token)
	s.NoError(err)
	s.NotNil(found)

	err = s.repo.ResetToken.DeleteResetToken(ctx, s.userID)
	s.NoError(err)
}

func TestAuthRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(AuthRepositoryTestSuite))
}
