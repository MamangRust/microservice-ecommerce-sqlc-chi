package repository

import (
	db "github.com/MamangRust/microservice-ecommerce-grpc-review/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	pb_product "github.com/MamangRust/microservice-ecommerce-grpc-pb/product"
	pb_user "github.com/MamangRust/microservice-ecommerce-grpc-pb/user"
	productadapter "github.com/MamangRust/microservice-ecommerce-pkg/adapter/product"
	useradapter "github.com/MamangRust/microservice-ecommerce-pkg/adapter/user"
)

type GuardOptions struct {
	User    []adapter.GuardOption
	Product []adapter.GuardOption
}

type Repositories struct {
	ReviewQuery   ReviewQueryRepository
	ReviewCommand ReviewCommandRepository
	UserQuery     useradapter.QueryRepository
	ProductQuery  productadapter.QueryRepository
}

func NewRepositories(db *db.Queries,
	userQueryClient pb_user.UserQueryServiceClient,
	productQueryClient pb_product.ProductQueryServiceClient,
	guards ...GuardOptions,
) *Repositories {
	var g GuardOptions
	if len(guards) > 0 {
		g = guards[0]
	}

	userAdapter := useradapter.NewQueryAdapter(userQueryClient, g.User...)
	productAdapter := productadapter.NewQueryAdapter(productQueryClient, g.Product...)

	return &Repositories{
		ReviewQuery:   NewReviewQueryRepository(db, productAdapter),
		ReviewCommand: NewReviewCommandRepository(db),
		UserQuery:     userAdapter,
		ProductQuery:  productAdapter,
	}
}
