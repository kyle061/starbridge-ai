//go:build unit

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestPrepaidBalanceUsesLiveCreditWithoutOnlinePaymentTables(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, balance REAL, deleted_at TEXT);
    INSERT INTO users VALUES (7, 0, NULL), (8, 0, NULL);`)
	require.NoError(t, err)
	repo := newUserRepositoryWithSQL(nil, db)
	ctx := context.Background()
	// No payment order is needed: both admin and redeem credit reach this balance.
	for _, amount := range []float64{0, 20, 0, -1, 19} {
		_, err = db.Exec("UPDATE users SET balance = ? WHERE id = 7", amount)
		require.NoError(t, err)
		balance, err := repo.GetPrepaidBalance(ctx, 7)
		require.NoError(t, err)
		require.Equal(t, amount, balance.Balance)
		require.Equal(t, amount > 0, balance.HasPurchased)
		other, err := repo.GetPrepaidBalance(ctx, 8)
		require.NoError(t, err)
		require.Zero(t, other.Balance)
	}
	_, err = db.Exec("UPDATE users SET deleted_at = '2026-09-15' WHERE id = 7")
	require.NoError(t, err)
	_, err = repo.GetPrepaidBalance(ctx, 7)
	require.Error(t, err)
}
