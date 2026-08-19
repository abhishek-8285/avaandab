package sql

import (
	"database/sql"
)

// nullIfZero converts a zero float64 into an SQL NULL (columns are REAL
// nullable in migration 00042).
func nullIfZero(v float64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

// polygonOrNull converts an empty polygon string to SQL NULL.
func polygonOrNull(v string) interface{} {
	if v == "" {
		return nil
	}
	return v
}

// boolToInt converts a Go bool to SQLite INTEGER.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// nullFloatPtr converts a *float64 into a nullable scan target.
func nullFloatPtr(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *p, Valid: true}
}

// nullStringToPtr converts a scanned NullString into a *string.
func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}
