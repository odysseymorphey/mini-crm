package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
	"mini-crm/internal/domain"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (email, password_hash, name, created_at) VALUES (?, ?, ?, NOW())`
	res, err := r.db.ExecContext(ctx, query, user.Email, user.PasswordHash, user.Name)
	if err != nil {
		var mySQLErr *mysql.MySQLError
		if errors.As(err, &mySQLErr) && mySQLErr.Number == 1062 {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, created_at FROM users WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var u domain.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, created_at FROM users WHERE email = ?`
	row := r.db.QueryRowContext(ctx, query, email)

	var u domain.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
