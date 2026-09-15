//go:build unit

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestPrepaidBalanceRequiresCompletedPaidBalanceOrder(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, balance REAL, deleted_at TEXT);
	CREATE TABLE payment_orders (user_id INTEGER, order_type TEXT, status TEXT, paid_at TEXT, amount REAL, pay_amount REAL, refund_amount REAL);
	INSERT INTO users VALUES (7, 20, NULL), (8, 20, NULL);`)
	require.NoError(t, err)
	repo := newUserRepositoryWithSQL(nil, db)
	ctx := context.Background()
	initial, err := repo.GetPrepaidBalance(ctx, 7)
	require.NoError(t, err)
	require.False(t, initial.HasPurchased)
	for _, tc := range []struct {
		name, kind, status string
		paid               any
		refund             float64
		want               bool
	}{
		{"pending", "balance", "PENDING", nil, 0, false},
		{"not credited yet", "balance", "PAID", "2026-09-15", 0, false},
		{"failed fulfillment", "balance", "FAILED", "2026-09-15", 0, false},
		{"subscription", "subscription", "COMPLETED", "2026-09-15", 0, false},
		{"missing confirmation", "balance", "COMPLETED", nil, 0, false},
		{"paid and credited", "balance", "COMPLETED", "2026-09-15", 0, true},
		{"partial refund", "balance", "PARTIALLY_REFUNDED", "2026-09-15", 5, true},
		{"full refund", "balance", "REFUNDED", "2026-09-15", 20, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.Exec("DELETE FROM payment_orders")
			require.NoError(t, err)
			_, err = db.Exec("INSERT INTO payment_orders VALUES (7, ?, ?, ?, 20, 10, ?)", tc.kind, tc.status, tc.paid, tc.refund)
			require.NoError(t, err)
			balance, err := repo.GetPrepaidBalance(ctx, 7)
			require.NoError(t, err)
			require.Equal(t, tc.want, balance.HasPurchased)
			require.Equal(t, 20.0, balance.Balance)
			other, err := repo.GetPrepaidBalance(ctx, 8)
			require.NoError(t, err)
			require.False(t, other.HasPurchased)
		})
	}
}
