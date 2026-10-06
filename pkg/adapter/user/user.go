// Package user is the shared gRPC adapter for the user domain. It owns the only
// place that talks to pb_user's UserQueryService and UserCommandService, so
// consuming services never hold a raw *ServiceClient.
package user

import (
	"context"
	"time"

	pb_user "github.com/MamangRust/microservice-ecommerce-grpc-pb/user"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	"github.com/MamangRust/microservice-ecommerce-pkg/resilience"
	"github.com/MamangRust/microservice-ecommerce-shared/domain/requests"
)

// User is the transport-neutral view of a user row that the adapter exposes to
// dependent services. Password is only populated by the lookups that request it
// (FindByEmail); the plain response never carries it.
type User struct {
	UserID    int32
	Firstname string
	Lastname  string
	Email     string
	Password  string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// QueryRepository is the read path used by dependent services.
type QueryRepository interface {
	// FindByID returns a single user or a raw gRPC error.
	FindByID(ctx context.Context, id int) (*User, error)
	// FindByEmail returns the user with the password hash.
	FindByEmail(ctx context.Context, email string) (*User, error)
	// FindByVerificationCode returns the user owning the given code.
	FindByVerificationCode(ctx context.Context, code string) (*User, error)
}

// CommandRepository is the write path used by dependent services. Errors are
// passed through raw so each caller keeps its own error mapping.
type CommandRepository interface {
	Create(ctx context.Context, request *requests.RegisterRequest) (*User, error)
	UpdateIsVerified(ctx context.Context, id int, isVerified bool) (*User, error)
	UpdatePassword(ctx context.Context, id int, password string) (*User, error)
}

// Repository implements the user adapter interfaces on top of the user query
// and command gRPC clients.
type Repository struct {
	query   pb_user.UserQueryServiceClient
	command pb_user.UserCommandServiceClient
	guard   *resilience.DependencyGuard
}

// NewAdapter builds the full user adapter from its query and command clients.
func NewAdapter(query pb_user.UserQueryServiceClient, command pb_user.UserCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query, command: command}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewQueryAdapter builds a query-only adapter (command client unset).
func NewQueryAdapter(query pb_user.UserQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// FindByID implements QueryRepository.
func (r *Repository) FindByID(ctx context.Context, id int) (*User, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_user.ApiResponseUser, error) {
		return r.query.FindById(ctx, &pb_user.FindByIdUserRequest{Id: int32(id)})
	})
	if err != nil {
		return nil, err
	}
	return toUser(res.Data), nil
}

// FindByEmail implements QueryRepository.
func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_user.ApiResponseUserWithPassword, error) {
		return r.query.FindByEmail(ctx, &pb_user.FindByEmailRequest{Email: email})
	})
	if err != nil {
		return nil, err
	}
	return toUserWithPassword(res.Data), nil
}

// FindByVerificationCode implements QueryRepository.
func (r *Repository) FindByVerificationCode(ctx context.Context, code string) (*User, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_user.ApiResponseUser, error) {
		return r.query.FindByVerificationCode(ctx, &pb_user.FindByVerificationCodeRequest{VerificationCode: code})
	})
	if err != nil {
		return nil, err
	}
	return toUser(res.Data), nil
}

// Create implements CommandRepository. The password must already be hashed by
// the caller; the user service stores it verbatim.
func (r *Repository) Create(ctx context.Context, request *requests.RegisterRequest) (*User, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_user.ApiResponseUser, error) {
		return r.command.Create(ctx, &pb_user.CreateUserRequest{
			Firstname:       request.FirstName,
			Lastname:        request.LastName,
			Email:           request.Email,
			Password:        request.Password,
			ConfirmPassword: request.ConfirmPassword,
		})
	})
	if err != nil {
		return nil, err
	}
	return toUser(res.Data), nil
}

// UpdateIsVerified implements CommandRepository.
func (r *Repository) UpdateIsVerified(ctx context.Context, id int, isVerified bool) (*User, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_user.ApiResponseUser, error) {
		return r.command.UpdateIsVerified(ctx, &pb_user.UpdateUserIsVerifiedRequest{
			Id:         int32(id),
			IsVerified: isVerified,
		})
	})
	if err != nil {
		return nil, err
	}
	return toUser(res.Data), nil
}

// UpdatePassword implements CommandRepository.
func (r *Repository) UpdatePassword(ctx context.Context, id int, password string) (*User, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_user.ApiResponseUser, error) {
		return r.command.UpdatePassword(ctx, &pb_user.UpdateUserPasswordRequest{
			Id:       int32(id),
			Password: password,
		})
	})
	if err != nil {
		return nil, err
	}
	return toUser(res.Data), nil
}

func toUser(u *pb_user.UserResponse) *User {
	if u == nil {
		return nil
	}
	return &User{
		UserID:    u.Id,
		Firstname: u.Firstname,
		Lastname:  u.Lastname,
		Email:     u.Email,
		CreatedAt: adapter.ParseTimePtr(u.CreatedAt),
		UpdatedAt: adapter.ParseTimePtr(u.UpdatedAt),
	}
}

func toUserWithPassword(u *pb_user.UserResponseWithPassword) *User {
	if u == nil {
		return nil
	}
	return &User{
		UserID:    u.Id,
		Firstname: u.Firstname,
		Lastname:  u.Lastname,
		Email:     u.Email,
		Password:  u.Password,
		CreatedAt: adapter.ParseTimePtr(u.CreatedAt),
		UpdatedAt: adapter.ParseTimePtr(u.UpdatedAt),
	}
}
