package repository

import (
	db "github.com/MamangRust/microservice-ecommerce-grpc-cart/database/schema"
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
	CartQuery    CartQueryRepository
	CartCommand  CartCommandRepository
	UserQuery    useradapter.QueryRepository
	ProductQuery productadapter.QueryRepository
}

func NewRepositories(DB *db.Queries,
	userQueryClient pb_user.UserQueryServiceClient,
	productQueryClient pb_product.ProductQueryServiceClient,
	guards ...GuardOptions,
) *Repositories {
	var g GuardOptions
	if len(guards) > 0 {
		g = guards[0]
	}

	return &Repositories{
		CartQuery:    NewCartQueryRepository(DB),
		CartCommand:  NewCartCommandRepository(DB),
		UserQuery:    useradapter.NewQueryAdapter(userQueryClient, g.User...),
		ProductQuery: productadapter.NewQueryAdapter(productQueryClient, g.Product...),
	}
}
