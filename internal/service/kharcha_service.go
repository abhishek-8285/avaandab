package service

import (
	"context"
	"fmt"
	"time"

	"transport-app/internal/repository"
)

// KharchaExpense is the service-layer view of a driver expense with joined trip/driver data.
type KharchaExpense struct {
	ID             string
	TripID         string
	TripNumber     string
	DriverID       string
	DriverName     string
	Category       string // advance | fuel | toll | food | repair | other
	Amount         float64
	Description    string
	ReceiptURL     *string
	Status         string // pending | approved | rejected | settled
	RejectedReason *string
	ApprovedBy     *string
	ApprovedAt     *time.Time
	CreatedAt      time.Time
}

// KharchaStats holds dashboard summary counts/totals.
type KharchaStats struct {
	PendingCount   int
	ApprovedToday  int
	MonthTotal     float64
	UnsettledTotal float64
}

// KharchaService manages driver expense (kharcha) approvals and the ledger.
// It uses raw SQL because driver_expenses has no generated SQLC repository methods yet.
type KharchaService struct {
	baseService
}

// ListPendingExpenses returns all driver expenses awaiting approval.

func (s *KharchaService) ListPendingExpenses(ctx context.Context) ([]KharchaExpense, error) {
	getter, ok := s.store.(repository.DBGetter)
	if !ok {
		return nil, fmt.Errorf("store does not support raw DB access")
	}
	db := getter.DB()

	rows, err := db.QueryContext(ctx, `
		SELECT de.id,
		       COALESCE(de.trip_id, '') AS trip_id,
		       COALESCE(t.trip_number, '') AS trip_number,
		       COALESCE(de.driver_id, '') AS driver_id,
		       COALESCE(d.first_name||' '||d.last_name, '') AS driver_name,
		       COALESCE(de.category, de.expense_type, 'other') AS category,
		       de.amount,
		       COALESCE(de.description, '') AS description,
		       de.receipt_url,
		       COALESCE(de.status, 'pending') AS status,
		       de.rejected_reason,
		       de.approved_by,
		       de.approved_at,
		       de.created_at
		FROM driver_expenses de
		LEFT JOIN trips t ON t.id = de.trip_id
		LEFT JOIN drivers d ON d.id = de.driver_id
		WHERE COALESCE(de.status, 'pending') = 'pending'
		ORDER BY de.created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKharchaRows(rows)
}

// ListLedger returns expenses filtered by tripID (empty = all), newest first.
func (s *KharchaService) ListLedger(ctx context.Context, tripID string) ([]KharchaExpense, error) {
	getter, ok := s.store.(repository.DBGetter)
	if !ok {
		return nil, fmt.Errorf("store does not support raw DB access")
	}
	db := getter.DB()

	query := `
		SELECT de.id,
		       COALESCE(de.trip_id, '') AS trip_id,
		       COALESCE(t.trip_number, '') AS trip_number,
		       COALESCE(de.driver_id, '') AS driver_id,
		       COALESCE(d.first_name||' '||d.last_name, '') AS driver_name,
		       COALESCE(de.category, de.expense_type, 'other') AS category,
		       de.amount,
		       COALESCE(de.description, '') AS description,
		       de.receipt_url,
		       COALESCE(de.status, 'pending') AS status,
		       de.rejected_reason,
		       de.approved_by,
		       de.approved_at,
		       de.created_at
		FROM driver_expenses de
		LEFT JOIN trips t ON t.id = de.trip_id
		LEFT JOIN drivers d ON d.id = de.driver_id`
	args := []interface{}{}
	if tripID != "" {
		query += " WHERE de.trip_id = ?"
		args = append(args, tripID)
	}
	query += " ORDER BY de.created_at DESC LIMIT 200"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKharchaRows(rows)
}

// GetExpenseByID retrieves a single expense with joined data.
func (s *KharchaService) GetExpenseByID(ctx context.Context, id string) (KharchaExpense, error) {
	getter, ok := s.store.(repository.DBGetter)
	if !ok {
		return KharchaExpense{}, fmt.Errorf("store does not support raw DB access")
	}
	db := getter.DB()

	row := db.QueryRowContext(ctx, `
		SELECT de.id,
		       COALESCE(de.trip_id, '') AS trip_id,
		       COALESCE(t.trip_number, '') AS trip_number,
		       COALESCE(de.driver_id, '') AS driver_id,
		       COALESCE(d.first_name||' '||d.last_name, '') AS driver_name,
		       COALESCE(de.category, de.expense_type, 'other') AS category,
		       de.amount,
		       COALESCE(de.description, '') AS description,
		       de.receipt_url,
		       COALESCE(de.status, 'pending') AS status,
		       de.rejected_reason,
		       de.approved_by,
		       de.approved_at,
		       de.created_at
		FROM driver_expenses de
		LEFT JOIN trips t ON t.id = de.trip_id
		LEFT JOIN drivers d ON d.id = de.driver_id
		WHERE de.id = ?`, id)

	var e KharchaExpense
	var receiptURL, rejectedReason, approvedBy *string
	var approvedAt *time.Time
	if err := row.Scan(
		&e.ID, &e.TripID, &e.TripNumber, &e.DriverID, &e.DriverName,
		&e.Category, &e.Amount, &e.Description, &receiptURL,
		&e.Status, &rejectedReason, &approvedBy, &approvedAt, &e.CreatedAt,
	); err != nil {
		return KharchaExpense{}, fmt.Errorf("expense not found: %w", err)
	}
	e.ReceiptURL = receiptURL
	e.RejectedReason = rejectedReason
	e.ApprovedBy = approvedBy
	e.ApprovedAt = approvedAt
	return e, nil
}

// ApproveExpense approves an expense in a transaction and deducts from the driver's settlement.
func (s *KharchaService) ApproveExpense(ctx context.Context, expenseID, approvedByUserID string) error {
	getter, ok := s.store.(repository.DBGetter)
	if !ok {
		return fmt.Errorf("store does not support raw DB access")
	}
	db := getter.DB()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()

	// 1. Mark approved (only if currently pending)
	res, err := tx.ExecContext(ctx,
		`UPDATE driver_expenses
		 SET status = 'approved', approved_by = ?, approved_at = ?
		 WHERE id = ? AND COALESCE(status, 'pending') = 'pending'`,
		approvedByUserID, now, expenseID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("expense already processed or not found")
	}

	// 2. Fetch trip_id, driver_id, amount to update settlement
	var tripID, driverID string
	var amount float64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(trip_id,''), COALESCE(driver_id,''), amount FROM driver_expenses WHERE id = ?`,
		expenseID).Scan(&tripID, &driverID, &amount); err != nil {
		return err
	}

	// 3. Deduct from settlement net_payout if a settlement row exists for this trip
	if tripID != "" && driverID != "" {
		_, _ = tx.ExecContext(ctx,
			`UPDATE driver_settlements
			 SET advances_kharcha = advances_kharcha + ?,
			     net_payout = MAX(0.0, net_payout - ?)
			 WHERE trip_id = ? AND driver_id = ?`,
			amount, amount, tripID, driverID)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.logAudit(ctx, nil, "approve_kharcha", "driver_expenses", expenseID, nil, nil)
	s.log.Info("kharcha approved", "expense_id", expenseID, "amount", amount, "by", approvedByUserID)
	return nil
}

// RejectExpense rejects an expense with a mandatory reason.
func (s *KharchaService) RejectExpense(ctx context.Context, expenseID, rejectedByUserID, reason string) error {
	if reason == "" {
		return fmt.Errorf("rejection reason is required")
	}
	getter, ok := s.store.(repository.DBGetter)
	if !ok {
		return fmt.Errorf("store does not support raw DB access")
	}
	db := getter.DB()

	res, err := db.ExecContext(ctx,
		`UPDATE driver_expenses
		 SET status = 'rejected', approved_by = ?, rejected_reason = ?
		 WHERE id = ? AND COALESCE(status, 'pending') = 'pending'`,
		rejectedByUserID, reason, expenseID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("expense already processed or not found")
	}

	s.logAudit(ctx, nil, "reject_kharcha", "driver_expenses", expenseID, nil, &reason)
	s.log.Info("kharcha rejected", "expense_id", expenseID, "reason", reason, "by", rejectedByUserID)
	return nil
}

// CreateExpense logs a new driver kharcha claim.
func (s *KharchaService) CreateExpense(ctx context.Context, tripID, driverID, category string, amount float64, description, receiptURL string) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be greater than zero")
	}
	validCategories := map[string]bool{
		"advance": true, "fuel": true, "toll": true, "food": true, "repair": true, "other": true,
	}
	if !validCategories[category] {
		return "", fmt.Errorf("invalid category: %s", category)
	}

	getter, ok := s.store.(repository.DBGetter)
	if !ok {
		return "", fmt.Errorf("store does not support raw DB access")
	}
	db := getter.DB()

	expID := generateID()
	var recURL interface{} = nil
	if receiptURL != "" {
		recURL = receiptURL
	}
	var desc interface{} = nil
	if description != "" {
		desc = description
	}
	var tID interface{} = nil
	if tripID != "" {
		tID = tripID
	}
	var dID interface{} = nil
	if driverID != "" {
		dID = driverID
	}

	_, err := db.ExecContext(ctx,
		`INSERT INTO driver_expenses
		 (id, trip_id, driver_id, expense_type, category, amount, description, receipt_url, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
		expID, tID, dID, category, category, amount, desc, recURL, time.Now())
	if err != nil {
		return "", err
	}

	s.logAudit(ctx, nil, "create_kharcha", "driver_expenses", expID, nil, nil)
	s.log.Info("kharcha created", "expense_id", expID, "driver_id", driverID, "amount", amount)
	return expID, nil
}

// GetKharchaStats returns dashboard summary statistics.
func (s *KharchaService) GetKharchaStats(ctx context.Context) (KharchaStats, error) {
	getter, ok := s.store.(repository.DBGetter)
	if !ok {
		return KharchaStats{}, fmt.Errorf("store does not support raw DB access")
	}
	db := getter.DB()

	var stats KharchaStats

	_ = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM driver_expenses WHERE COALESCE(status,'pending') = 'pending'`).
		Scan(&stats.PendingCount)

	today := time.Now().Format("2006-01-02")
	_ = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM driver_expenses WHERE status = 'approved' AND DATE(approved_at) = ?`, today).
		Scan(&stats.ApprovedToday)

	_ = db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM driver_expenses
		 WHERE status = 'approved' AND strftime('%Y-%m', approved_at) = strftime('%Y-%m','now')`).
		Scan(&stats.MonthTotal)

	_ = db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM driver_expenses
		 WHERE COALESCE(status,'pending') IN ('pending','approved')`).
		Scan(&stats.UnsettledTotal)

	return stats, nil
}

// --- internal helpers ---

type kharchaScanner interface {
	Next() bool
	Scan(...interface{}) error
	Close() error
}

func scanKharchaRows(rows kharchaScanner) ([]KharchaExpense, error) {
	var expenses []KharchaExpense
	for rows.Next() {
		var e KharchaExpense
		var receiptURL, rejectedReason, approvedBy *string
		var approvedAt *time.Time
		if err := rows.Scan(
			&e.ID, &e.TripID, &e.TripNumber, &e.DriverID, &e.DriverName,
			&e.Category, &e.Amount, &e.Description, &receiptURL,
			&e.Status, &rejectedReason, &approvedBy, &approvedAt, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		e.ReceiptURL = receiptURL
		e.RejectedReason = rejectedReason
		e.ApprovedBy = approvedBy
		e.ApprovedAt = approvedAt
		expenses = append(expenses, e)
	}
	return expenses, rows.Close()
}
