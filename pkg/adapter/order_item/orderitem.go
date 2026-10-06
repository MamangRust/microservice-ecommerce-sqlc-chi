// Package orderitem is the shared gRPC adapter for the order-item domain. It
// owns the only place that talks to pb_order_item's OrderItemQueryService and
// OrderItemCommandService, so consuming services never hold a raw *ServiceClient.
package orderitem

import (
	"context"
	"time"

	pb_orderitem "github.com/MamangRust/microservice-ecommerce-grpc-pb/order_item"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	"github.com/MamangRust/microservice-ecommerce-pkg/resilience"
	"github.com/MamangRust/microservice-ecommerce-shared/domain/requests"
	"google.golang.org/protobuf/types/known/emptypb"
)

// OrderItem is the transport-neutral view of an order-item row.
type OrderItem struct {
	OrderItemID int32
	OrderID     int32
	ProductID   int32
	Quantity    int32
	Price       int32
	CreatedAt   *time.Time
}

// QueryRepository is the read path used by dependent services.
type QueryRepository interface {
	// FindOrderItemByOrder returns every order item for an order.
	FindOrderItemByOrder(ctx context.Context, orderID int) ([]OrderItem, error)
	// FindAll returns one page of order items plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]OrderItem, int, error)
}

// CommandRepository is the write path used by dependent services. Errors are
// passed through raw so each caller keeps its own error mapping.
type CommandRepository interface {
	// CalculateTotalPrice returns the summed price of an order's items.
	CalculateTotalPrice(ctx context.Context, orderID int) (*int32, error)
	Create(ctx context.Context, req *requests.CreateOrderItemRecordRequest) (*OrderItem, error)
	Update(ctx context.Context, req *requests.UpdateOrderItemRecordRequest) (*OrderItem, error)
	Trash(ctx context.Context, orderID int) (*OrderItem, error)
	Restore(ctx context.Context, orderID int) (*OrderItem, error)
	DeletePermanent(ctx context.Context, orderID int) (bool, error)
	DeleteByOrderIDPermanent(ctx context.Context, orderID int) (bool, error)
	RestoreAll(ctx context.Context) (bool, error)
	DeleteAll(ctx context.Context) (bool, error)
}

// BulkRepository is the paged read path used by the stats backfill.
type BulkRepository interface {
	// FindAll returns one page of order items plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]OrderItem, int, error)
}

// Repository implements the order-item adapter interfaces on top of the
// order-item query and command gRPC clients.
type Repository struct {
	query   pb_orderitem.OrderItemQueryServiceClient
	command pb_orderitem.OrderItemCommandServiceClient
	guard   *resilience.DependencyGuard
}

// NewAdapter builds the full order-item adapter from its query and command clients.
func NewAdapter(query pb_orderitem.OrderItemQueryServiceClient, command pb_orderitem.OrderItemCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query, command: command}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewQueryAdapter builds a query-only adapter (command client unset).
func NewQueryAdapter(query pb_orderitem.OrderItemQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewBulkAdapter builds the paged read adapter for the stats backfill.
func NewBulkAdapter(query pb_orderitem.OrderItemQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// FindOrderItemByOrder implements QueryRepository.
func (r *Repository) FindOrderItemByOrder(ctx context.Context, orderID int) ([]OrderItem, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponsesOrderItem, error) {
		return r.query.FindOrderItemByOrder(ctx, &pb_orderitem.FindByIdOrderItemRequest{Id: int32(orderID)})
	})
	if err != nil {
		return nil, err
	}
	return toOrderItems(res.Data), nil
}

// CalculateTotalPrice implements QueryRepository.
func (r *Repository) CalculateTotalPrice(ctx context.Context, orderID int) (*int32, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.CalculateTotalPriceResponse, error) {
		return r.command.CalculateTotalPrice(ctx, &pb_orderitem.CalculateTotalPriceRequest{OrderId: int32(orderID)})
	})
	if err != nil {
		return nil, err
	}
	total := int32(res.TotalPrice)
	return &total, nil
}

// FindAll implements BulkRepository and QueryRepository.
func (r *Repository) FindAll(ctx context.Context, page, pageSize int) ([]OrderItem, int, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponsePaginationOrderItem, error) {
		return r.query.FindAll(ctx, &pb_orderitem.FindAllOrderItemRequest{
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
	})
	if err != nil {
		return nil, 0, err
	}
	total := 0
	if res.Pagination != nil {
		total = int(res.Pagination.TotalRecords)
	}
	return toOrderItems(res.Data), total, nil
}

// Create implements CommandRepository.
func (r *Repository) Create(ctx context.Context, req *requests.CreateOrderItemRecordRequest) (*OrderItem, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponseOrderItem, error) {
		return r.command.CreateOrderItem(ctx, &pb_orderitem.CreateOrderItemRecordRequest{
			OrderId:   int32(req.OrderID),
			ProductId: int32(req.ProductID),
			Quantity:  int32(req.Quantity),
			Price:     int32(req.Price),
		})
	})
	if err != nil {
		return nil, err
	}
	return toOrderItem(res.Data), nil
}

// Update implements CommandRepository.
func (r *Repository) Update(ctx context.Context, req *requests.UpdateOrderItemRecordRequest) (*OrderItem, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponseOrderItem, error) {
		return r.command.UpdateOrderItem(ctx, &pb_orderitem.UpdateOrderItemRecordRequest{
			OrderItemId: int32(req.OrderItemID),
			Quantity:    int32(req.Quantity),
			Price:       int32(req.Price),
		})
	})
	if err != nil {
		return nil, err
	}
	return toOrderItem(res.Data), nil
}

// Trash implements CommandRepository.
func (r *Repository) Trash(ctx context.Context, orderID int) (*OrderItem, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponseOrderItem, error) {
		return r.command.TrashOrderItem(ctx, &pb_orderitem.FindByIdOrderItemRequest{Id: int32(orderID)})
	})
	if err != nil {
		return nil, err
	}
	return toOrderItem(res.Data), nil
}

// Restore implements CommandRepository.
func (r *Repository) Restore(ctx context.Context, orderID int) (*OrderItem, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponseOrderItem, error) {
		return r.command.RestoreOrderItem(ctx, &pb_orderitem.FindByIdOrderItemRequest{Id: int32(orderID)})
	})
	if err != nil {
		return nil, err
	}
	return toOrderItem(res.Data), nil
}

// DeletePermanent implements CommandRepository.
func (r *Repository) DeletePermanent(ctx context.Context, orderID int) (bool, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponseOrderItemDelete, error) {
		return r.command.DeleteOrderItemPermanent(ctx, &pb_orderitem.FindByIdOrderItemRequest{Id: int32(orderID)})
	})
	if err != nil {
		return false, err
	}
	return res.Status == "success", nil
}

// DeleteByOrderIDPermanent implements CommandRepository.
func (r *Repository) DeleteByOrderIDPermanent(ctx context.Context, orderID int) (bool, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponseOrderItemDelete, error) {
		return r.command.DeleteOrderItemByOrderPermanent(ctx, &pb_orderitem.FindByIdOrderItemRequest{Id: int32(orderID)})
	})
	if err != nil {
		return false, err
	}
	return res.Status == "success", nil
}

// RestoreAll implements CommandRepository.
func (r *Repository) RestoreAll(ctx context.Context) (bool, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponseOrderItemAll, error) {
		return r.command.RestoreAllOrdersItem(ctx, &emptypb.Empty{})
	})
	if err != nil {
		return false, err
	}
	return res.Status == "success", nil
}

// DeleteAll implements CommandRepository.
func (r *Repository) DeleteAll(ctx context.Context) (bool, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_orderitem.ApiResponseOrderItemAll, error) {
		return r.command.DeleteAllPermanentOrdersItem(ctx, &emptypb.Empty{})
	})
	if err != nil {
		return false, err
	}
	return res.Status == "success", nil
}

func toOrderItems(data []*pb_orderitem.OrderItemResponse) []OrderItem {
	items := make([]OrderItem, 0, len(data))
	for _, it := range data {
		items = append(items, *toOrderItem(it))
	}
	return items
}

func toOrderItem(it *pb_orderitem.OrderItemResponse) *OrderItem {
	if it == nil {
		return nil
	}
	return &OrderItem{
		OrderItemID: it.Id,
		OrderID:     it.OrderId,
		ProductID:   it.ProductId,
		Quantity:    it.Quantity,
		Price:       it.Price,
		CreatedAt:   adapter.ParseTimePtr(it.CreatedAt),
	}
}
