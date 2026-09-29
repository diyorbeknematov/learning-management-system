package repository

import (
	"context"
	"fmt"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
	User
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		User: postgres.NewUserRepository(db),
	}
}

func (r *Repository) WithTx(ctx context.Context, fn func(txRepo *Repository) error) error {
	tx, err := r.db.Begin(ctx)

	if err != nil {
		return fmt.Errorf("tx boshlashda xato: %w", err)
	}

	txRepo := &Repository{
		db:   r.db,
		User: postgres.NewUserRepository(tx),
	}

	if err := fn(txRepo); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback xato: %v (asosiy: %w)", rbErr, err)
		}
		return err
	}

	return tx.Commit(ctx)
}

type User interface {
	Create(context.Context, models.CreateUser) (uuid.UUID, error)
	Update(context.Context, models.UpdateUser) (*models.User, error)
	UpdateStatus(context.Context, string, string,
	) error
	Delete(context.Context, string) error
	GetByID(context.Context, string) (*models.User, error)
	GetList(context.Context, models.UserFilter) ([]models.User, int, error)
}
