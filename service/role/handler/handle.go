package handler

import (
	"github.com/MamangRust/microservice-ecommerce-grpc-pb/role"
	pb_user_role "github.com/MamangRust/microservice-ecommerce-grpc-pb/user_role"
	"github.com/MamangRust/microservice-ecommerce-grpc-role/service"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"
)

type Deps struct {
	Service *service.Service
	Logger  logger.LoggerInterface
}

type Handler struct {
	RoleQuery       pb_role.RoleQueryServiceServer
	RoleCommand     pb_role.RoleCommandServiceServer
	UserRoleQuery   pb_user_role.UserRoleQueryServiceServer
	UserRoleCommand pb_user_role.UserRoleCommandServiceServer
}

func NewHandler(deps *Deps) *Handler {
	return &Handler{
		RoleQuery:       NewRoleQueryHandler(deps.Service.RoleQuery, deps.Logger),
		RoleCommand:     NewRoleCommandHandler(deps.Service.RoleCommand, deps.Logger),
		UserRoleQuery:   NewUserRoleQueryHandler(deps.Service.UserRoleQuery, deps.Logger),
		UserRoleCommand: NewUserRoleCommandHandler(deps.Service.UserRoleCommand, deps.Logger),
	}
}
