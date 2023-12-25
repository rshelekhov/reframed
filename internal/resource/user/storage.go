package user

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rshelekhov/remedi/internal/storage"
)

// TODO: implement sqlx.DB

type Storage interface {
	ListUsers() ([]User, error)
	CreateUser(user User) error
	ReadUser(id uuid.UUID) (User, error)
	UpdateUser(id uuid.UUID) error
	DeleteUser(id uuid.UUID) error
}

type userStorage struct {
	db *sql.DB
}

// NewStorage creates a new storage
func NewStorage(conn *sql.DB) Storage {
	return &userStorage{db: conn}
}

// ListUsers returns a list of users
func (s *userStorage) ListUsers() ([]User, error) {
	const op = "user.storage.ListUsers"
	users := make([]User, 0)
	return users, nil
}

// CreateUser creates a new user
func (s *userStorage) CreateUser(user User) error {
	const op = "user.storage.CreateUser"

	querySelectRoleID := `SELECT id FROM roles WHERE id = $1`

	queryInsertUser := `INSERT INTO users (id, email, password, role_id, first_name, last_name, phone, created_at, updated_at)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	// Check if role exists
	var roleID int
	err = tx.QueryRow(querySelectRoleID, user.RoleID).Scan(&roleID)
	if err != nil {
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
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == storage.UniqueConstraintViolation {
				return fmt.Errorf("%s: %w", op, storage.ErrUserAlreadyExists)
			}
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}

// ReadUser returns a user by id
func (s *userStorage) ReadUser(id uuid.UUID) (User, error) {
	const op = "user.storage.ReadUser"

	var user User

	return user, nil
}

// UpdateUser updates a user by id
func (s *userStorage) UpdateUser(id uuid.UUID) error {
	const op = "user.storage.UpdateUser"

	return nil
}

// DeleteUser deletes a user by id
func (s *userStorage) DeleteUser(id uuid.UUID) error {
	const op = "user.storage.DeleteUser"

	return nil
}
