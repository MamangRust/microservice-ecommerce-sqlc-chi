// Package role is the shared gRPC adapter for the role domain. It owns the only
// place that talks to pb_role's RoleQueryService, so consuming services never
// hold a raw *ServiceClient. Role assignment lives in the user_role adapter,
// which is backed by pb_user_role's UserRoleCommandService client.
package role

import (
	"context"
	"time"

	pb_role "github.com/MamangRust/microservice-ecommerce-grpc-pb/role"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	"github.com/MamangRust/microservice-ecommerce-pkg/resilience"
)

// Role is the transport-neutral view of a role row that the adapter exposes to
// dependent services.
type Role struct {
	RoleID    int32
	RoleName  string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// QueryRepository is the read path used by dependent services.
type QueryRepository interface {
	// FindByID returns a single role or a raw gRPC error.
	FindByID(ctx context.Context, id int) (*Role, error)
	// FindByName returns the role with the given name, or a raw gRPC error.
	FindByName(ctx context.Context, name string) (*Role, error)
	// FindAll returns one page of roles plus the total record count.
	FindAll(ctx context.Context, search string, page, pageSize int) ([]Role, int, error)
}

// Repository implements QueryRepository on top of the role query gRPC client.
type Repository struct {
	query pb_role.RoleQueryServiceClient
	guard *resilience.DependencyGuard
}

// NewAdapter builds the role adapter from its query client.
func NewAdapter(query pb_role.RoleQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewQueryAdapter is the query-only constructor, kept as an alias of NewAdapter
// because the role adapter has no write path of its own (role assignment lives
// in the user_role adapter).
func NewQueryAdapter(query pb_role.RoleQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// FindByID implements QueryRepository. Errors are passed through raw so each
// caller keeps its own error mapping.
func (r *Repository) FindByID(ctx context.Context, id int) (*Role, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_role.ApiResponseRole, error) {
		return r.query.FindByIdRole(ctx, &pb_role.FindByIdRoleRequest{RoleId: int32(id)})
	})
	if err != nil {
		return nil, err
	}
	return toRole(res.Data), nil
}

// FindByName implements QueryRepository.
func (r *Repository) FindByName(ctx context.Context, name string) (*Role, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_role.ApiResponseRole, error) {
		return r.query.FindByNameRole(ctx, &pb_role.FindByNameRoleRequest{Name: name})
	})
	if err != nil {
		return nil, err
	}
	return toRole(res.Data), nil
}

// FindAll implements QueryRepository.
func (r *Repository) FindAll(ctx context.Context, search string, page, pageSize int) ([]Role, int, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_role.ApiResponsePaginationRole, error) {
		return r.query.FindAllRole(ctx, &pb_role.FindAllRoleRequest{
			Search:   search,
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
	})
	if err != nil {
		return nil, 0, err
	}

	roles := make([]Role, 0, len(res.Data))
	for _, item := range res.Data {
		roles = append(roles, *toRole(item))
	}

	total := 0
	if res.Pagination != nil {
		total = int(res.Pagination.TotalRecords)
	}
	return roles, total, nil
}

func toRole(r *pb_role.RoleResponse) *Role {
	if r == nil {
		return nil
	}
	return &Role{
		RoleID:    r.Id,
		RoleName:  r.Name,
		CreatedAt: adapter.ParseTimePtr(r.CreatedAt),
		UpdatedAt: adapter.ParseTimePtr(r.UpdatedAt),
	}
}
