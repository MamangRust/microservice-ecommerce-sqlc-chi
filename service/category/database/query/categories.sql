-- name: GetCategories :many
SELECT
    category_id,
    name,
    description,
    slug_category,
    image_category,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM categories
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR name ILIKE '%' || $1 || '%'
        OR slug_category ILIKE '%' || $1 || '%'
    )
ORDER BY created_at DESC
LIMIT $2
OFFSET
    $3;

-- GetCategoriesActive: Retrieves paginated list of active categories with search capability
-- Purpose: List all active product categories for management UI
-- Parameters:
--   $1: search_term - Optional text to filter categories by name or slug (NULL for no filter)
--   $2: limit - Maximum number of records to return
--   $3: offset - Number of records to skip for pagination
-- Returns:
--   All category fields plus total_count of matching records
-- Business Logic:
--   - Excludes soft-deleted categories (deleted_at IS NULL)
--   - Supports partial text matching on name and slug_category fields (case-insensitive)
--   - Returns newest categories first (created_at DESC)
--   - Provides total_count for pagination calculations
-- name: GetCategoriesActive :many
SELECT
    category_id,
    name,
    description,
    slug_category,
    image_category,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM categories
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR name ILIKE '%' || $1 || '%'
        OR slug_category ILIKE '%' || $1 || '%'
    )
ORDER BY created_at DESC
LIMIT $2
OFFSET
    $3;

-- GetCategoriesTrashed: Retrieves paginated list of soft-deleted categories
-- Purpose: View/manage deleted categories for potential restoration
-- Parameters:
--   $1: search_term - Optional filter text (NULL for all trashed categories)
--   $2: limit - Pagination limit
--   $3: offset - Pagination offset
-- Returns:
--   Trashed category records with total_count
-- Business Logic:
--   - Only returns soft-deleted records (deleted_at IS NOT NULL)
--   - Same search functionality as active categories
--   - Maintains consistent sorting with active records
--   - Used in trash management/recovery interfaces
-- name: GetCategoriesTrashed :many
SELECT
    category_id,
    name,
    description,
    slug_category,
    image_category,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM categories
WHERE
    deleted_at IS NOT NULL
    AND (
        $1::TEXT IS NULL
        OR name ILIKE '%' || $1 || '%'
        OR slug_category ILIKE '%' || $1 || '%'
    )
ORDER BY created_at DESC
LIMIT $2
OFFSET
    $3;

-- name: CreateCategory :one
INSERT INTO
    categories (
        name,
        description,
        slug_category,
        image_category
    )
VALUES ($1, $2, $3, $4)
RETURNING
    category_id,
    name,
    description,
    slug_category,
    image_category,
    created_at,
    updated_at;

-- GetCategoryByID: Fetches a single category by its ID
-- Purpose: Retrieve details of an active (non-deleted) category
-- Parameters:
--   $1: Category ID to search for
-- Returns:
--   Full category record if found and not deleted
-- Business Logic:
--   - Excludes soft-deleted categories
-- name: GetCategoryByID :one
SELECT
    category_id,
    name,
    description,
    slug_category,
    image_category,
    created_at,
    updated_at
FROM categories
WHERE
    category_id = $1
    AND deleted_at IS NULL;

-- GetCategoryByIDTrashed: Fetches a single category by its ID
-- Purpose: Retrieve details of an active (non-deleted) category
-- Parameters:
--   $1: Category ID to search for
-- Returns:
--   Full category record if found and not deleted
-- Business Logic:
--   - Excludes soft-deleted categories
-- name: GetCategoryByIDTrashed :one
SELECT *
FROM categories
WHERE
    category_id = $1
    AND deleted_at IS NOT NULL;

-- UpdateCategory: Updates category details
-- Purpose: Modify existing category data while maintaining soft delete integrity
-- Parameters:
--   $1: Category ID
--   $2: Updated name
--   $3: Updated description
--   $4: Updated slug
-- Returns:
--   Updated category record
-- Business Logic:
--   - Automatically updates the updated_at field
--   - Skips if category has been soft-deleted
-- name: UpdateCategory :one
UPDATE categories
SET
    name = $2,
    description = $3,
    slug_category = $4,
    image_category = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE
    category_id = $1
    AND deleted_at IS NULL
RETURNING
    category_id,
    name,
    description,
    slug_category,
    image_category,
    created_at,
    updated_at;

-- TrashCategory: Soft-deletes a category
-- Purpose: Moves category to trash without permanent deletion
-- Parameters:
--   $1: Category ID
-- Returns:
--   The soft-deleted category record
-- Business Logic:
--   - Updates deleted_at with current timestamp
--   - Prevents repeat trashing of already-deleted records
-- name: TrashCategory :one
UPDATE categories
SET
    deleted_at = current_timestamp
WHERE
    category_id = $1
    AND deleted_at IS NULL
RETURNING
    category_id,
    name,
    description,
    slug_category,
    image_category,
    created_at,
    updated_at,
    deleted_at;

-- RestoreCategory: Recovers a previously trashed category
-- Purpose: Restores a soft-deleted category for reuse
-- Parameters:
--   $1: Category ID
-- Returns:
--   Restored category record
-- Business Logic:
--   - Only applies to categories currently marked as deleted
-- name: RestoreCategory :one
UPDATE categories
SET
    deleted_at = NULL
WHERE
    category_id = $1
    AND deleted_at IS NOT NULL
RETURNING
    category_id,
    name,
    description,
    slug_category,
    image_category,
    created_at,
    updated_at, deleted_at;

-- DeleteCategoryPermanently: Removes a soft-deleted category permanently
-- Purpose: Final cleanup of trashed categories
-- Parameters:
--   $1: Category ID
-- Returns:
--   Nothing (command only)
-- Business Logic:
--   - Ensures category is deleted only if it has been soft-deleted
-- name: DeleteCategoryPermanently :exec
DELETE FROM categories
WHERE
    category_id = $1
    AND deleted_at IS NOT NULL;

-- RestoreAllCategories: Recovers all trashed categories
-- Purpose: Bulk restore of all soft-deleted category records
-- Parameters: None
-- Returns: None
-- Business Logic:
--   - Resets deleted_at for all soft-deleted records
-- name: RestoreAllCategories :exec
UPDATE categories
SET
    deleted_at = NULL
WHERE
    deleted_at IS NOT NULL;

-- DeleteAllPermanentCategories: Permanently deletes all trashed categories
-- Purpose: Bulk purge of all soft-deleted category records
-- Parameters: None
-- Returns: None
-- Business Logic:
--   - Only affects records marked as deleted
-- name: DeleteAllPermanentCategories :exec
DELETE FROM categories WHERE deleted_at IS NOT NULL;
