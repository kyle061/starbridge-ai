package repository

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsDashboardErrorWhereExcludesClientDisconnects(t *testing.T) {
	start := time.Date(2026, time.September, 26, 0, 0, 0, 0, time.UTC)
	where, args, _ := buildErrorWhere(nil, start, start.Add(time.Hour), 1)
	require.Contains(t, where, "status_code IS DISTINCT FROM 499")
	require.Len(t, args, 2)
	require.Contains(t, where, "is_count_tokens = FALSE")
}

func TestOpsHourlyPreaggregationExcludesClientDisconnects(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	start := time.Date(2026, time.September, 26, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	mock.ExpectExec(`(?s)FROM ops_error_logs.*AND is_count_tokens = FALSE\s+AND status_code IS DISTINCT FROM 499`).
		WithArgs(start, end).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, (&opsRepository{db: db}).UpsertHourlyMetrics(context.Background(), start, end))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertSystemMetricsNullableIntegerMetrics(t *testing.T) {
	createdAt := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC)
	zero := 0

	tests := []struct {
		name         string
		dbConnActive *int
		wantDBActive driver.Value
	}{
		{name: "explicit zero is preserved", dbConnActive: &zero, wantDBActive: int64(0)},
		{name: "unavailable metric remains null", dbConnActive: nil, wantDBActive: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			t.Cleanup(func() { _ = db.Close() })

			args := make([]driver.Value, 40)
			args[0] = createdAt
			args[1] = int64(1)
			for i := 4; i <= 12; i++ {
				args[i] = int64(0)
			}
			args[35] = tt.wantDBActive

			mock.ExpectExec("INSERT INTO ops_system_metrics").
				WithArgs(args...).
				WillReturnResult(sqlmock.NewResult(1, 1))

			repo := &opsRepository{db: db}
			err = repo.InsertSystemMetrics(context.Background(), &service.OpsInsertSystemMetricsInput{
				CreatedAt:    createdAt,
				DBConnActive: tt.dbConnActive,
			})
			if err != nil {
				t.Fatalf("InsertSystemMetrics: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet SQL expectations: %v", err)
			}
		})
	}
}
