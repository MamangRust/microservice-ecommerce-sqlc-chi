package service

import (
	"context"

	"github.com/MamangRust/microservice-ecommerce-grpc-role/cache"
	db "github.com/MamangRust/microservice-ecommerce-grpc-role/database/schema"
	"github.com/MamangRust/microservice-ecommerce-grpc-role/repository"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"
	"github.com/MamangRust/microservice-ecommerce-shared/domain/requests"
	"github.com/MamangRust/microservice-ecommerce-shared/errorhandler"
	"github.com/MamangRust/microservice-ecommerce-shared/errors/role_errors"
	"github.com/MamangRust/microservice-ecommerce-shared/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type userRoleCommandService struct {
	observability      observability.TraceLoggerObservability
	cache              cache.RoleCommandCache
	userRoleRepository repository.UserRoleRepository
	logger             logger.LoggerInterface
}

type UserRoleCommandServiceDeps struct {
	Observability      observability.TraceLoggerObservability
	Cache              cache.RoleCommandCache
	UserRoleRepository repository.UserRoleRepository
	Logger             logger.LoggerInterface
}

func NewUserRoleCommandService(deps *UserRoleCommandServiceDeps) UserRoleCommandService {
	return &userRoleCommandService{
		observability:      deps.Observability,
		cache:              deps.Cache,
		userRoleRepository: deps.UserRoleRepository,
		logger:             deps.Logger,
	}
}

func (s *userRoleCommandService) AssignRoleToUser(ctx context.Context, request *requests.CreateUserRoleRequest) (*db.UserRole, error) {
	const method = "AssignRoleToUser"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", request.UserId),
		attribute.Int("role_id", request.RoleId))

	defer func() {
		end(status)
	}()

	userRole, err := s.userRoleRepository.AssignRoleToUser(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UserRole](
			s.logger,
			err,
			method,
			span,
			zap.Int("user_id", request.UserId),
			zap.Int("role_id", request.RoleId),
		)
	}

	logSuccess("Successfully assigned role to user", zap.Int("user_id", request.UserId), zap.Int("role_id", request.RoleId))

	return userRole, nil
}

func (s *userRoleCommandService) RemoveRoleFromUser(ctx context.Context, request *requests.RemoveUserRoleRequest) error {
	const method = "RemoveRoleFromUser"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", request.UserId),
		attribute.Int("role_id", request.RoleId))

	defer func() {
		end(status)
	}()

	err := s.userRoleRepository.RemoveRoleFromUser(ctx, request)
	if err != nil {
		status = "error"
		_, err := errorhandler.HandleError[any](
			s.logger,
			err,
			method,
			span,
			zap.Int("user_id", request.UserId),
			zap.Int("role_id", request.RoleId),
		)
		return err
	}

	logSuccess("Successfully removed role from user", zap.Int("user_id", request.UserId), zap.Int("role_id", request.RoleId))

	return nil
}

type userRoleQueryService struct {
	observability      observability.TraceLoggerObservability
	cache              cache.RoleQueryCache
	userRoleRepository repository.UserRoleRepository
	logger             logger.LoggerInterface
}

type UserRoleQueryServiceDeps struct {
	Observability      observability.TraceLoggerObservability
	Cache              cache.RoleQueryCache
	UserRoleRepository repository.UserRoleRepository
	Logger             logger.LoggerInterface
}

func NewUserRoleQueryService(deps *UserRoleQueryServiceDeps) UserRoleQueryService {
	return &userRoleQueryService{
		observability:      deps.Observability,
		cache:              deps.Cache,
		userRoleRepository: deps.UserRoleRepository,
		logger:             deps.Logger,
	}
}

func (s *userRoleQueryService) FindByUserId(ctx context.Context, id int) ([]*db.Role, error) {
	const method = "FindByUserId"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user.id", id))

	defer func() {
		end(status)
	}()

	if data, found := s.cache.GetCachedRoleByUserId(ctx, id); found {
		logSuccess("Data found in cache", zap.Int("user.id", id))
		return data, nil
	}

	res, err := s.userRoleRepository.FindByUserId(ctx, id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[[]*db.Role](
			s.logger,
			role_errors.ErrRoleNotFound,
			method,
			span,
			zap.Int("user.id", id),
		)
	}

	s.cache.SetCachedRoleByUserId(ctx, id, res)

	logSuccess("Successfully fetched role by user ID", zap.Int("user.id", id))

	return res, nil
}
