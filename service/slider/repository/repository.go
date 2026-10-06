package repository

import (
	db "github.com/MamangRust/microservice-ecommerce-grpc-slider/database/schema"
)

type Repositories struct {
	SliderQuery   SliderQueryRepository
	SliderCommand SliderCommandRepository
}

func NewRepositories(DB *db.Queries) *Repositories {
	return &Repositories{
		SliderQuery:   NewSliderQueryRepository(DB),
		SliderCommand: NewSliderCommandRepository(DB),
	}
}
