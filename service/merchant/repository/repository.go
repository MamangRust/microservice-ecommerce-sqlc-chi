package repository

import (
	db "github.com/MamangRust/microservice-ecommerce-grpc-merchant/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	pb_user "github.com/MamangRust/microservice-ecommerce-grpc-pb/user"
	useradapter "github.com/MamangRust/microservice-ecommerce-pkg/adapter/user"
)

type GuardOptions struct {
	User []adapter.GuardOption
}

type Repositories struct {
	MerchantQuery           MerchantQueryRepository
	MerchantCommand         MerchantCommandRepository
	MerchantDocumentCommand MerchantDocumentCommandRepository
	MerchantDocumentQuery   MerchantDocumentQueryRepository
	UserQuery               useradapter.QueryRepository
}

func NewRepositories(DB *db.Queries, userQueryClient pb_user.UserQueryServiceClient, guards ...GuardOptions) *Repositories {
	var g GuardOptions

	if len(guards) > 0 {
		g = guards[0]
	}

	return &Repositories{
		MerchantQuery:           NewMerchantQueryRepository(DB),
		MerchantCommand:         NewMerchantCommandRepository(DB),
		MerchantDocumentCommand: NewMerchantDocumentCommandRepository(DB),
		MerchantDocumentQuery:   NewMerchantDocumentQueryRepository(DB),
		UserQuery:               useradapter.NewQueryAdapter(userQueryClient, g.User...),
	}
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
