package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/lukivan8/look-at-finance-api/internal/modules/products/models"
	"github.com/lukivan8/look-at-finance-api/internal/shared/database"
)

var ErrProductNotFound = errors.New("product not found")

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, userID string, req models.ListProductsRequest) ([]models.ProductListItem, int, error) {
	// Build the query for products the user has purchased
	countQuery := `
		SELECT COUNT(DISTINCT p.id)
		FROM products p
		INNER JOIN receipt_items ri ON ri.product_id = p.id
		INNER JOIN receipts r ON r.id = ri.receipt_id AND r.user_id = $1
		WHERE 1=1
	`

	dataQuery := `
		WITH product_stats AS (
			SELECT 
				p.id,
				p.name,
				COALESCE(p.category, '') as category,
				MAX(pp.price) FILTER (WHERE pp.recorded_at = (
					SELECT MAX(pp2.recorded_at) FROM product_prices pp2 WHERE pp2.product_id = p.id
				)) as last_price,
				(SELECT s.name FROM stores s 
				 INNER JOIN product_prices pp3 ON pp3.store_id = s.id 
				 WHERE pp3.product_id = p.id 
				 ORDER BY pp3.recorded_at DESC LIMIT 1) as last_store,
				MAX(pp.recorded_at) as last_seen,
				MIN(pp.price) as min_price,
				MAX(pp.price) as max_price
			FROM products p
			INNER JOIN receipt_items ri ON ri.product_id = p.id
			INNER JOIN receipts r ON r.id = ri.receipt_id AND r.user_id = $1
			LEFT JOIN product_prices pp ON pp.product_id = p.id
			WHERE 1=1
	`

	args := []interface{}{userID}
	argCount := 1

	if req.Search != "" {
		argCount++
		countQuery += fmt.Sprintf(" AND p.name ILIKE $%d", argCount)
		dataQuery += fmt.Sprintf(" AND p.name ILIKE $%d", argCount)
		args = append(args, "%"+req.Search+"%")
	}

	if req.Category != "" {
		argCount++
		countQuery += fmt.Sprintf(" AND p.category = $%d", argCount)
		dataQuery += fmt.Sprintf(" AND p.category = $%d", argCount)
		args = append(args, req.Category)
	}

	if req.StoreID != "" {
		argCount++
		countQuery += fmt.Sprintf(" AND EXISTS(SELECT 1 FROM product_prices pp WHERE pp.product_id = p.id AND pp.store_id = $%d)", argCount)
		dataQuery += fmt.Sprintf(" AND EXISTS(SELECT 1 FROM product_prices ppf WHERE ppf.product_id = p.id AND ppf.store_id = $%d)", argCount)
		args = append(args, req.StoreID)
	}

	dataQuery += `
			GROUP BY p.id
		)
		SELECT id, name, category, COALESCE(last_price, 0), COALESCE(last_store, ''), COALESCE(last_seen, NOW()), COALESCE(min_price, 0), COALESCE(max_price, 0)
		FROM product_stats
	`

	// Add sorting
	sortBy := "last_seen"
	switch req.SortBy {
	case "name":
		sortBy = "name"
	case "lastPrice":
		sortBy = "last_price"
	case "lastSeen":
		sortBy = "last_seen"
	}

	sortOrder := "DESC"
	if req.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	dataQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Add pagination
	offset := (req.Page - 1) * req.Limit
	argCount++
	dataQuery += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, req.Limit)
	argCount++
	dataQuery += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	// Get total count
	var total int
	countArgs := args[:len(args)-2] // Remove limit and offset
	err := r.db.Pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get data
	rows, err := r.db.Pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []models.ProductListItem
	for rows.Next() {
		var item models.ProductListItem
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Category,
			&item.LastPrice,
			&item.LastStore,
			&item.LastSeen,
			&item.PriceRange.Min,
			&item.PriceRange.Max,
		)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, item)
	}

	return result, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, userID string, productID string) (*models.Product, error) {
	query := `
		SELECT DISTINCT p.id, p.name, COALESCE(p.category, ''), p.created_at
		FROM products p
		INNER JOIN receipt_items ri ON ri.product_id = p.id
		INNER JOIN receipts r ON r.id = ri.receipt_id AND r.user_id = $1
		WHERE p.id = $2
	`
	product := &models.Product{}
	err := r.db.Pool.QueryRow(ctx, query, userID, productID).
		Scan(&product.ID, &product.Name, &product.Category, &product.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return product, nil
}

func (r *Repository) GetPriceHistory(ctx context.Context, productID string) ([]models.ProductPrice, error) {
	query := `
		SELECT pp.id, pp.product_id, pp.store_id, s.name, pp.price, pp.recorded_at
		FROM product_prices pp
		INNER JOIN stores s ON s.id = pp.store_id
		WHERE pp.product_id = $1
		ORDER BY pp.store_id, pp.recorded_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ProductPrice
	for rows.Next() {
		var price models.ProductPrice
		err := rows.Scan(
			&price.ID,
			&price.ProductID,
			&price.StoreID,
			&price.StoreName,
			&price.Price,
			&price.RecordedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, price)
	}

	return result, rows.Err()
}

func (r *Repository) GetTotalPurchases(ctx context.Context, userID string, productID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM receipt_items ri
		INNER JOIN receipts r ON r.id = ri.receipt_id AND r.user_id = $1
		WHERE ri.product_id = $2
	`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, userID, productID).Scan(&count)
	return count, err
}

func (r *Repository) GetLastSeen(ctx context.Context, productID string) (*models.ProductPrice, error) {
	query := `
		SELECT pp.id, pp.product_id, pp.store_id, s.name, pp.price, pp.recorded_at
		FROM product_prices pp
		INNER JOIN stores s ON s.id = pp.store_id
		WHERE pp.product_id = $1
		ORDER BY pp.recorded_at DESC
		LIMIT 1
	`
	price := &models.ProductPrice{}
	err := r.db.Pool.QueryRow(ctx, query, productID).
		Scan(&price.ID, &price.ProductID, &price.StoreID, &price.StoreName, &price.Price, &price.RecordedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return price, nil
}

func (r *Repository) GetOrCreateByName(ctx context.Context, name string, category string) (*models.Product, error) {
	// Try to find existing product
	query := `
		SELECT id, name, COALESCE(category, ''), created_at
		FROM products
		WHERE name = $1
		LIMIT 1
	`
	product := &models.Product{}
	err := r.db.Pool.QueryRow(ctx, query, name).
		Scan(&product.ID, &product.Name, &product.Category, &product.CreatedAt)
	if err == nil {
		return product, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Create new product
	insertQuery := `
		INSERT INTO products (name, category)
		VALUES ($1, $2)
		RETURNING id, name, COALESCE(category, ''), created_at
	`
	err = r.db.Pool.QueryRow(ctx, insertQuery, name, category).
		Scan(&product.ID, &product.Name, &product.Category, &product.CreatedAt)
	if err != nil {
		return nil, err
	}

	return product, nil
}

func (r *Repository) RecordPrice(ctx context.Context, productID string, storeID string, price float64) error {
	query := `
		INSERT INTO product_prices (product_id, store_id, price)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Pool.Exec(ctx, query, productID, storeID, price)
	return err
}
