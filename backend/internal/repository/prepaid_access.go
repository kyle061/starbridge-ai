package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var _ service.PrepaidAccessRepository = (*userRepository)(nil)

func (r *userRepository) GetPrepaidBalance(ctx context.Context, userID int64) (*service.PrepaidBalance, error) {
	const query = `SELECT u.balance FROM users u WHERE u.id = $1 AND u.deleted_at IS NULL`
	executor := txAwareSQLExecutor(ctx, r.sql, r.client)
	if executor == nil {
		return nil, fmt.Errorf("prepaid balance database is unavailable")
	}
	rows, err := executor.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrUserNotFound
	}
	var balance service.PrepaidBalance
	if err := rows.Scan(&balance.Balance); err != nil {
		return nil, err
	}
	// Retain the legacy field for clients; offline funding needs no payment order.
	balance.HasPurchased = balance.Balance > 0
	return &balance, rows.Err()
}
