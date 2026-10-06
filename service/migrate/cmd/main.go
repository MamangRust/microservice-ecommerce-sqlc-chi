package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	"github.com/MamangRust/microservice-ecommerce-pkg/dotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/viper"
)

const dialect = "pgx"

var (
	flags = flag.NewFlagSet("migrate", flag.ExitOnError)
	root  = flags.String("root", ".", "directory holding the service/ tree with per-service migrations")
)

func main() {
	flags.Usage = usage
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	args := flags.Args()
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		flags.Usage()
		return
	}

	command := args[0]

	if err := dotenv.Viper(); err != nil {
		log.Fatalf("Error loading environment variables: %v", err)
	}

	ctx := context.Background()
	for _, cluster := range database.Clusters {
		if err := migrateCluster(ctx, cluster, command, args[1:]...); err != nil {
			log.Fatalf("Migration failed for %s: %v", cluster, err)
		}
	}
}

// migrateCluster runs the goose command against one context database for every
// service that owns tables in it. Each service keeps its own goose version
// table so sibling services sharing the database never see each other's
// versions (which would otherwise trigger goose's "missing migrations" error).
func migrateCluster(ctx context.Context, cluster, command string, extra ...string) error {
	dsn, err := contextDSN(cluster)
	if err != nil {
		return err
	}

	db, err := goose.OpenDBWithDriver(dialect, dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing %s database: %v", cluster, err)
		}
	}()

	for _, svc := range database.ServicesForCluster(cluster) {
		dir := filepath.Join(*root, "service", svc, "database", "migration")
		if _, err := os.Stat(dir); err != nil {
			log.Printf("skip %s/%s: no migration directory at %s", cluster, svc, dir)
			continue
		}

		goose.SetTableName(database.MigrationTableName(svc))
		log.Printf("goose %s: %s (%s)", command, svc, dir)
		if err := goose.RunContext(ctx, command, db, dir, extra...); err != nil {
			return fmt.Errorf("%s: %w", svc, err)
		}
	}

	return nil
}

// contextDSN builds the DSN for a context from its mandatory DB_<CTX>_* keys,
// falling back to the shared DB_USERNAME/DB_PASSWORD. It fails fast when a
// required key is missing rather than connecting to the wrong instance.
func contextDSN(cluster string) (string, error) {
	host := viper.GetString(cluster + "_HOST")
	port := viper.GetString(cluster + "_PORT")
	name := viper.GetString(cluster + "_NAME")
	user := viper.GetString(cluster + "_USERNAME")
	password := viper.GetString(cluster + "_PASSWORD")

	if user == "" {
		user = viper.GetString("DB_USERNAME")
	}
	if password == "" {
		password = viper.GetString("DB_PASSWORD")
	}
	if host == "" || port == "" || name == "" {
		return "", fmt.Errorf("%s_HOST, %s_PORT and %s_NAME are required", cluster, cluster, cluster)
	}

	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		host, port, user, name, password), nil
}

func usage() {
	fmt.Println(usagePrefix)
	flags.PrintDefaults()
	fmt.Println(usageCommands)
}

var (
	usagePrefix = `Usage: migrate [-root DIR] COMMAND
The command is applied to all six bounded-context databases (DB_IDENTITY,
DB_CATALOG, DB_MERCHANT, DB_SALES, DB_EXPERIENCE, DB_EMAIL) in dependency order.
Examples:
    migrate up
    migrate status
    migrate -root /app up
`

	usageCommands = `
Commands:
    up                   Migrate the DB to the most recent version available
    up-by-one            Migrate the DB up by 1
    up-to VERSION        Migrate the DB to a specific VERSION
    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION
    redo                 Re-run the latest migration
    reset                Roll back all migrations
    status               Dump the migration status for the current DB
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with the current timestamp
    fix                  Apply sequential ordering to migrations`
)
