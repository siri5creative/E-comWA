package testutil

import "testing"

func TestNewTestDBAppliesMigrations(t *testing.T) {
	pool := NewTestDB(t)

	var tableCount int
	err := pool.QueryRow(t.Context(), `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public'
	`).Scan(&tableCount)
	if err != nil {
		t.Fatalf("failed to count tables: %v", err)
	}
	if tableCount < 10 {
		t.Fatalf("expected migrations to create at least 10 tables, found %d", tableCount)
	}

	var hasPaymentMethod bool
	err = pool.QueryRow(t.Context(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'orders' AND column_name = 'payment_method'
		)
	`).Scan(&hasPaymentMethod)
	if err != nil {
		t.Fatalf("failed to check payment_method column: %v", err)
	}
	if !hasPaymentMethod {
		t.Fatal("expected migration 0003 (payment_method column) to have applied")
	}
}
