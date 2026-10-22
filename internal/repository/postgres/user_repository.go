package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jie10/greenlight/internal/domain"
	"github.com/jie10/greenlight/internal/repository"
)

const queryTimeout = 3 * time.Second

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

var _ repository.UserRepository = (*UserRepository)(nil)

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		INSERT INTO users (id, name, email, password_hash, activated)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at, version`

	err := r.pool.QueryRow(ctx, query, u.ID, u.Name, u.Email, u.HashedPassword, u.Activated).
		Scan(&u.CreatedAt, &u.UpdatedAt, &u.Version)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
			return domain.ErrDuplicateEmail
		}
		return err
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, created_at, updated_at, name, email, password_hash, activated, version
		FROM users
		WHERE id = $1`

	return r.scanOne(ctx, query, id)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, created_at, updated_at, name, email, password_hash, activated, version
		FROM users
		WHERE email = $1`

	return r.scanOne(ctx, query, email)
}

func (r *UserRepository) scanOne(ctx context.Context, query string, arg any) (*domain.User, error) {
	u := &domain.User{}

	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Name, &u.Email,
		&u.HashedPassword, &u.Activated, &u.Version,
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrNotFound
	case err != nil:
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, activated = $4,
		    updated_at = now(), version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING updated_at, version`

	err := r.pool.QueryRow(ctx, query,
		u.Name, u.Email, u.HashedPassword, u.Activated, u.ID, u.Version,
	).Scan(&u.UpdatedAt, &u.Version)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.ErrEditConflict
	case err != nil:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
			return domain.ErrDuplicateEmail
		}
		return err
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
