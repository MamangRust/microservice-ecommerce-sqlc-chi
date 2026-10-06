-- sqlc-only schema overlay (NOT applied by goose).
--
-- The order service runs a retention query (DeleteOldProductStockAdjustments)
-- against the product_stock_adjustments table, which is owned by the product
-- service. In the per-service database model that table does not exist in the
-- order database, so the migration dir cannot contain it; this overlay exists
-- only so sqlc can type-check the query. It is never mounted as a migration.
CREATE TABLE "product_stock_adjustments" (
    "operation_id" VARCHAR(255) PRIMARY KEY,
    "product_id" INT NOT NULL,
    "delta" INT NOT NULL CHECK ("delta" <> 0),
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
