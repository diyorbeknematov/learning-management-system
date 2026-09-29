package postgres_test

import (
	"testing"

	"github.com/diyorbeknematov/lms/internal/config"
	"github.com/diyorbeknematov/lms/internal/repository"
	"github.com/diyorbeknematov/lms/internal/repository/postgres"
	"github.com/stretchr/testify/require"
)

type TestContext struct {
	DB   *postgres.Postgres
	Repo *repository.Repository
}

func TestNew(t *testing.T) {
	cfg := config.Load()

	db, err := postgres.New(cfg.DB)

	require.NoError(t, err)
	require.NotNil(t, db)
	require.NotNil(t, db.Pool)

	db.Close()
}

func setupTest(t *testing.T) *TestContext {
	t.Helper()

	cfg := config.Load()

	db, err := postgres.New(cfg.DB)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	repo := repository.NewRepository(db.Pool)

	return &TestContext{
		DB:   db,
		Repo: repo,
	}
}
