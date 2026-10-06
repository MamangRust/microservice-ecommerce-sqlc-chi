// Package order is the shared gRPC adapter for the order domain. It owns the
// only place that talks to pb_order's OrderQueryService/OrderCommandService, so
// consuming services never hold a raw *ServiceClient.
package order

import (
	"context"
	"time"

	pb_order "github.com/MamangRust/microservice-ecommerce-grpc-pb/order"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	"github.com/MamangRust/microservice-ecommerce-pkg/resilience"
)

// Order is the transport-neutral view of an order row that the adapters expose.
type Order struct {
	OrderID    int32
	UserID     int32
	MerchantID int32
	TotalPrice int32
	CreatedAt  *time.Time
}

// QueryRepository is the read path used by dependent services.
type QueryRepository interface {
	// FindByID returns a single order or a raw gRPC error.
	FindByID(ctx context.Context, orderID int) (*Order, error)
	// FindAll returns one page of orders plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]Order, int, error)
}

// BulkRepository is the paged read path used by the stats backfill to walk
// every order without touching the order service's PostgreSQL directly.
type BulkRepository interface {
	// FindAll returns one page of orders plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]Order, int, error)
}

// Repository implements the order adapter interfaces on top of the order query
// and command gRPC clients.
type Repository struct {
	query   pb_order.OrderQueryServiceClient
	command pb_order.OrderCommandServiceClient
	guard   *resilience.DependencyGuard
}

// NewAdapter builds the full order adapter from its query and command clients.
func NewAdapter(query pb_order.OrderQueryServiceClient, command pb_order.OrderCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query, command: command}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewQueryAdapter builds a query-only adapter (command client unset).
func NewQueryAdapter(query pb_order.OrderQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewBulkAdapter builds the paged read adapter for the stats backfill.
func NewBulkAdapter(query pb_order.OrderQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// FindByID implements QueryRepository.
func (r *Repository) FindByID(ctx context.Context, orderID int) (*Order, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_order.ApiResponseOrder, error) {
		return r.query.FindById(ctx, &pb_order.FindByIdOrderRequest{Id: int32(orderID)})
	})
	if err != nil {
		return nil, err
	}
	return &Order{
		OrderID:    res.Data.Id,
		UserID:     res.Data.UserId,
		MerchantID: res.Data.MerchantId,
		TotalPrice: res.Data.TotalPrice,
		CreatedAt:  adapter.ParseTimePtr(res.Data.CreatedAt),
	}, nil
}

// FindAll implements BulkRepository and QueryRepository.
func (r *Repository) FindAll(ctx context.Context, page, pageSize int) ([]Order, int, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_order.ApiResponsePaginationOrder, error) {
		return r.query.FindAll(ctx, &pb_order.FindAllOrderRequest{
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
	})
	if err != nil {
		return nil, 0, err
	}

	orders := make([]Order, 0, len(res.Data))
	for _, o := range res.Data {
		orders = append(orders, Order{
			OrderID:    o.Id,
			UserID:     o.UserId,
			MerchantID: o.MerchantId,
			TotalPrice: o.TotalPrice,
			CreatedAt:  adapter.ParseTimePtr(o.CreatedAt),
		})
	}

	total := 0
	if res.Pagination != nil {
		total = int(res.Pagination.TotalRecords)
	}
	return orders, total, nil
}
