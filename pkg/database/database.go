package database

import (
	"context"
	"fmt"
	"time"

	"github.com/MamangRust/microservice-ecommerce-pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// NewClientWithPrefix connects to the database configured via the given
// cluster prefix keys (e.g. DB_IDENTITY_HOST, DB_IDENTITY_NAME). Each bounded
// context has its own prefix so a service talks exclusively to its own
// PostgreSQL instance; there is no shared/fallback database.
//
// Host, port and database name are mandatory per context: if any is empty the
// connection fails fast instead of silently falling back to a default and
// pointing at the wrong instance. Only DB_USERNAME / DB_PASSWORD still fall
// back to the base keys.
func NewClientWithPrefix(logger logger.LoggerInterface, prefix string) (*pgxpool.Pool, error) {
	if prefix == "" {
		return nil, fmt.Errorf("database: cluster prefix must not be empty (expected one of DB_IDENTITY/DB_CATALOG/DB_MERCHANT/DB_SALES/DB_EXPERIENCE/DB_EMAIL)")
	}

	dbDriver := viper.GetString(fmt.Sprintf("%s_DRIVER", prefix))
	if dbDriver == "" {
		dbDriver = viper.GetString("DB_DRIVER")
	}

	if dbDriver != "postgres" && dbDriver != "pgx" {
		logger.Error("pgxpool only supports PostgreSQL", zap.String("DB_DRIVER", dbDriver))
		return nil, fmt.Errorf("pgxpool only supports PostgreSQL, got: %s", dbDriver)
	}

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

	// Fail-fast: a missing per-context host/port/name means the service would
	// otherwise connect to the wrong (or no) database.
	if host == "" {
		return nil, fmt.Errorf("database: %s is required (cluster prefix %s)", hostKey, prefix)
	}
	if port == "" {
		return nil, fmt.Errorf("database: %s is required (cluster prefix %s)", portKey, prefix)
	}
	if dbname == "" {
		return nil, fmt.Errorf("database: %s is required (cluster prefix %s)", nameKey, prefix)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		host, port, user, dbname, password,
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		logger.Error("Failed to parse database config", zap.Error(err))
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	maxOpenConns := viper.GetInt("DB_MAX_OPEN_CONNS")
	if maxOpenConns <= 0 {
		maxOpenConns = 100
	}
	config.MaxConns = int32(maxOpenConns)

	minIdleConns := viper.GetInt("DB_MIN_IDLE_CONNS")
	if minIdleConns <= 0 {
		minIdleConns = 50
	}
	config.MinConns = int32(minIdleConns)

	connMaxLifetime := viper.GetDuration("DB_CONN_MAX_LIFETIME")
	if connMaxLifetime == 0 {
		connMaxLifetime = time.Hour
	}
	config.MaxConnLifetime = connMaxLifetime

	connMaxIdleTime := viper.GetDuration("DB_CONN_MAX_IDLE_TIME")
	if connMaxIdleTime == 0 {
		connMaxIdleTime = 30 * time.Minute
	}
	config.MaxConnIdleTime = connMaxIdleTime

	healthCheckPeriod := viper.GetDuration("DB_HEALTH_CHECK_PERIOD")
	if healthCheckPeriod == 0 {
		healthCheckPeriod = time.Minute
	}
	config.HealthCheckPeriod = healthCheckPeriod

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		logger.Error("Failed to create connection pool", zap.Error(err))
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		logger.Error("Failed to ping database", zap.Error(err))
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Debug("Database connection pool established successfully",
		zap.String("prefix", prefix),
		zap.String("dbname", dbname),
		zap.String("DB_DRIVER", "pgx"),
		zap.Int32("MaxConns", config.MaxConns),
		zap.Int32("MinConns", config.MinConns),
		zap.Duration("MaxConnLifetime", config.MaxConnLifetime),
		zap.Duration("MaxConnIdleTime", config.MaxConnIdleTime),
		zap.Duration("HealthCheckPeriod", config.HealthCheckPeriod),
	)

	return pool, nil
}
