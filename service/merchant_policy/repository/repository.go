package repository

import (
	db "github.com/MamangRust/microservice-ecommerce-grpc-merchant_policy/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/adapter"
	pb_merchant "github.com/MamangRust/microservice-ecommerce-grpc-pb/merchant"
	merchantadapter "github.com/MamangRust/microservice-ecommerce-pkg/adapter/merchant"
)

type GuardOptions struct {
	Merchant []adapter.GuardOption
}

type Repositories struct {
	MerchantPoliciesQuery   MerchantPoliciesQueryRepository
	MerchantPoliciesCommand MerchantPoliciesCommandRepository
	MerchantQuery           merchantadapter.QueryRepository
}

func NewRepositories(db *db.Queries, merchantQueryClient pb_merchant.MerchantQueryServiceClient, guards ...GuardOptions) *Repositories {
	var g GuardOptions
	if len(guards) > 0 {
		g = guards[0]
	}

	return &Repositories{
		MerchantPoliciesQuery:   NewMerchantPolicyQueryRepository(db),
		MerchantPoliciesCommand: NewMerchantPolicyCommandRepository(db),
		MerchantQuery:           merchantadapter.NewQueryAdapter(merchantQueryClient, g.Merchant...),
	}
}
