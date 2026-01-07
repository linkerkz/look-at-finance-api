package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts/models"
	"github.com/lukivan8/look-at-finance-api/internal/shared/database"
)

var (
	ErrReceiptNotFound  = errors.New("receipt not found")
	ErrDuplicateReceipt = errors.New("receipt with this fiscal ID already exists")
)

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, receipt *models.Receipt) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Check for duplicate fiscal ID for this user
	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM receipts WHERE fiscal_id = $1 AND user_id = $2)", receipt.FiscalID, receipt.UserID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateReceipt
	}

	// Insert receipt
	query := `
		INSERT INTO receipts (user_id, store_id, fiscal_id, purchase_date, total_amount, raw_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err = tx.QueryRow(ctx, query,
		receipt.UserID,
		receipt.StoreID,
		receipt.FiscalID,
		receipt.PurchaseDate,
		receipt.TotalAmount,
		receipt.RawURL,
	).Scan(&receipt.ID, &receipt.CreatedAt)
	if err != nil {
		return err
	}

	// Insert items
	for i := range receipt.Items {
		item := &receipt.Items[i]
		itemQuery := `
			INSERT INTO receipt_items (receipt_id, product_id, name, quantity, price, total_price, unit)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`
		var productID interface{}
		if item.ProductID != "" {
			productID = item.ProductID
		}
		err = tx.QueryRow(ctx, itemQuery,
			receipt.ID,
			productID,
			item.Name,
			item.Quantity,
			item.Price,
			item.TotalPrice,
			item.Unit,
		).Scan(&item.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, userID string, receiptID string) (*models.Receipt, error) {
	query := `
		SELECT 
			r.id, r.user_id, r.store_id, r.fiscal_id, r.purchase_date, r.total_amount, r.raw_url, r.created_at,
			s.name as store_name, COALESCE(s.address, '') as store_address
		FROM receipts r
		LEFT JOIN stores s ON s.id = r.store_id
		WHERE r.id = $1 AND r.user_id = $2
	`
	receipt := &models.Receipt{}
	err := r.db.Pool.QueryRow(ctx, query, receiptID, userID).Scan(
		&receipt.ID,
		&receipt.UserID,
		&receipt.StoreID,
		&receipt.FiscalID,
		&receipt.PurchaseDate,
		&receipt.TotalAmount,
		&receipt.RawURL,
		&receipt.CreatedAt,
		&receipt.StoreName,
		&receipt.StoreAddress,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReceiptNotFound
		}
		return nil, err
	}

	// Get items
	items, err := r.GetItems(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	receipt.Items = items

	return receipt, nil
}

func (r *Repository) GetItems(ctx context.Context, receiptID string) ([]models.ReceiptItem, error) {
	query := `
		SELECT id, receipt_id, COALESCE(product_id::text, ''), name, quantity, price, total_price, unit
		FROM receipt_items
		WHERE receipt_id = $1
		ORDER BY id
	`
	rows, err := r.db.Pool.Query(ctx, query, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ReceiptItem
	for rows.Next() {
		var item models.ReceiptItem
		err := rows.Scan(
			&item.ID,
			&item.ReceiptID,
			&item.ProductID,
			&item.Name,
			&item.Quantity,
			&item.Price,
			&item.TotalPrice,
			&item.Unit,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) List(ctx context.Context, userID string, req models.ListReceiptsRequest) ([]models.Receipt, int, error) {
	// Build query
	countQuery := `
		SELECT COUNT(*)
		FROM receipts r
		WHERE r.user_id = $1
	`

	dataQuery := `
		SELECT 
			r.id, r.user_id, r.store_id, r.fiscal_id, r.purchase_date, r.total_amount, r.raw_url, r.created_at,
			s.name as store_name, COALESCE(s.address, '') as store_address,
			(SELECT COUNT(*) FROM receipt_items ri WHERE ri.receipt_id = r.id) as item_count
		FROM receipts r
		LEFT JOIN stores s ON s.id = r.store_id
		WHERE r.user_id = $1
	`

	args := []interface{}{userID}
	argCount := 1

	if req.StartDate != "" {
		argCount++
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			countQuery += fmt.Sprintf(" AND r.purchase_date >= $%d", argCount)
			dataQuery += fmt.Sprintf(" AND r.purchase_date >= $%d", argCount)
			args = append(args, startDate)
		}
	}

	if req.EndDate != "" {
		argCount++
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err == nil {
			// Add 1 day to include the end date
			endDate = endDate.Add(24 * time.Hour)
			countQuery += fmt.Sprintf(" AND r.purchase_date < $%d", argCount)
			dataQuery += fmt.Sprintf(" AND r.purchase_date < $%d", argCount)
			args = append(args, endDate)
		}
	}

	if req.StoreID != "" {
		argCount++
		countQuery += fmt.Sprintf(" AND r.store_id = $%d", argCount)
		dataQuery += fmt.Sprintf(" AND r.store_id = $%d", argCount)
		args = append(args, req.StoreID)
	}

	dataQuery += " ORDER BY r.purchase_date DESC"

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
	countArgs := args[:len(args)-2]
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

	var result []models.Receipt
	for rows.Next() {
		var receipt models.Receipt
		var itemCount int
		err := rows.Scan(
			&receipt.ID,
			&receipt.UserID,
			&receipt.StoreID,
			&receipt.FiscalID,
			&receipt.PurchaseDate,
			&receipt.TotalAmount,
			&receipt.RawURL,
			&receipt.CreatedAt,
			&receipt.StoreName,
			&receipt.StoreAddress,
			&itemCount,
		)
		if err != nil {
			return nil, 0, err
		}
		// Store item count temporarily in Items slice
		receipt.Items = make([]models.ReceiptItem, itemCount)
		result = append(result, receipt)
	}

	return result, total, rows.Err()
}

func (r *Repository) ExistsByFiscalID(ctx context.Context, userID string, fiscalID string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM receipts WHERE fiscal_id = $1 AND user_id = $2)", fiscalID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) Delete(ctx context.Context, userID string, receiptID string) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Verify receipt belongs to user
	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM receipts WHERE id = $1 AND user_id = $2)", receiptID, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrReceiptNotFound
	}

	// Delete receipt items first (foreign key constraint)
	_, err = tx.Exec(ctx, "DELETE FROM receipt_items WHERE receipt_id = $1", receiptID)
	if err != nil {
		return err
	}

	// Delete receipt
	_, err = tx.Exec(ctx, "DELETE FROM receipts WHERE id = $1 AND user_id = $2", receiptID, userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
