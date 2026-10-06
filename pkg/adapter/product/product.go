// Package product is the shared gRPC adapter for the product domain. It owns
// the only place that talks to pb_product's ProductQueryService and
// ProductCommandService, so consuming services never hold a raw *ServiceClient.
package product

import (
	"context"

	pb_product "github.com/MamangRust/microservice-ecommerce-grpc-pb/product"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	"github.com/MamangRust/microservice-ecommerce-pkg/resilience"
)

// Product is the transport-neutral view of a product row that the adapters
// expose to dependent services.
type Product struct {
	ProductID   int32
	MerchantID  int32
	CategoryID  int32
	Name        string
	Description string
	Price       int32
	CountStock  int32
	Brand       string
	Weight      int32
	Rating      float32
	SlugProduct string
	ImageProduct string
}

// QueryRepository is the read path used by dependent services.
type QueryRepository interface {
	// FindByID returns a single product or a raw gRPC error.
	FindByID(ctx context.Context, id int) (*Product, error)
	// FindByMerchant returns one page of a merchant's products plus the total.
	FindByMerchant(ctx context.Context, merchantID, page, pageSize int) ([]Product, int, error)
	// FindAll returns one page of products plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]Product, int, error)
}

// CommandRepository is the write path used by dependent services. Errors are
// passed through raw so each caller keeps its own error mapping.
type CommandRepository interface {
	UpdateProductCountStock(ctx context.Context, productID, stock int) (*Product, error)
	AdjustProductStock(ctx context.Context, productID, delta int, operationID string) (*Product, error)
}

// BulkRepository is the paged read path used by the stats backfill.
type BulkRepository interface {
	// FindAll returns one page of products plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]Product, int, error)
}

// Repository implements the product adapter interfaces on top of the product
// query and command gRPC clients.
type Repository struct {
	query   pb_product.ProductQueryServiceClient
	command pb_product.ProductCommandServiceClient
	guard   *resilience.DependencyGuard
}

// NewAdapter builds the full product adapter from its query and command clients.
func NewAdapter(query pb_product.ProductQueryServiceClient, command pb_product.ProductCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query, command: command}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewQueryAdapter builds a query-only adapter (command client unset).
func NewQueryAdapter(query pb_product.ProductQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewBulkAdapter builds the paged read adapter (query client only) for stats.
func NewBulkAdapter(query pb_product.ProductQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// FindByID implements QueryRepository.
func (r *Repository) FindByID(ctx context.Context, id int) (*Product, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_product.ApiResponseProduct, error) {
		return r.query.FindById(ctx, &pb_product.FindByIdProductRequest{Id: int32(id)})
	})
	if err != nil {
		return nil, err
	}
	return toProduct(res.Data), nil
}

// FindByMerchant implements QueryRepository.
func (r *Repository) FindByMerchant(ctx context.Context, merchantID, page, pageSize int) ([]Product, int, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_product.ApiResponsePaginationProduct, error) {
		return r.query.FindByMerchant(ctx, &pb_product.FindAllProductMerchantRequest{
			MerchantId: int32(merchantID),
			Page:       int32(page),
			PageSize:   int32(pageSize),
		})
	})
	if err != nil {
		return nil, 0, err
	}
	products := make([]Product, 0, len(res.Data))
	for _, p := range res.Data {
		products = append(products, *toProduct(p))
	}
	total := 0
	if res.Pagination != nil {
		total = int(res.Pagination.TotalRecords)
	}
	return products, total, nil
}

// FindAll implements BulkRepository and QueryRepository.
func (r *Repository) FindAll(ctx context.Context, page, pageSize int) ([]Product, int, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_product.ApiResponsePaginationProduct, error) {
		return r.query.FindAll(ctx, &pb_product.FindAllProductRequest{
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
	})
	if err != nil {
		return nil, 0, err
	}
	products := make([]Product, 0, len(res.Data))
	for _, p := range res.Data {
		products = append(products, *toProduct(p))
	}
	total := 0
	if res.Pagination != nil {
		total = int(res.Pagination.TotalRecords)
	}
	return products, total, nil
}

// UpdateProductCountStock implements CommandRepository.
func (r *Repository) UpdateProductCountStock(ctx context.Context, productID, stock int) (*Product, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_product.ApiResponseProduct, error) {
		return r.command.UpdateProductCountStock(ctx, &pb_product.UpdateProductCountStockRequest{
			ProductId: int32(productID),
			Stock:     int32(stock),
		})
	})
	if err != nil {
		return nil, err
	}
	return &Product{ProductID: res.Data.Id, CountStock: res.Data.CountInStock}, nil
}

// AdjustProductStock implements CommandRepository.
func (r *Repository) AdjustProductStock(ctx context.Context, productID, delta int, operationID string) (*Product, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_product.ApiResponseProduct, error) {
		return r.command.AdjustProductStock(ctx, &pb_product.AdjustProductStockRequest{
			ProductId:   int32(productID),
			Delta:       int32(delta),
			OperationId: operationID,
		})
	})
	if err != nil {
		return nil, err
	}
	return &Product{ProductID: res.Data.Id, CountStock: res.Data.CountInStock}, nil
}

func toProduct(p *pb_product.ProductResponse) *Product {
	if p == nil {
		return nil
	}
	return &Product{
		ProductID:   p.Id,
		MerchantID:  p.MerchantId,
		CategoryID:  p.CategoryId,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		CountStock:  p.CountInStock,
		Brand:       p.Brand,
		Weight:      p.Weight,
		Rating:      p.Rating,
		SlugProduct: p.SlugProduct,
		ImageProduct: p.ImageProduct,
	}
}
