package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/lukivan8/look-at-finance-api/internal/modules/stores/models"
	"github.com/lukivan8/look-at-finance-api/internal/shared/database"
)

var ErrStoreNotFound = errors.New("store not found")

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByUser(ctx context.Context, userID string, search string) ([]models.Store, error) {
	query := `
		SELECT 
			s.id,
			s.name,
			COALESCE(s.address, '') as address,
			COUNT(r.id) as receipt_count,
			COALESCE(SUM(r.total_amount), 0) as total_spent,
			COALESCE(MAX(r.purchase_date), s.created_at) as last_visit
		FROM stores s
		LEFT JOIN receipts r ON r.store_id = s.id AND r.user_id = $1
		WHERE r.user_id = $1
	`

	args := []interface{}{userID}

	if search != "" {
		query += ` AND s.name ILIKE $2`
		args = append(args, "%"+search+"%")
	}

	query += `
		GROUP BY s.id
		ORDER BY last_visit DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Store
	for rows.Next() {
		var store models.Store
		err := rows.Scan(
			&store.ID,
			&store.Name,
			&store.Address,
			&store.ReceiptCount,
			&store.TotalSpent,
			&store.LastVisit,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, store)
	}

	return result, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*models.Store, error) {
	query := `
		SELECT id, name, COALESCE(address, '') as address, created_at
		FROM stores
		WHERE id = $1
	`
	store := &models.Store{}
	err := r.db.Pool.QueryRow(ctx, query, id).
		Scan(&store.ID, &store.Name, &store.Address, &store.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStoreNotFound
		}
		return nil, err
	}
	return store, nil
}

func (r *Repository) GetOrCreateByName(ctx context.Context, name string, address string) (*models.Store, error) {
	// Try to find existing store
	query := `
		SELECT id, name, COALESCE(address, '') as address, created_at
		FROM stores
		WHERE name = $1
		LIMIT 1
	`
	store := &models.Store{}
	err := r.db.Pool.QueryRow(ctx, query, name).
		Scan(&store.ID, &store.Name, &store.Address, &store.CreatedAt)
	if err == nil {
		return store, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Create new store
	insertQuery := `
		INSERT INTO stores (name, address)
		VALUES ($1, $2)
		RETURNING id, name, address, created_at
	`
	err = r.db.Pool.QueryRow(ctx, insertQuery, name, address).
		Scan(&store.ID, &store.Name, &store.Address, &store.CreatedAt)
	if err != nil {
		return nil, err
	}

	return store, nil
}
