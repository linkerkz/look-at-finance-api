package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/lukivan8/look-at-finance-api/internal/modules/expenses/models"
	"github.com/lukivan8/look-at-finance-api/internal/shared/database"
)

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetSummary(ctx context.Context, userID string, startDate, endDate time.Time) (*models.SummaryResponse, error) {
	// Get total spent and receipt count
	var totalSpent float64
	var receiptCount int
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM receipts
		WHERE user_id = $1 AND purchase_date >= $2 AND purchase_date < $3
	`, userID, startDate, endDate).Scan(&totalSpent, &receiptCount)
	if err != nil {
		return nil, err
	}

	var avgPerReceipt float64
	if receiptCount > 0 {
		avgPerReceipt = totalSpent / float64(receiptCount)
	}

	// Get stats by store
	storeStats, err := r.getStoreStats(ctx, userID, startDate, endDate, totalSpent)
	if err != nil {
		return nil, err
	}

	// Get stats by category (based on most common item category in receipt)
	categoryStats, err := r.getCategoryStats(ctx, userID, startDate, endDate, totalSpent)
	if err != nil {
		return nil, err
	}

	// Get trend data
	trend, err := r.getTrend(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get comparison with previous period
	comparison, err := r.getComparison(ctx, userID, startDate, endDate, totalSpent)
	if err != nil {
		return nil, err
	}

	return &models.SummaryResponse{
		Period: models.PeriodInfo{
			Start: startDate.Format("2006-01-02"),
			End:   endDate.Add(-time.Second).Format("2006-01-02"),
		},
		TotalSpent:        totalSpent,
		ReceiptCount:      receiptCount,
		AveragePerReceipt: avgPerReceipt,
		ByCategory:        categoryStats,
		ByStore:           storeStats,
		Trend:             trend,
		Comparison:        comparison,
	}, nil
}

func (r *Repository) getStoreStats(ctx context.Context, userID string, startDate, endDate time.Time, totalSpent float64) ([]models.StoreStat, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT s.id, s.name, COALESCE(SUM(r.total_amount), 0) as amount
		FROM receipts r
		INNER JOIN stores s ON s.id = r.store_id
		WHERE r.user_id = $1 AND r.purchase_date >= $2 AND r.purchase_date < $3
		GROUP BY s.id, s.name
		ORDER BY amount DESC
		LIMIT 10
	`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.StoreStat
	for rows.Next() {
		var stat models.StoreStat
		if err := rows.Scan(&stat.StoreID, &stat.StoreName, &stat.Amount); err != nil {
			return nil, err
		}
		if totalSpent > 0 {
			stat.Percentage = (stat.Amount / totalSpent) * 100
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

func (r *Repository) getCategoryStats(ctx context.Context, userID string, startDate, endDate time.Time, totalSpent float64) ([]models.CategoryStat, error) {
	// Group by product category
	rows, err := r.db.Pool.Query(ctx, `
		SELECT COALESCE(p.category, 'Uncategorized') as category, SUM(ri.total_price) as amount
		FROM receipts r
		INNER JOIN receipt_items ri ON ri.receipt_id = r.id
		LEFT JOIN products p ON p.id = ri.product_id
		WHERE r.user_id = $1 AND r.purchase_date >= $2 AND r.purchase_date < $3
		GROUP BY p.category
		ORDER BY amount DESC
		LIMIT 10
	`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.CategoryStat
	for rows.Next() {
		var stat models.CategoryStat
		if err := rows.Scan(&stat.Category, &stat.Amount); err != nil {
			return nil, err
		}
		if totalSpent > 0 {
			stat.Percentage = (stat.Amount / totalSpent) * 100
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

func (r *Repository) getTrend(ctx context.Context, userID string, startDate, endDate time.Time) ([]models.TrendPoint, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT DATE(purchase_date) as date, COALESCE(SUM(total_amount), 0) as amount
		FROM receipts
		WHERE user_id = $1 AND purchase_date >= $2 AND purchase_date < $3
		GROUP BY DATE(purchase_date)
		ORDER BY date
	`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trend []models.TrendPoint
	for rows.Next() {
		var point models.TrendPoint
		var date time.Time
		if err := rows.Scan(&date, &point.Amount); err != nil {
			return nil, err
		}
		point.Date = date.Format("2006-01-02")
		trend = append(trend, point)
	}

	return trend, rows.Err()
}

func (r *Repository) getComparison(ctx context.Context, userID string, startDate, endDate time.Time, currentTotal float64) (models.Comparison, error) {
	// Calculate the previous period of the same length
	duration := endDate.Sub(startDate)
	prevEnd := startDate
	prevStart := prevEnd.Add(-duration)

	var prevTotal float64
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM receipts
		WHERE user_id = $1 AND purchase_date >= $2 AND purchase_date < $3
	`, userID, prevStart, prevEnd).Scan(&prevTotal)
	if err != nil {
		return models.Comparison{}, err
	}

	comparison := models.Comparison{
		PreviousPeriod: prevTotal,
	}

	if prevTotal > 0 {
		change := ((currentTotal - prevTotal) / prevTotal) * 100
		comparison.Change = change
		if change > 0 {
			comparison.ChangeType = "increase"
		} else if change < 0 {
			comparison.ChangeType = "decrease"
			comparison.Change = -change // Make it positive for display
		} else {
			comparison.ChangeType = "same"
		}
	} else if currentTotal > 0 {
		comparison.Change = 100
		comparison.ChangeType = "increase"
	} else {
		comparison.ChangeType = "same"
	}

	return comparison, nil
}

func (r *Repository) ListExpenses(ctx context.Context, userID string, req models.ListExpensesRequest) ([]models.ExpenseListItem, int, error) {
	// Build query
	countQuery := `
		SELECT COUNT(*)
		FROM receipts r
		WHERE r.user_id = $1
	`

	dataQuery := `
		SELECT 
			r.id, r.purchase_date, s.name as store_name, s.id as store_id, r.total_amount,
			COALESCE((
				SELECT p.category FROM receipt_items ri 
				LEFT JOIN products p ON p.id = ri.product_id 
				WHERE ri.receipt_id = r.id AND p.category IS NOT NULL 
				LIMIT 1
			), 'Uncategorized') as category,
			(SELECT COUNT(*) FROM receipt_items ri WHERE ri.receipt_id = r.id) as item_count
		FROM receipts r
		LEFT JOIN stores s ON s.id = r.store_id
		WHERE r.user_id = $1
	`

	args := []interface{}{userID}
	argCount := 1

	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			argCount++
			countQuery += fmt.Sprintf(" AND r.purchase_date >= $%d", argCount)
			dataQuery += fmt.Sprintf(" AND r.purchase_date >= $%d", argCount)
			args = append(args, startDate)
		}
	}

	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err == nil {
			endDate = endDate.Add(24 * time.Hour)
			argCount++
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

	if req.MinAmount > 0 {
		argCount++
		countQuery += fmt.Sprintf(" AND r.total_amount >= $%d", argCount)
		dataQuery += fmt.Sprintf(" AND r.total_amount >= $%d", argCount)
		args = append(args, req.MinAmount)
	}

	if req.MaxAmount > 0 {
		argCount++
		countQuery += fmt.Sprintf(" AND r.total_amount <= $%d", argCount)
		dataQuery += fmt.Sprintf(" AND r.total_amount <= $%d", argCount)
		args = append(args, req.MaxAmount)
	}

	// Category filter requires a subquery
	if req.Category != "" {
		argCount++
		categoryFilter := fmt.Sprintf(` AND EXISTS(
			SELECT 1 FROM receipt_items ri 
			LEFT JOIN products p ON p.id = ri.product_id 
			WHERE ri.receipt_id = r.id AND p.category = $%d
		)`, argCount)
		countQuery += categoryFilter
		dataQuery += categoryFilter
		args = append(args, req.Category)
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

	var result []models.ExpenseListItem
	for rows.Next() {
		var item models.ExpenseListItem
		err := rows.Scan(
			&item.ID,
			&item.Date,
			&item.StoreName,
			&item.StoreID,
			&item.Amount,
			&item.Category,
			&item.ItemCount,
		)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, item)
	}

	return result, total, rows.Err()
}
