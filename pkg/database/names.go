// Package database owns the connection logic for the 6 physically separate
// PostgreSQL instances, one per bounded context. This file is the single
// source of truth for the database names and the env-prefix (DBCluster) used
// to resolve each context's DB_* connection keys.
package database

import (
	"sort"
	"strings"
)

// Database names — one PostgreSQL instance per bounded context. Each instance
// serves exactly one of these databases.
const (
	IdentityDB   = "ec_identity"
	CatalogDB    = "ec_catalog"
	MerchantDB   = "ec_merchant"
	SalesDB      = "ec_sales"
	ExperienceDB = "ec_experience"
	EmailDB      = "ec_email"
)

// Cluster prefixes — the env prefix each service uses to resolve its own
// context's DB_HOST / DB_PORT / DB_NAME / DB_USERNAME / DB_PASSWORD. Every
// service in a context shares the same prefix, so they all talk to the same
// PostgreSQL instance.
const (
	IdentityCluster   = "DB_IDENTITY"
	CatalogCluster    = "DB_CATALOG"
	MerchantCluster   = "DB_MERCHANT"
	SalesCluster      = "DB_SALES"
	ExperienceCluster = "DB_EXPERIENCE"
	EmailCluster      = "DB_EMAIL"
)

// Clusters is the ordered list of context prefixes, used by the context-aware
// migrator (service/migrate) and any tooling that must iterate all six
// databases. The order follows dependency boundaries: identity → merchant →
// catalog → sales → experience → email.
var Clusters = []string{
	IdentityCluster,
	CatalogCluster,
	MerchantCluster,
	SalesCluster,
	ExperienceCluster,
	EmailCluster,
}

// ServiceCluster maps each DB-backed service (its directory name under
// service/) to the cluster prefix of the context that owns its tables. This is
// the source of truth used by the context-aware migrator; the per-service
// main.go files set the same value via the exported *Cluster constants above.
//
// Keep this in sync with the per-context table ownership documented in the
// repository summary: identity(auth,user,role), catalog(category,product),
// merchant(merchant,merchant_award,merchant_business,merchant_detail,
// merchant_policy), sales(order,order_item,transaction),
// experience(cart,shipping_address,banner,slider,review,review_detail),
// email(email).
var ServiceCluster = map[string]string{
	"auth":              IdentityCluster,
	"user":              IdentityCluster,
	"role":              IdentityCluster,
	"category":          CatalogCluster,
	"product":           CatalogCluster,
	"merchant":          MerchantCluster,
	"merchant_award":    MerchantCluster,
	"merchant_business": MerchantCluster,
	"merchant_detail":   MerchantCluster,
	"merchant_policy":   MerchantCluster,
	"order":             SalesCluster,
	"order_item":        SalesCluster,
	"transaction":       SalesCluster,
	"cart":              ExperienceCluster,
	"shipping_address":  ExperienceCluster,
	"banner":            ExperienceCluster,
	"slider":            ExperienceCluster,
	"review":            ExperienceCluster,
	"review_detail":     ExperienceCluster,
	"email":             EmailCluster,
}

// ServicesForCluster returns the services that own tables in the given context,
// sorted for deterministic iteration by the migrator.
func ServicesForCluster(cluster string) []string {
	var services []string
	for svc, c := range ServiceCluster {
		if c == cluster {
			services = append(services, svc)
		}
	}
	sort.Strings(services)
	return services
}

// MigrationTableName derives the goose version table for a service so that
// services sharing one context database each track their own migrations. Without
// this, goose sees a sibling service's higher version and fails with "found N
// missing migrations". Both the long form ("merchant_award-service") and the
// short directory form ("merchant_award") map to the same table.
func MigrationTableName(service string) string {
	s := strings.TrimSuffix(service, "-service")
	s = strings.ReplaceAll(s, "-", "_")
	return "goose_db_version_" + s
}
