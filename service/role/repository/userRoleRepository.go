package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-ecommerce-grpc-role/database/schema"
	"github.com/MamangRust/microservice-ecommerce-shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-ecommerce-shared/errors"
	"github.com/MamangRust/microservice-ecommerce-shared/errors/role_errors"
)

type userRoleRepository struct {
	db *db.Queries
}

func NewUserRoleRepository(db *db.Queries) UserRoleRepository {
	return &userRoleRepository{
		db: db,
	}
}

func (r *userRoleRepository) AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*db.UserRole, error) {
	arg := db.AssignRoleToUserParams{
		UserID: int32(req.UserId),
		RoleID: int32(req.RoleId),
	}

	res, err := r.db.AssignRoleToUser(ctx, arg)
	if err != nil {
		return nil, role_errors.ErrAssignRole.WithInternal(err)
	}

	return res, nil
}

func (r *userRoleRepository) RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error {
	arg := db.RemoveRoleFromUserParams{
		UserID: int32(req.UserId),
		RoleID: int32(req.RoleId),
	}

	err := r.db.RemoveRoleFromUser(ctx, arg)
	if err != nil {
		return role_errors.ErrRemoveRole.WithInternal(err)
	}

	return nil
}

func (r *userRoleRepository) FindByUserId(ctx context.Context, user_id int) ([]*db.Role, error) {
	res, err := r.db.GetUserRoles(ctx, int32(user_id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, role_errors.ErrRoleNotFound.WithInternal(err)
		}

		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}
