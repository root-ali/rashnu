package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresRepository implements the Repository interface using PostgreSQL
func NewPostgresRepository(pool *pgxpool.Pool, logger *zap.Logger) Repository {
	return &PostgresRepository{
		pool:   pool,
		logger: logger,
	}
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user User) error {
	const q = `
		INSERT INTO users (id, full_name, email, role, status, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, q,
		user.ID,
		user.FullName,
		user.Email,
		user.Role,
		user.Status,
		user.Password,
		user.CreateAt,
		user.UpdateAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrUserAlreadyExist
		}
		r.logger.Error("create user", zap.String("email", user.Email), zap.Error(err))
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	const q = `
		SELECT id, full_name, email, role, status, password, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1`

	var u User
	err := r.pool.QueryRow(ctx, q, email).Scan(
		&u.ID,
		&u.FullName,
		&u.Email,
		&u.Role,
		&u.Status,
		&u.Password,
		&u.CreateAt,
		&u.UpdateAt,
		&u.DeleteAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		r.logger.Error("get user", zap.String("email", email), zap.Error(err))
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (r *PostgresRepository) GetUsers(ctx context.Context) ([]*User, error) {
	rows, err := r.pool.Query(ctx, "SELECT id,email,full_name,role,status,created_at FROM users")
	if err != nil {
		r.logger.Error("get users", zap.Error(err))
	}
	defer rows.Close()
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*User, error) {
		var u User
		err := row.Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.Status, &u.CreateAt)
		if err != nil {
			return nil, err
		}
		return &u, nil
	})
	return users, err
}

func (r *PostgresRepository) UpdateUser(ctx context.Context, user *User) error {
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET full_name = $2, email = $3, role = $4, status = $5
		WHERE id = $1
		RETURNING id`, user.ID, user.FullName, user.Email, user.Role, user.Status).Scan(&user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		r.logger.Error("update user", zap.String("email", user.Email), zap.Error(err))
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteUser(ctx context.Context, id string) error {
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET deleted_at = now()
		WHERE id = $1
		RETURNING id`, id).Scan(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		r.logger.Error("deleted user", zap.Any("id", id), zap.Error(err))
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `SELECT 
		id, full_name, email, role, status, password, created_at 
		FROM users
		WHERE id=$1`, id).Scan(&u)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		r.logger.Error("get user by id", zap.Any("id", id), zap.Error(err))
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *PostgresRepository) UpdateUserPassword(ctx context.Context, user *User) error {
	_, err := r.pool.Query(ctx, `
		UPDATE users
		SET password = $2, updated_at = now()
		WHERE id = $1`, user.ID, user.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		r.logger.Error("update user password", zap.String("email", user.Email), zap.Error(err))
		return fmt.Errorf("update user password: %w", err)
	}
	return nil
}

func isDuplicateKeyError(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == "23505" || strings.Contains(pgErr.Message, "duplicate key")
	}
	return false
}
