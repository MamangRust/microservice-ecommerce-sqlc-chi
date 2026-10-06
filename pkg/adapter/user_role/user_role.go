// Package user_role is the shared gRPC adapter for the user_roles relation.
// The relation is owned by the role service, but its contract lives in its own
// pb.user_role package, so this adapter talks to the UserRoleCommandService
// client instead of the role command client.
package user_role

import (
	"context"

	pb_user_role "github.com/MamangRust/microservice-ecommerce-grpc-pb/user_role"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	"github.com/MamangRust/microservice-ecommerce-pkg/resilience"
	"github.com/MamangRust/microservice-ecommerce-shared/domain/requests"
	"google.golang.org/protobuf/types/known/emptypb"
)

// UserRole is the transport-neutral view of a user_roles row.
type UserRole struct {
	UserRoleID int32
	UserID     int32
	RoleID     int32
}

// CommandRepository is the write path used by dependent services.
type CommandRepository interface {
	AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*UserRole, error)
	RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error
}

// Repository implements CommandRepository on top of the user-role command gRPC
// client.
type Repository struct {
	command pb_user_role.UserRoleCommandServiceClient
	guard   *resilience.DependencyGuard
}

// NewAdapter builds the user-role adapter from the user-role command client.
func NewAdapter(command pb_user_role.UserRoleCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{command: command}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewCommandAdapter is an alias of NewAdapter, kept for call sites that make the
// command-only nature explicit.
func NewCommandAdapter(command pb_user_role.UserRoleCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{command: command}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// AssignRoleToUser implements CommandRepository.
func (r *Repository) AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*UserRole, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_user_role.ApiResponseUserRole, error) {
		return r.command.AssignRoleToUser(ctx, &pb_user_role.AssignRoleToUserRequest{
			UserId: int32(req.UserId),
			RoleId: int32(req.RoleId),
		})
	})
	if err != nil {
		return nil, err
	}
	return toUserRole(res.Data), nil
}

// RemoveRoleFromUser implements CommandRepository.
func (r *Repository) RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error {
	_, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*emptypb.Empty, error) {
		return r.command.RemoveRoleFromUser(ctx, &pb_user_role.RemoveRoleFromUserRequest{
			UserId: int32(req.UserId),
			RoleId: int32(req.RoleId),
		})
	})
	return err
}

func toUserRole(ur *pb_user_role.UserRoleResponse) *UserRole {
	if ur == nil {
		return nil
	}
	return &UserRole{
		UserRoleID: ur.UserRoleId,
		UserID:     ur.UserId,
		RoleID:     ur.RoleId,
	}
}
