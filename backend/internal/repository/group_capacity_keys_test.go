package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestListActiveAPIKeyIDsByGroupIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := newGroupRepositoryWithSQL(nil, db)

	mock.ExpectQuery("SELECT group_id, id FROM api_keys").
		WithArgs("{10,20}", service.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "id"}).
			AddRow(int64(10), int64(101)).AddRow(int64(20), int64(201)))
	rows, err := repo.ListActiveAPIKeyIDsByGroupIDs(context.Background(), []int64{10, 20})
	require.NoError(t, err)
	require.Equal(t, []service.GroupCapacityAPIKeyRow{
		{GroupID: 10, APIKeyID: 101}, {GroupID: 20, APIKeyID: 201},
	}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}
