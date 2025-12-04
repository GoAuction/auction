package postgres

import (
	"auction/app"
	"auction/domain"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

type PgRepository struct {
	db *sqlx.DB
}

func NewPgRepository(host, database, user, password, port string) *PgRepository {
	db := sqlx.MustConnect("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database,
	))

	// Connection pool configuration
	// With 3 replicas × 15 conns = 45 total connections (safer for default PG max_connections=100)
	db.SetMaxOpenConns(15)                 // Max concurrent DB connections per instance
	db.SetMaxIdleConns(8)                  // Keep 8 idle connections in pool
	db.SetConnMaxLifetime(5 * time.Minute) // Recycle connections every 5 min
	db.SetConnMaxIdleTime(2 * time.Minute) // Close idle connections after 2 min

	return &PgRepository{db: db}
}

func (r *PgRepository) Close() error {
	return r.db.Close()
}

// GetPoolStats returns current connection pool statistics
func (r *PgRepository) GetPoolStats() map[string]any {
	stats := r.db.Stats()
	return map[string]any{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,                   // How many times waited for connection
		"wait_duration_ms":     stats.WaitDuration.Milliseconds(), // Total time spent waiting
		"max_idle_closed":      stats.MaxIdleClosed,               // Connections closed due to idle
		"max_lifetime_closed":  stats.MaxLifetimeClosed,           // Connections closed due to max lifetime
	}
}

func (r *PgRepository) Create(ctx context.Context, req *app.CreateItemRequest) (domain.Item, error) {
	// Start transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.Item{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be no-op if transaction is committed

	// Insert item using positional parameters
	var itemID string
	query := `
		INSERT INTO items (
			name, description, seller_id, currency_code,
			start_price, bid_increment, reserve_price,
			buyout_price, end_price, start_date, end_date,
			status
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10, $11,
			$12
		) RETURNING id`

	err = tx.QueryRowContext(ctx, query,
		req.Name,
		req.Description,
		req.SellerID,
		req.CurrencyCode,
		req.StartPrice,
		req.BidIncrement,
		req.ReservePrice,
		req.BuyoutPrice,
		req.EndPrice,
		req.StartDate,
		req.EndDate,
		req.Status,
	).Scan(&itemID)

	if err != nil {
		return domain.Item{}, fmt.Errorf("failed to insert item: %w", err)
	}

	// Insert item categories if provided
	if len(req.CategoryIDs) > 0 {
		categoryQuery := `INSERT INTO item_categories (item_id, category_id) VALUES ($1, $2)`
		for _, categoryID := range req.CategoryIDs {
			if _, err := tx.ExecContext(ctx, categoryQuery, itemID, categoryID); err != nil {
				return domain.Item{}, fmt.Errorf("failed to insert item category: %w", err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return domain.Item{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch and return the complete item with categories
	return r.GetItem(ctx, itemID)
}

func (r *PgRepository) GetItems(ctx context.Context, limit, offset int) ([]domain.Item, error) {
	// Temporary struct to hold the query result with JSON categories
	type itemWithCategories struct {
		domain.Item
		CategoriesJSON sql.NullString `db:"categories"`
	}

	query := `
		SELECT
			items.*,
			COALESCE(
				json_agg(
					json_build_object(
						'id', categories.id,
						'name', categories.name,
						'description', categories.description,
						'parent_id', categories.parent_id,
						'status', categories.status,
						'created_at', categories.created_at,
						'updated_at', categories.updated_at
					)
				) FILTER (WHERE categories.id IS NOT NULL),
				'[]'
			) as categories
		FROM items
		LEFT JOIN item_categories ON items.id = item_categories.item_id
		LEFT JOIN categories ON item_categories.category_id = categories.id
		GROUP BY items.id
		ORDER BY items.created_at DESC
		LIMIT $1 OFFSET $2`

	var tempItems []itemWithCategories
	err := r.db.SelectContext(ctx, &tempItems, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert to final Item slice with unmarshaled categories
	items := make([]domain.Item, len(tempItems))
	for i, temp := range tempItems {
		items[i] = temp.Item

		// Unmarshal categories JSON if present
		if temp.CategoriesJSON.Valid && temp.CategoriesJSON.String != "[]" {
			if err := json.Unmarshal([]byte(temp.CategoriesJSON.String), &items[i].Categories); err != nil {
				return nil, fmt.Errorf("failed to unmarshal categories: %w", err)
			}
		} else {
			items[i].Categories = []domain.Category{}
		}
	}

	return items, nil
}

func (r *PgRepository) GetCategories(ctx context.Context, limit, offset int) ([]domain.Category, error) {
	categories := make([]domain.Category, 0)
	query := `SELECT * FROM categories ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	err := r.db.SelectContext(ctx, &categories, query, limit, offset)

	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *PgRepository) CountItems(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM items`

	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PgRepository) CountCategories(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM categories`

	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PgRepository) GetItem(ctx context.Context, id string) (domain.Item, error) {
	// Temporary struct to hold the query result with JSON categories
	type itemWithCategories struct {
		domain.Item
		CategoriesJSON sql.NullString `db:"categories"`
	}

	query := `
		SELECT
			items.*,
			COALESCE(
				json_agg(
					json_build_object(
						'id', categories.id,
						'name', categories.name,
						'description', categories.description,
						'parent_id', categories.parent_id,
						'status', categories.status,
						'created_at', categories.created_at,
						'updated_at', categories.updated_at
					)
				) FILTER (WHERE categories.id IS NOT NULL),
				'[]'
			) as categories
		FROM items
		LEFT JOIN item_categories ON items.id = item_categories.item_id
		LEFT JOIN categories ON item_categories.category_id = categories.id
		WHERE items.id = $1
		GROUP BY items.id`

	var temp itemWithCategories
	err := r.db.GetContext(ctx, &temp, query, id)
	if err != nil {
		return domain.Item{}, err
	}

	item := temp.Item

	// Unmarshal categories JSON if present
	if temp.CategoriesJSON.Valid && temp.CategoriesJSON.String != "[]" {
		if err := json.Unmarshal([]byte(temp.CategoriesJSON.String), &item.Categories); err != nil {
			return domain.Item{}, fmt.Errorf("failed to unmarshal categories: %w", err)
		}
	} else {
		item.Categories = []domain.Category{}
	}

	return item, nil
}

func (r *PgRepository) GetUserItem(ctx context.Context, id string, userId string) (domain.Item, error) {
	// Temporary struct to hold the query result with JSON categories
	type itemWithCategories struct {
		domain.Item
		CategoriesJSON sql.NullString `db:"categories"`
	}

	query := `
		SELECT
			items.*,
			COALESCE(
				json_agg(
					json_build_object(
						'id', categories.id,
						'name', categories.name,
						'description', categories.description,
						'parent_id', categories.parent_id,
						'status', categories.status,
						'created_at', categories.created_at,
						'updated_at', categories.updated_at
					)
				) FILTER (WHERE categories.id IS NOT NULL),
				'[]'
			) as categories
		FROM items
		LEFT JOIN item_categories ON items.id = item_categories.item_id
		LEFT JOIN categories ON item_categories.category_id = categories.id
		WHERE items.id = $1 AND items.seller_id = $2
		GROUP BY items.id`

	var temp itemWithCategories
	err := r.db.GetContext(ctx, &temp, query, id, userId)
	if err != nil {
		return domain.Item{}, err
	}

	item := temp.Item

	// Unmarshal categories JSON if present
	if temp.CategoriesJSON.Valid && temp.CategoriesJSON.String != "[]" {
		if err := json.Unmarshal([]byte(temp.CategoriesJSON.String), &item.Categories); err != nil {
			return domain.Item{}, fmt.Errorf("failed to unmarshal categories: %w", err)
		}
	} else {
		item.Categories = []domain.Category{}
	}

	return item, nil
}

func (r *PgRepository) DeleteItem(ctx context.Context, id string, userId string) error {
	query := `DELETE FROM items WHERE id = $1 AND seller_id = $2`

	_, err := r.db.ExecContext(ctx, query, id, userId)

	return err
}

func (r *PgRepository) UpdateUserItem(ctx context.Context, item domain.Item, userId string) error {
	query := `
        UPDATE items SET
            name = :name,
            description = :description,
            seller_id = :seller_id,
            currency_code = :currency_code,
            start_price = :start_price,
            bid_increment = :bid_increment,
            reserve_price = :reserve_price,
            buyout_price = :buyout_price,
            end_price = :end_price,
            start_date = :start_date,
            end_date = :end_date,
            status = :status
        WHERE id = :id AND seller_id = :seller_id_filter
    `

	// named param map: item alanları + seller_id_filter (WHERE için)
	params := map[string]interface{}{
		"id":               item.ID,
		"name":             item.Name,
		"description":      item.Description,
		"seller_id":        item.SellerID,
		"currency_code":    item.CurrencyCode,
		"start_price":      item.StartPrice,
		"bid_increment":    item.BidIncrement,
		"reserve_price":    item.ReservePrice,
		"buyout_price":     item.BuyoutPrice,
		"end_price":        item.EndPrice,
		"start_date":       item.StartDate,
		"end_date":         item.EndDate,
		"status":           item.Status,
		"seller_id_filter": userId,
	}

	_, err := r.db.NamedExecContext(ctx, query, params)
	return err
}

func (r *PgRepository) Update(ctx context.Context, item domain.Item) error {
	query := `
        UPDATE items SET
            name = :name,
            description = :description,
            seller_id = :seller_id,
            currency_code = :currency_code,
            current_price = :current_price,
            start_price = :start_price,
            bid_increment = :bid_increment,
            reserve_price = :reserve_price,
            buyout_price = :buyout_price,
            end_price = :end_price,
            start_date = :start_date,
            end_date = :end_date,
            status = :status
        WHERE id = :id
    `

	params := map[string]interface{}{
		"id":            item.ID,
		"name":          item.Name,
		"description":   item.Description,
		"seller_id":     item.SellerID,
		"currency_code": item.CurrencyCode,
		"current_price": item.CurrentPrice,
		"start_price":   item.StartPrice,
		"bid_increment": item.BidIncrement,
		"reserve_price": item.ReservePrice,
		"buyout_price":  item.BuyoutPrice,
		"end_price":     item.EndPrice,
		"start_date":    item.StartDate,
		"end_date":      item.EndDate,
		"status":        item.Status,
	}

	_, err := r.db.NamedExecContext(ctx, query, params)
	return err
}

func (r *PgRepository) GetCategoryByID(ctx context.Context, id string) (domain.Category, error) {
	var category domain.Category

	err := r.db.GetContext(ctx, &category, "SELECT * FROM categories WHERE id = $1", id)
	if err != nil {
		return category, err
	}

	return category, nil
}

func (r *PgRepository) GetCategoriesByItemID(ctx context.Context, itemId string) ([]domain.Category, error) {
	categories := make([]domain.Category, 0)

	err := r.db.SelectContext(ctx, &categories, "SELECT * FROM categories WHERE id IN (SELECT category_id FROM item_categories WHERE item_id = $1)", itemId)
	if err != nil {
		return categories, err
	}

	return categories, nil
}

func (r *PgRepository) GetItemCommentsByItemID(ctx context.Context, itemID string, page, pageSize int) ([]domain.ItemComment, error) {
	comments := make([]domain.ItemComment, 0)

	limit := pageSize
	offset := (page - 1) * pageSize

	err := r.db.SelectContext(ctx, &comments, "SELECT * FROM item_comments WHERE item_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3", itemID, limit, offset)
	if err != nil {
		return comments, err
	}

	return comments, nil
}

func (r *PgRepository) CountItemComments(ctx context.Context, itemID string) (int, error) {
	var count int

	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM item_comments WHERE item_id = $1", itemID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PgRepository) CreateComment(ctx context.Context, itemID string, content string, userID string, parentID *string) (domain.ItemComment, error) {
	query := `
		INSERT INTO item_comments (item_id, content, user_id, parent_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, item_id, content, user_id, parent_id, created_at, updated_at
	`

	var comment domain.ItemComment
	err := r.db.GetContext(ctx, &comment, query, itemID, content, userID, parentID)
	if err != nil {
		return domain.ItemComment{}, err
	}

	return comment, nil
}

func (r *PgRepository) DeleteComment(ctx context.Context, id string) error {
	query := `
		DELETE FROM item_comments WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}

func (r *PgRepository) GetCommentByID(ctx context.Context, id string) (domain.ItemComment, error) {
	var comment domain.ItemComment

	err := r.db.GetContext(ctx, &comment, "SELECT * FROM item_comments WHERE id = $1", id)
	if err != nil {
		return domain.ItemComment{}, err
	}

	return comment, nil
}

func (r *PgRepository) GetItemImages(ctx context.Context, itemID string, page, pageSize int) ([]domain.ItemImage, error) {
	images := make([]domain.ItemImage, 0)

	offset := (page - 1) * pageSize

	err := r.db.SelectContext(ctx, &images, "SELECT * FROM item_images WHERE item_id = $1 ORDER BY display_order ASC LIMIT $2 OFFSET $3", itemID, pageSize, offset)
	if err != nil {
		return images, err
	}

	return images, nil
}

func (r *PgRepository) CountItemImages(ctx context.Context, itemID string) (int, error) {
	var count int

	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM item_images WHERE item_id = $1", itemID)
	if err != nil {
		return 0, err
	}

	return count, nil
}
