-- name: GetOrders :many
SELECT
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM orders
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR order_id::TEXT ILIKE '%' || $1 || '%'
        OR total_price::TEXT ILIKE '%' || $1 || '%'
    )
ORDER BY created_at DESC
LIMIT $2
OFFSET
    $3;

-- GetOrdersActive: Retrieves paginated list of active orders (identical to GetOrders)
-- Purpose: Maintains consistent API pattern with other active/trashed endpoints
-- Parameters:
--   $1: search_term - Optional filter text for order ID or price
--   $2: limit - Pagination limit
--   $3: offset - Pagination offset
-- Returns:
--   Active order records with total_count
-- Business Logic:
--   - Same functionality as GetOrders
--   - Exists for consistency in API design patterns
-- Note: Could be consolidated with GetOrders if duplicate functionality is undesired
-- name: GetOrdersActive :many
SELECT
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM orders
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR order_id::TEXT ILIKE '%' || $1 || '%'
        OR total_price::TEXT ILIKE '%' || $1 || '%'
    )
ORDER BY created_at DESC
LIMIT $2
OFFSET
    $3;

-- GetOrdersTrashed: Retrieves paginated list of soft-deleted orders
-- Purpose: View and manage deleted orders for potential restoration
-- Parameters:
--   $1: search_term - Optional text to filter trashed orders
--   $2: limit - Maximum records per page
--   $3: offset - Records to skip
-- Returns:
--   Trashed order records with total_count
-- Business Logic:
--   - Only returns soft-deleted records (deleted_at IS NOT NULL)
--   - Maintains same search functionality as active order queries
--   - Preserves chronological sorting (newest first)
--   - Used in order recovery/audit interfaces
--   - Includes total_count for pagination in trash management UI
-- name: GetOrdersTrashed :many
SELECT
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM orders
WHERE
    deleted_at IS NOT NULL
    AND (
        $1::TEXT IS NULL
        OR order_id::TEXT ILIKE '%' || $1 || '%'
        OR total_price::TEXT ILIKE '%' || $1 || '%'
    )
ORDER BY created_at DESC
LIMIT $2
OFFSET
    $3;

-- GetOrdersByMerchant: Retrieves merchant-specific orders with pagination
-- Purpose: List orders filtered by merchant ID
-- Parameters:
--   $1: search_term - Optional text to filter orders
--   $2: limit - Pagination limit
--   $3: offset - Pagination offset
--   $4: merchant_id - Optional merchant UUID to filter by (NULL for all merchants)
-- Returns:
--   Order records with total_count
-- Business Logic:
--   - Combines merchant filtering with search functionality
--   - Maintains same sorting and pagination as other order queries
--   - Useful for merchant-specific order dashboards
--   - NULL merchant_id parameter returns all merchants' orders
-- name: GetOrdersByMerchant :many
SELECT
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM orders
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR order_id::TEXT ILIKE '%' || $1 || '%'
        OR total_price::TEXT ILIKE '%' || $1 || '%'
    )
    AND (
        $4::INT IS NULL
        OR merchant_id = $4
    )
ORDER BY created_at DESC
LIMIT $2
OFFSET
    $3;

-- name: CreateOrder :one
INSERT INTO
    orders (
        merchant_id,
        user_id,
        total_price
    )
VALUES ($1, $2, $3)
RETURNING
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at;

-- GetOrderByID: Retrieves an active order by ID
-- Purpose: Fetch order details for display/processing
-- Parameters:
--   $1: order_id - UUID of the order to retrieve
-- Returns: Full order record if found and active
-- Business Logic:
--   - Excludes soft-deleted orders
--   - Used for order viewing, receipts, and processing
--   - Typically joined with order_items in application
-- name: GetOrderByID :one
SELECT
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at
FROM orders
WHERE
    order_id = $1
    AND deleted_at IS NULL;

-- GetTrashedOrder: Retrieves one soft-deleted order for guarded permanent deletion.
-- name: GetTrashedOrder :one
SELECT
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at,
    deleted_at
FROM orders
WHERE order_id = $1 AND deleted_at IS NOT NULL;

-- UpdateOrder: Modifies order information
-- Purpose: Update order details (primarily total price)
-- Parameters:
--   $1: order_id - UUID of order to update
--   $2: total_price - New total amount
-- Returns: Updated order record
-- Business Logic:
--   - Auto-updates updated_at timestamp
--   - Only modifies active (non-deleted) orders
--   - Used when order items change
--   - Should trigger recalculation of total_price
-- name: UpdateOrder :one
UPDATE orders
SET
    total_price = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE
    order_id = $1
    AND deleted_at IS NULL
RETURNING
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at;

-- TrashedOrder: Soft-deletes an order
-- Purpose: Cancel/void an order without permanent deletion
-- Parameters:
--   $1: order_id - UUID of order to cancel
-- Returns: The soft-deleted order record
-- Business Logic:
--   - Sets deleted_at to current timestamp
--   - Preserves order data for reporting
--   - Only processes active orders
--   - Can be restored via RestoreOrder
-- name: TrashedOrder :one
UPDATE orders
SET
    deleted_at = current_timestamp
WHERE
    order_id = $1
    AND deleted_at IS NULL
RETURNING
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at,
    deleted_at;

-- RestoreOrder: Recovers a soft-deleted order
-- Purpose: Reactivate a cancelled order
-- Parameters:
--   $1: order_id - UUID of order to restore
-- Returns: The restored order record
-- Business Logic:
--   - Nullifies deleted_at field
--   - Only works on previously cancelled orders
--   - Maintains all original order data
-- name: RestoreOrder :one
UPDATE orders
SET
    deleted_at = NULL
WHERE
    order_id = $1
    AND deleted_at IS NOT NULL
RETURNING
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at,
    deleted_at;

-- GetTrashedOrders: Retrieves all soft-deleted orders for per-order restoration.
-- name: GetTrashedOrders :many
SELECT
    order_id,
    user_id,
    merchant_id,
    total_price,
    created_at,
    updated_at,
    deleted_at
FROM orders
WHERE deleted_at IS NOT NULL
ORDER BY order_id;

-- DeleteOrderPermanently: Hard-deletes an order
-- Purpose: Completely remove order from database
-- Parameters:
--   $1: order_id - UUID of order to delete
-- Business Logic:
--   - Permanent deletion of already cancelled orders
--   - No return value (exec-only operation)
--   - Irreversible action - use with caution
--   - Should trigger deletion of related order_items
-- name: DeleteOrderPermanently :exec
DELETE FROM orders WHERE order_id = $1 AND deleted_at IS NOT NULL;

-- DeleteOrderPermanentlyWithChildren: Atomically purges a trashed order and all
-- of its child rows (stock reservations, order items, transactions, shipping
-- addresses) in a single statement so a mid-way failure cannot orphan children.
-- The trashed guard lives in a leading CTE that gates EVERY child delete, so a
-- call on a non-trashed order deletes nothing and returns no row (the caller
-- surfaces ErrOrderNotFound) instead of wiping children of an active order.
-- name: DeleteOrderPermanentlyWithChildren :one
-- Purges a trashed order and its local child rows (stock reservations) in a
-- single statement. Order items, transactions, and shipping addresses live in
-- their own per-service databases (cross-DB FK split), so those children are
-- removed by the order service via gRPC calls to the owning services instead.
-- The trashed guard lives in a leading CTE that gates every child delete, so a
-- call on a non-trashed order deletes nothing and returns no row (the caller
-- surfaces ErrOrderNotFound) instead of wiping children of an active order.
WITH
    trashed AS (
        SELECT order_id FROM orders WHERE order_id = $1 AND deleted_at IS NOT NULL
    ),
    deleted_reservations AS (
        DELETE FROM order_stock_reservations WHERE order_id IN (SELECT order_id FROM trashed)
    )
DELETE FROM orders o
WHERE o.order_id = $1 AND o.deleted_at IS NOT NULL
RETURNING o.order_id;

-- RestoreAllOrders: Mass restoration of cancelled orders
-- Purpose: Recover all trashed orders at once
-- Business Logic:
--   - Reactivates all soft-deleted orders
--   - No parameters needed (bulk operation)
--   - Typically used during system recovery
-- name: RestoreAllOrders :exec
UPDATE orders
SET
    deleted_at = NULL
WHERE
    deleted_at IS NOT NULL;

-- DeleteAllPermanentOrders: Purges all cancelled orders
-- Purpose: Clean up all soft-deleted order records
-- Business Logic:
--   - Irreversible bulk deletion operation
--   - Only affects already cancelled orders
--   - Typically used during database maintenance
--   - Should be restricted to admin users
-- name: DeleteAllPermanentOrders :exec
DELETE FROM orders WHERE deleted_at IS NOT NULL;
