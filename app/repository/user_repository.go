package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"pemrograman-code/app/model"
)

// UserRepository adalah kontrak akses data user. Service dan middleware
// bergantung pada interface ini, bukan pada struct PostgreSQL, sehingga test
// dapat menggantinya dengan fake (Modul 12, A.4).
type UserRepository interface {
	Create(ctx context.Context, u model.User) (model.User, error)
	FindByID(ctx context.Context, id int) (model.User, error)
	FindByUsername(ctx context.Context, username string) (model.User, error)
}

type userRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, u model.User) (model.User, error) {
	// User baru selalu diberi role viewer (default paling rendah hak aksesnya).
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role_id, is_active)
		 VALUES ($1, $2, $3, (SELECT id FROM roles WHERE name = 'viewer'), $4)
		 RETURNING id, role_id, created_at`,
		u.Username, u.Email, u.Password, u.IsActive,
	).Scan(&u.ID, &u.RoleID, &u.CreatedAt)
	if isUniqueViolation(err) {
		return model.User{}, ErrDuplicate
	}
	if err != nil {
		return model.User{}, fmt.Errorf("menyimpan user: %w", err)
	}
	u.Role = "viewer"
	return u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id int) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.username, u.email, u.password, u.role_id, r.name, u.is_active, u.created_at
		 FROM users u JOIN roles r ON r.id = u.role_id WHERE u.id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &u.Role, &u.IsActive, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}
	return u, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.username, u.email, u.password, u.role_id, r.name, u.is_active, u.created_at
		 FROM users u JOIN roles r ON r.id = u.role_id
		 WHERE LOWER(u.username) = LOWER($1)`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &u.Role, &u.IsActive, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}
	return u, nil
}
