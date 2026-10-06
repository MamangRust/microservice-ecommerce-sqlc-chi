// Package transaction is the shared gRPC adapter for the transaction domain. It
// owns the only place that talks to pb_transaction's TransactionQueryService and
// TransactionCommandService, so consuming services never hold a raw *ServiceClient.
package transaction

import (
	"context"
	"time"

	pb_transaction "github.com/MamangRust/microservice-ecommerce-grpc-pb/transaction"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	"github.com/MamangRust/microservice-ecommerce-pkg/resilience"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Transaction is the transport-neutral view of a transaction row.
type Transaction struct {
	TransactionID int32
	OrderID       int32
	MerchantID    int32
	PaymentMethod string
	Amount        int32
	Status        string
	CreatedAt     *time.Time
}

// QueryRepository is the paged read path used by the stats backfill.
type QueryRepository interface {
	// FindAll returns one page of transactions plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]Transaction, int, error)
}

// CommandRepository is the write path used by dependent services. Errors are
// passed through raw so each caller keeps its own error mapping.
type CommandRepository interface {
	DeleteByOrderIDPermanent(ctx context.Context, orderID int) (bool, error)
	DeleteAll(ctx context.Context) (bool, error)
}

// BulkRepository is the paged read path used by the stats backfill.
type BulkRepository interface {
	// FindAll returns one page of transactions plus the total record count.
	FindAll(ctx context.Context, page, pageSize int) ([]Transaction, int, error)
}

// Repository implements the transaction adapter interfaces on top of the
// transaction query and command gRPC clients.
type Repository struct {
	query   pb_transaction.TransactionQueryServiceClient
	command pb_transaction.TransactionCommandServiceClient
	guard   *resilience.DependencyGuard
}

// NewAdapter builds the full transaction adapter from its query and command clients.
func NewAdapter(query pb_transaction.TransactionQueryServiceClient, command pb_transaction.TransactionCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query, command: command}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewCommandAdapter builds a command-only adapter (query client unset).
func NewCommandAdapter(command pb_transaction.TransactionCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{command: command}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewBulkAdapter builds the paged read adapter (query client only) for stats.
func NewBulkAdapter(query pb_transaction.TransactionQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// FindAll implements BulkRepository and QueryRepository.
func (r *Repository) FindAll(ctx context.Context, page, pageSize int) ([]Transaction, int, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_transaction.ApiResponsePaginationTransaction, error) {
		return r.query.FindAllTransactions(ctx, &pb_transaction.FindAllTransactionRequest{
			Page:     int32(page),
			PageSize: int32(pageSize),
		})
	})
	if err != nil {
		return nil, 0, err
	}

	transactions := make([]Transaction, 0, len(res.Data))
	for _, t := range res.Data {
		transactions = append(transactions, Transaction{
			TransactionID: t.Id,
			OrderID:       t.OrderId,
			MerchantID:    t.MerchantId,
			PaymentMethod: t.PaymentMethod,
			Amount:        t.Amount,
			Status:        t.PaymentStatus,
			CreatedAt:     adapter.ParseTimePtr(t.CreatedAt),
		})
	}

	total := 0
	if res.Pagination != nil {
		total = int(res.Pagination.TotalRecords)
	}
	return transactions, total, nil
}

// DeleteByOrderIDPermanent implements CommandRepository.
func (r *Repository) DeleteByOrderIDPermanent(ctx context.Context, orderID int) (bool, error) {
	_, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_transaction.ApiResponseTransactionDelete, error) {
		return r.command.DeleteTransactionByOrderPermanent(ctx, &pb_transaction.FindByIdTransactionRequest{
			Id: int32(orderID),
		})
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// DeleteAll implements CommandRepository.
func (r *Repository) DeleteAll(ctx context.Context) (bool, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pb_transaction.ApiResponseTransactionAll, error) {
		return r.command.DeleteAllTransactionPermanent(ctx, &emptypb.Empty{})
	})
	if err != nil {
		return false, err
	}
	return res.Status == "success", nil
}
