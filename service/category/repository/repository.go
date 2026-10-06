package repository

import (
	db "github.com/MamangRust/microservice-ecommerce-grpc-category/database/schema"
)

type Repositories struct {
	CategoryQuery           CategoryQueryRepository
	CategoryCommand         CategoryCommandRepository
	// F5: legacy OLTP stats repositories removed.
}

func NewRepositories(DB *db.Queries) *Repositories {
	return &Repositories{
		CategoryQuery:           NewCategoryQueryRepository(DB),
		CategoryCommand:         NewCategoryCommandRepository(DB),

	}
}
