-- name: GetTransactions :many
SELECT
    *,
    COUNT(*) OVER() AS total_count
FROM transactions
WHERE deleted_at IS NULL
  AND ($1::TEXT IS NULL OR payment_method ILIKE '%' || $1 || '%' OR payment_status ILIKE '%' || $1 || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;


-- GetTransactionsActive: Retrieves paginated list of active transactions (identical to GetTransactions)
-- Purpose: Maintains consistent API pattern with other active/trashed endpoints
-- Parameters:
--   $1: search_term - Optional filter text for payment method/status
--   $2: limit - Pagination limit
--   $3: offset - Pagination offset
-- Returns:
--   Active transaction records with total_count
-- Business Logic:
--   - Same functionality as GetTransactions
--   - Exists for consistency in API design patterns
-- Note: Could be consolidated with GetTransactions if duplicate functionality is undesired
-- name: GetTransactionsActive :many
SELECT
    *,
    COUNT(*) OVER() AS total_count
FROM transactions
WHERE deleted_at IS NULL
AND ($1::TEXT IS NULL OR payment_method ILIKE '%' || $1 || '%' OR payment_status ILIKE '%' || $1 || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- GetTransactionsTrashed: Retrieves paginated list of soft-deleted transactions
-- Purpose: View and manage deleted transactions for audit/recovery
-- Parameters:
--   $1: search_term - Optional text to filter trashed transactions
--   $2: limit - Maximum records per page
--   $3: offset - Records to skip
-- Returns:
--   Trashed transaction records with total_count
-- Business Logic:
--   - Only returns soft-deleted records (deleted_at IS NOT NULL)
--   - Maintains same search functionality as active transaction queries
--   - Preserves chronological sorting (newest first)
--   - Used in transaction recovery/audit interfaces
--   - Includes total_count for pagination in trash management UI
-- name: GetTransactionsTrashed :many
SELECT
    *,
    COUNT(*) OVER() AS total_count
FROM transactions
WHERE deleted_at IS NOT NULL
AND ($1::TEXT IS NULL OR payment_method ILIKE '%' || $1 || '%' OR payment_status ILIKE '%' || $1 || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;


-- GetTransactionByMerchant: Retrieves merchant-specific transactions with pagination
-- Purpose: List transactions filtered by merchant ID
-- Parameters:
--   $1: search_term - Optional text to filter transactions
--   $2: merchant_id - Optional merchant ID to filter by (NULL for all merchants)
--   $3: limit - Pagination limit
--   $4: offset - Pagination offset
-- Returns:
--   Transaction records with total_count
-- Business Logic:
--   - Combines merchant filtering with search functionality
--   - Maintains same sorting and pagination as other transaction queries
--   - Useful for merchant-specific transaction reporting
--   - NULL merchant_id parameter returns all merchants' transactions
-- name: GetTransactionByMerchant :many
SELECT
    *,
    COUNT(*) OVER() AS total_count
FROM transactions
WHERE deleted_at IS NULL
  AND ($1::TEXT IS NULL OR payment_method ILIKE '%' || $1 || '%' OR payment_status ILIKE '%' || $1 || '%')
  AND ($2::INT IS NULL OR merchant_id = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;


-- name: CreateTransaction :one
INSERT INTO transactions (
    order_id, merchant_id, payment_method, amount, payment_status
) VALUES ($1, $2, $3, $4, $5)
RETURNING transaction_id,
    order_id,
    merchant_id,
    payment_method,
    amount,
    payment_status,
    created_at,
    updated_at;


-- GetTransactionByOrderID: Retrieves transaction by order reference
-- Purpose: Lookup transaction associated with specific order
-- Parameters:
--   $1: order_id - The order ID to search by
-- Returns: Transaction record if found and active
-- Business Logic:
--   - Only returns non-deleted transactions
--   - Used for order payment verification
--   - Helps prevent duplicate payments
-- name: GetTransactionByOrderID :one
SELECT 
    transaction_id,
    order_id,
    merchant_id,
    payment_method,
    amount,
    payment_status,
    created_at,
    updated_at
FROM transactions
WHERE order_id = $1
  AND deleted_at IS NULL;

-- GetTransactionByID: Retrieves transaction by transaction ID
-- Purpose: Fetch specific transaction details
-- Parameters:
--   $1: transaction_id - The unique transaction ID
-- Returns: Full transaction record if active
-- Business Logic:
--   - Excludes deleted transactions
--   - Used for transaction details/receipts
--   - Primary lookup for transaction management
-- name: GetTransactionByID :one
SELECT transaction_id,
    order_id,
    merchant_id,
    payment_method,
    amount,
    payment_status,
    created_at,
    updated_at
FROM transactions
WHERE transaction_id = $1
  AND deleted_at IS NULL;


-- UpdateTransaction: Modifies transaction details
-- Purpose: Update transaction information
-- Parameters:
--   $1: transaction_id - ID of transaction to update
--   $2: merchant_id - Updated merchant reference
--   $3: payment_method - Updated payment method
--   $4: amount - Updated transaction amount
--   $5: change_amount - Updated change amount
--   $6: payment_status - Updated payment status
--   $7: order_id - Updated order reference
-- Returns: Updated transaction record
-- Business Logic:
--   - Auto-updates updated_at timestamp
--   - Only modifies active transactions
--   - Validates all payment fields
--   - Used for payment corrections/updates
-- name: UpdateTransaction :one
UPDATE transactions
SET merchant_id = $2,
    payment_method = $3,
    amount = $4,
    payment_status = $5,
    order_id = $6,
    updated_at = CURRENT_TIMESTAMP
WHERE transaction_id = $1
  AND deleted_at IS NULL
RETURNING transaction_id,
    order_id,
    merchant_id,
    payment_method,
    amount,
    payment_status,
    created_at,
    updated_at;


-- TrashTransaction: Soft-deletes a transaction
-- Purpose: Void/cancel a transaction without permanent deletion
-- Parameters:
--   $1: transaction_id - ID of transaction to cancel
-- Returns: The soft-deleted transaction record
-- Business Logic:
--   - Sets deleted_at to current timestamp
--   - Preserves transaction for reporting
--   - Only processes active transactions
--   - Can be restored if needed
-- name: TrashTransaction :one
UPDATE transactions
SET
    deleted_at = current_timestamp
WHERE
    transaction_id = $1
    AND deleted_at IS NULL
    RETURNING transaction_id,
    order_id,
    merchant_id,
    payment_method,
    amount,
    payment_status,
    created_at,
    updated_at,
    deleted_at;


-- RestoreTransaction: Recovers a soft-deleted transaction
-- Purpose: Reactivate a cancelled transaction
-- Parameters:
--   $1: transaction_id - ID of transaction to restore
-- Returns: The restored transaction record
-- Business Logic:
--   - Nullifies deleted_at field
--   - Only works on previously cancelled transactions
--   - Maintains all original transaction data
-- name: RestoreTransaction :one
UPDATE transactions
SET
    deleted_at = NULL
WHERE
    transaction_id = $1
    AND deleted_at IS NOT NULL
  RETURNING transaction_id,
    order_id,
    merchant_id,
    payment_method,
    amount,
    payment_status,
    created_at,
    updated_at,
    deleted_at;


-- DeleteTransactionPermanently: Hard-deletes a transaction
-- Purpose: Completely remove transaction from database
-- Parameters:
--   $1: transaction_id - ID of transaction to delete
-- Business Logic:
--   - Permanent deletion of already cancelled transactions
--   - No return value (exec-only operation)
--   - Irreversible action - use with caution
--   - Should be restricted to admin users
-- name: DeleteTransactionPermanently :exec
DELETE FROM transactions WHERE transaction_id = $1 AND deleted_at IS NOT NULL;


-- RestoreAllTransactions: Mass restoration of cancelled transactions
-- Purpose: Recover all trashed transactions at once
-- Business Logic:
--   - Reactivates all soft-deleted transactions
--   - No parameters needed (bulk operation)
--   - Typically used during system recovery
-- name: RestoreAllTransactions :exec
UPDATE transactions
SET
    deleted_at = NULL
WHERE
    deleted_at IS NOT NULL;


-- DeleteAllPermanentTransactions: Purges all cancelled transactions
-- Purpose: Clean up all soft-deleted transaction records
-- Business Logic:
--   - Irreversible bulk deletion operation
--   - Only affects already cancelled transactions
--   - Typically used during database maintenance
--   - Should be restricted to admin users
-- name: DeleteAllPermanentTransactions :exec
-- Purges all soft-deleted transactions. The orders cross-DB subquery was
-- removed: orders live in the order service DB (1 DB per service), so cleanup
-- of transactions whose parent order is trashed happens per-order via
-- DeleteTransactionByOrderPermanent instead.
DELETE FROM transactions
WHERE
    deleted_at IS NOT NULL;

-- DeleteTransactionByOrderPermanent: Permanently deletes all transactions for a specific order
-- Purpose: Cascading cleanup during order permanent deletion
-- Parameters:
--   $1: order_id
-- name: DeleteTransactionByOrderPermanent :exec
DELETE FROM transactions
WHERE
    order_id = $1;



