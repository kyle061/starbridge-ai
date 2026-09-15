package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var _ service.PrepaidAccessRepository = (*userRepository)(nil)

func (r *userRepository) GetPrepaidBalance(ctx context.Context, userID int64) (*service.PrepaidBalance, error) {
	const query = `SELECT u.balance, EXISTS (
		SELECT 1 FROM payment_orders p
		WHERE p.user_id = u.id AND p.order_type = 'balance'
		AND (p.status = 'COMPLETED' OR (p.status = 'PARTIALLY_REFUNDED' AND p.refund_amount < p.amount))
		AND p.paid_at IS NOT NULL
		AND p.amount > 0 AND p.pay_amount > 0
	) FROM users u WHERE u.id = $1 AND u.deleted_at IS NULL`
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
	if err := rows.Scan(&balance.Balance, &balance.HasPurchased); err != nil {
		return nil, err
	}
	return &balance, rows.Err()
}
