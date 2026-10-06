package repository

import (
	db "github.com/MamangRust/microservice-ecommerce-grpc-order-item/database/schema"
)

type Repositories struct {
	OrderItemQuery   OrderItemQueryRepository
	OrderItemCommand OrderItemCommandRepository
}

func NewRepositories(DB *db.Queries) *Repositories {
	return &Repositories{
		OrderItemQuery:   NewOrderItemQueryRepository(DB),
		OrderItemCommand: NewOrderItemCommandRepository(DB),
	}
}
