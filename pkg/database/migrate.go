package database

import (
	"context"
	"fmt"

	"github.com/MamangRust/microservice-ecommerce-pkg/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// RunMigrations executes database migrations using goose against the DB
// configured via the given cluster prefix (e.g. "DB_SALES"). Host, port and
// database name are mandatory per context and fail fast when empty; only
// DB_USERNAME / DB_PASSWORD fall back to the base keys.
// path: directory containing migration files.
// service: the owning service name (e.g. "order-service" or "order"). Several
// services share one context database, so each gets its own goose version
// table; otherwise goose would reject a service whose migration versions are
// lower than a sibling's already-applied versions.
func RunMigrations(log logger.LoggerInterface, prefix, path, service string) error {
	if prefix == "" {
		return fmt.Errorf("migrate: cluster prefix must not be empty (expected one of DB_IDENTITY/DB_CATALOG/DB_MERCHANT/DB_SALES/DB_EXPERIENCE/DB_EMAIL)")
	}
	if service == "" {
		return fmt.Errorf("migrate: service name must not be empty (used to derive the goose version table)")
	}

	goose.SetTableName(MigrationTableName(service))

	hostKey := fmt.Sprintf("%s_HOST", prefix)
	portKey := fmt.Sprintf("%s_PORT", prefix)
	userKey := fmt.Sprintf("%s_USERNAME", prefix)
	nameKey := fmt.Sprintf("%s_NAME", prefix)
	passKey := fmt.Sprintf("%s_PASSWORD", prefix)

	host := viper.GetString(hostKey)
	port := viper.GetString(portKey)
	user := viper.GetString(userKey)
	if user == "" {
		user = viper.GetString("DB_USERNAME")
	}
	dbname := viper.GetString(nameKey)
	password := viper.GetString(passKey)
	if password == "" {
		password = viper.GetString("DB_PASSWORD")
	}

	if host == "" {
		return fmt.Errorf("migrate: %s is required (cluster prefix %s)", hostKey, prefix)
	}
	if port == "" {
		return fmt.Errorf("migrate: %s is required (cluster prefix %s)", portKey, prefix)
	}
	if dbname == "" {
		return fmt.Errorf("migrate: %s is required (cluster prefix %s)", nameKey, prefix)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		host, port, user, dbname, password,
	)

	db, err := goose.OpenDBWithDriver("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Failed to close database after migrations", zap.Error(err))
		}
	}()

	log.Info("Running database migrations",
		zap.String("path", path),
		zap.String("dbname", dbname),
		zap.String("prefix", prefix),
	)

	if err := goose.RunContext(context.Background(), "up", db, path); err != nil {
		return fmt.Errorf("migration 'up' failed: %w", err)
	}

	log.Info("Database migrations completed successfully")
	return nil
}
