package postgres

import (
	"auction/app/item"
	"auction/domain"
	"context"
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
func (r *PgRepository) GetPoolStats() map[string]interface{} {
	stats := r.db.Stats()
	return map[string]interface{}{
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

func (r *PgRepository) Create(ctx context.Context, req *item.CreateItemRequest) (domain.Item, error) {
	var i domain.Item
	query := `
		INSERT INTO items (
			name, description, seller_id, currency_code,
			start_price, bid_increment, reserve_price,
			buyout_price, end_price, start_date, end_date,
			status
		) VALUES (
			:name, :description, :seller_id, :currency_code,
			:start_price, :bid_increment, :reserve_price,
			:buyout_price, :end_price, :start_date, :end_date,
			:status
		) RETURNING *`

	rows, err := r.db.NamedQueryContext(ctx, query, req)
	if err != nil {
		return i, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.StructScan(&i)
	}
	return i, err
}

func (r *PgRepository) GetItems(ctx context.Context, limit, offset int) ([]domain.Item, error) {
	items := make([]domain.Item, 0)
	query := `SELECT * FROM items ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	err := r.db.SelectContext(ctx, &items, query, limit, offset)

	if err != nil {
		return nil, err
	}

	return items, nil
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

func (r *PgRepository) GetItem(ctx context.Context, id string) (domain.Item, error) {
	var i domain.Item
	query := `SELECT * FROM items WHERE id = $1`

	err := r.db.GetContext(ctx, &i, query, id)

	if err != nil {
		return i, err
	}

	return i, nil
}

func (r *PgRepository) GetUserItem(ctx context.Context, id string, userId string) (domain.Item, error) {
	var i domain.Item
	query := `SELECT * FROM items WHERE id = $1 AND seller_id = $2`

	err := r.db.GetContext(ctx, &i, query, id, userId)

	if err != nil {
		return i, err
	}

	return i, nil
}

func (r *PgRepository) DeleteItem(ctx context.Context, id string, userId string) error {
	query := `DELETE FROM items WHERE id = $1 AND seller_id = $2`

	_, err := r.db.ExecContext(ctx, query, id, userId)

	return err
}

// sqlx ile NamedExecContext kullanarak
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
