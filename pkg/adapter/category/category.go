// Package category is the shared gRPC adapter for the category domain. It owns
// the only place that talks to pb_category's CategoryQueryService, so consuming
// services never hold a raw *ServiceClient.
package category

import (
	"context"

	pb_category "github.com/MamangRust/microservice-ecommerce-grpc-pb/category"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	"github.com/MamangRust/microservice-ecommerce-pkg/resilience"
)

// Category is the transport-neutral view of a category row.
type Category struct {
	CategoryID  int32
	Name        string
	Description string
}

// QueryRepository is the read path used by dependent services.
type QueryRepository interface {
	// FindByID returns a single category or a raw gRPC error.
	FindByID(ctx context.Context, id int) (*Category, error)
	// FindByName resolves a category by name (first match) or a raw gRPC error.
	FindByName(ctx context.Context, name string) (*Category, error)
	// FindAll returns one page of categories plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]Category, int, error)
}

// BulkRepository is the paged read path used by the stats backfill.
type BulkRepository interface {
	// FindAll returns one page of categories plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]Category, int, error)
}

// Repository implements the category adapter interfaces on top of the category
// query gRPC client.
type Repository struct {
	query pb_category.CategoryQueryServiceClient
	guard *resilience.DependencyGuard
}

// NewAdapter builds the category adapter from its query client.
func NewAdapter(query pb_category.CategoryQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewBulkAdapter builds the paged read adapter for the category service.
func NewBulkAdapter(query pb_category.CategoryQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// FindByID implements QueryRepository.
func (r *Repository) FindByID(ctx context.Context, id int) (*Category, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_category.ApiResponseCategory, error) {
		return r.query.FindById(ctx, &pb_category.FindByIdCategoryRequest{Id: int32(id)})
	})
	if err != nil {
		return nil, err
	}
	return &Category{
		CategoryID:  res.Data.Id,
		Name:        res.Data.Name,
		Description: res.Data.Description,
	}, nil
}

// FindByName implements QueryRepository.
func (r *Repository) FindByName(ctx context.Context, name string) (*Category, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_category.ApiResponsePaginationCategory, error) {
		return r.query.FindAll(ctx, &pb_category.FindAllCategoryRequest{
			Page:     1,
			PageSize: 1,
			Search:   name,
		})
	})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	return &Category{
		CategoryID:  res.Data[0].Id,
		Name:        res.Data[0].Name,
		Description: res.Data[0].Description,
	}, nil
}

// FindAll implements BulkRepository and QueryRepository.
func (r *Repository) FindAll(ctx context.Context, page, pageSize int) ([]Category, int, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_category.ApiResponsePaginationCategory, error) {
		return r.query.FindAll(ctx, &pb_category.FindAllCategoryRequest{
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
	})
	if err != nil {
		return nil, 0, err
	}

	categories := make([]Category, 0, len(res.Data))
	for _, c := range res.Data {
		categories = append(categories, Category{
			CategoryID:  c.Id,
			Name:        c.Name,
			Description: c.Description,
		})
	}

	total := 0
	if res.Pagination != nil {
		total = int(res.Pagination.TotalRecords)
	}
	return categories, total, nil
}
