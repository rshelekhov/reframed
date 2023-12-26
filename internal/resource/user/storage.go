package user

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/rshelekhov/remedi/internal/lib/api/models"
	"github.com/rshelekhov/remedi/internal/storage"
)

type Storage interface {
	CreateUser(user User) error
	GetUser(id string) (GetUser, error)
	GetUsers(models.Pagination) ([]GetUser, error)
	UpdateUser(id string) error
	DeleteUser(id string) error
}

type userStorage struct {
	db *sqlx.DB
}

// NewStorage creates a new storage
func NewStorage(conn *sqlx.DB) Storage {
	return &userStorage{db: conn}
}

// CreateUser creates a new user
func (s *userStorage) CreateUser(user User) error {
	const op = "user.storage.CreateUser"

	querySelectRoleID := `SELECT id FROM roles WHERE id = $1`

	queryInsertUser := `INSERT INTO users (id, email, password, role_id, first_name, last_name, phone, updated_at)
							VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer func(tx *sql.Tx) {
		err := tx.Rollback()
		if err != nil {
			return
		}
	}(tx)

	// Check if role exists
	var roleID int
	err = tx.QueryRow(querySelectRoleID, user.RoleID).Scan(&roleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%s: role not found: %w", op, storage.ErrRoleNotFound)
		}
		return fmt.Errorf("%s: failed to check if role exists: %w", op, err)
	}

	// Insert user
	_, err = tx.Exec(
		queryInsertUser,
		user.ID,
		user.Email,
		user.Password,
		roleID,
		user.FirstName,
		user.LastName,
		user.Phone,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == storage.UniqueConstraintViolation {
				return fmt.Errorf("%s: %w", op, storage.ErrUserAlreadyExists)
			}
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}

// GetUser returns a user by ID
func (s *userStorage) GetUser(id string) (GetUser, error) {
	const op = "user.storage.ReadUser"

	var user GetUser
	querySelectUser := `SELECT id, email, role_id, first_name, last_name, phone, updated_at
							FROM users WHERE id = $1 AND deleted_at IS NULL`

	err := s.db.Get(&user, querySelectUser, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetUser{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return GetUser{}, fmt.Errorf("%s: failed to get user: %w", op, err)
	}

	return user, nil
}

// GetUsers returns a list of users
func (s *userStorage) GetUsers(pgn models.Pagination) ([]GetUser, error) {
	const op = "user.storage.GetUsers"

	var users []GetUser
	querySelectUsers := `SELECT id, email, role_id, first_name, last_name, phone, updated_at
							FROM users WHERE deleted_at IS NULL ORDER BY id DESC LIMIT $1 OFFSET $2`

	err := s.db.Select(&users, querySelectUsers, pgn.Limit, pgn.Offset)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: no users found: %w", op, storage.ErrNoUsersFound)
		}
		return nil, fmt.Errorf("%s: failed to get users: %w", op, err)
	}

	return users, nil
}

// UpdateUser updates a user by ID
func (s *userStorage) UpdateUser(id string) error {
	const op = "user.storage.UpdateUser"

	return nil
}

// DeleteUser deletes a user by ID
func (s *userStorage) DeleteUser(id string) error {
	const op = "user.storage.DeleteUser"

	return nil
}
