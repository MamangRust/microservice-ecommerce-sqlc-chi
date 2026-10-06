package repository

import (
	"context"

	"errors"
	"github.com/jackc/pgx/v5"

	productadapter "github.com/MamangRust/microservice-ecommerce-pkg/adapter/product"
	db "github.com/MamangRust/microservice-ecommerce-grpc-review/database/schema"
	"github.com/MamangRust/microservice-ecommerce-shared/domain/requests"
	review_errors "github.com/MamangRust/microservice-ecommerce-shared/errors/review"
)

type reviewQueryRepository struct {
	db            *db.Queries
	productClient productadapter.QueryRepository
}

func NewReviewQueryRepository(db *db.Queries, productClient productadapter.QueryRepository) *reviewQueryRepository {
	return &reviewQueryRepository{
		db:            db,
		productClient: productClient,
	}
}

func (r *reviewQueryRepository) FindAll(ctx context.Context, req *requests.FindAllReview) ([]*db.GetReviewsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetReviewsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetReviews(ctx, reqDb)

	if err != nil {
		return nil, review_errors.ErrFindAllReviews.WithInternal(err)
	}

	return res, nil
}

func (r *reviewQueryRepository) FindByProduct(ctx context.Context, req *requests.FindAllReviewByProduct) ([]*db.GetReviewByProductIdRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetReviewByProductIdParams{
		ProductID: int32(req.ProductID),
		Column2:   int32(req.Rating),
		Limit:     int32(req.PageSize),
		Offset:    int32(offset),
	}

	res, err := r.db.GetReviewByProductId(ctx, reqDb)

	if err != nil {
		return nil, review_errors.ErrFindReviewsByProduct.WithInternal(err)
	}

	return res, nil
}

func (r *reviewQueryRepository) FindByMerchant(ctx context.Context, req *requests.FindAllReviewByMerchant) ([]*db.GetReviewByMerchantIdRow, error) {
	// The review service owns no products table (per-service DB split), so the
	// merchant's product IDs are resolved via the product service gRPC before
	// querying reviews. Fetch a large page to collect every product ID.
	productRes, _, err := r.productClient.FindByMerchant(ctx, req.MerchantID, 1, 100000)
	if err != nil {
		return nil, review_errors.ErrFindReviewsByMerchant.WithInternal(err)
	}

	productIDs := make([]int32, 0, len(productRes))
	for _, p := range productRes {
		productIDs = append(productIDs, p.ProductID)
	}

	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetReviewByMerchantIdParams{
		Column1: productIDs,
		Column2: int32(req.Rating),
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetReviewByMerchantId(ctx, reqDb)

	if err != nil {
		return nil, review_errors.ErrFindReviewsByMerchant.WithInternal(err)
	}

	return res, nil
}

func (r *reviewQueryRepository) FindActive(ctx context.Context, req *requests.FindAllReview) ([]*db.GetReviewsActiveRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetReviewsActiveParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetReviewsActive(ctx, reqDb)

	if err != nil {
		return nil, review_errors.ErrFindActiveReviews.WithInternal(err)
	}

	return res, nil
}

func (r *reviewQueryRepository) FindTrashed(ctx context.Context, req *requests.FindAllReview) ([]*db.GetReviewsTrashedRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetReviewsTrashedParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetReviewsTrashed(ctx, reqDb)

	if err != nil {
		return nil, review_errors.ErrFindTrashedReviews.WithInternal(err)
	}

	return res, nil
}

func (r *reviewQueryRepository) FindByID(ctx context.Context, id int) (*db.GetReviewByIDRow, error) {
	res, err := r.db.GetReviewByID(ctx, int32(id))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, review_errors.ErrReviewNotFound
		}
		return nil, review_errors.ErrFindReviewByID.WithInternal(err)
	}

	return res, nil
}
