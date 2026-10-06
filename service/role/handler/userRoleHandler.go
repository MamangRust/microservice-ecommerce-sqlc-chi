package handler

import (
	"context"

	pb_role "github.com/MamangRust/microservice-ecommerce-grpc-pb/role"
	pb_user_role "github.com/MamangRust/microservice-ecommerce-grpc-pb/user_role"
	"github.com/MamangRust/microservice-ecommerce-grpc-role/service"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"
	"github.com/MamangRust/microservice-ecommerce-shared/domain/requests"
	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"github.com/MamangRust/microservice-ecommerce-shared/errors/role_errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type userRoleCommandHandler struct {
	pb_user_role.UnimplementedUserRoleCommandServiceServer
	userRoleCommand service.UserRoleCommandService
	logger          logger.LoggerInterface
}

func NewUserRoleCommandHandler(userRoleCommand service.UserRoleCommandService, logger logger.LoggerInterface) pb_user_role.UserRoleCommandServiceServer {
	return &userRoleCommandHandler{
		userRoleCommand: userRoleCommand,
		logger:          logger,
	}
}

func (s *userRoleCommandHandler) AssignRoleToUser(ctx context.Context, request *pb_user_role.AssignRoleToUserRequest) (*pb_user_role.ApiResponseUserRole, error) {
	req := &requests.CreateUserRoleRequest{
		UserId: int(request.GetUserId()),
		RoleId: int(request.GetRoleId()),
	}

	userRole, err := s.userRoleCommand.AssignRoleToUser(ctx, req)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pb_user_role.ApiResponseUserRole{
		Status:  "success",
		Message: "Successfully assigned role to user",
		Data:    mapToProtoUserRoleResponse(userRole),
	}, nil
}

func (s *userRoleCommandHandler) RemoveRoleFromUser(ctx context.Context, request *pb_user_role.RemoveRoleFromUserRequest) (*emptypb.Empty, error) {
	req := &requests.RemoveUserRoleRequest{
		UserId: int(request.GetUserId()),
		RoleId: int(request.GetRoleId()),
	}

	err := s.userRoleCommand.RemoveRoleFromUser(ctx, req)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &emptypb.Empty{}, nil
}

type userRoleQueryHandler struct {
	pb_user_role.UnimplementedUserRoleQueryServiceServer
	userRoleQuery service.UserRoleQueryService
	logger        logger.LoggerInterface
}

func NewUserRoleQueryHandler(userRoleQuery service.UserRoleQueryService, logger logger.LoggerInterface) pb_user_role.UserRoleQueryServiceServer {
	return &userRoleQueryHandler{
		userRoleQuery: userRoleQuery,
		logger:        logger,
	}
}

func (s *userRoleQueryHandler) FindByUserId(ctx context.Context, req *pb_user_role.FindByIdUserRoleRequest) (*pb_role.ApiResponsesRole, error) {
	userID := int(req.GetUserId())
	if userID == 0 {
		return nil, role_errors.ErrGrpcRoleInvalidId
	}

	roles, err := s.userRoleQuery.FindByUserId(ctx, userID)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoRoles := make([]*pb_role.RoleResponse, len(roles))
	for i, role := range roles {
		protoRoles[i] = mapToProtoRoleResponse(role)
	}

	return &pb_role.ApiResponsesRole{
		Status:  "success",
		Message: "Successfully fetched role by user id",
		Data:    protoRoles,
	}, nil
}
