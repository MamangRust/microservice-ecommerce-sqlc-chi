package repository

import (
	db "github.com/MamangRust/microservice-ecommerce-grpc-review-detail/database/schema"
)

type Repositories struct {
	ReviewDetailQuery   ReviewDetailQueryRepository
	ReviewDetailCommand ReviewDetailCommandRepository
}

func NewRepositories(DB *db.Queries) *Repositories {
	return &Repositories{
		ReviewDetailQuery:   NewReviewDetailQueryRepository(DB),
		ReviewDetailCommand: NewReviewDetailCommandRepository(DB),
	}
}
