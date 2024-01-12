package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rshelekhov/reframed/internal/model"
	"github.com/rshelekhov/reframed/internal/storage"
	"strconv"
)

type UserStorage struct {
	*pgxpool.Pool
}

func NewUserStorage(pg *pgxpool.Pool) *UserStorage {
	return &UserStorage{pg}
}

// CreateUser creates a new user
func (s *UserStorage) CreateUser(ctx context.Context, user model.User) error {
	const op = "user.storage.CreateUser"

	tx, err := BeginTransaction(s.Pool, ctx, op)
	defer func() {
		RollbackOnError(&err, tx, ctx, op)
	}()

	userStatus, err := getUserStatus(ctx, tx, user.Email)
	if err != nil {
		return err
	}

	switch userStatus {
	case "active":
		return fmt.Errorf("%s: user with this email already exists %w", op, storage.ErrUserAlreadyExists)
	case "soft_deleted":
		if err = replaceSoftDeletedUser(ctx, tx, user); err != nil {
			return err
		}
	case "not_found":
		if err = insertUser(ctx, tx, user); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s: unknown user status: %s", op, userStatus)
	}

	CommitTransaction(&err, tx, ctx, op)

	return nil
}

// getUserStatus returns the status of the user with the given email
func getUserStatus(ctx context.Context, tx pgx.Tx, email string) (string, error) {

	const (
		op = "user.storage.getUserStatus"

		query = `SELECT CASE
						WHEN EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL) THEN 'active'
						WHEN EXISTS(SELECT 1 FROM users WHERE email = $1 and deleted_at IS NOT NULL) THEN 'soft_deleted'
						ELSE 'not_found' END AS status`
	)

	var status string

	err := tx.QueryRow(ctx, query, email).Scan(&status)
	if err != nil {
		RollbackOnError(&err, tx, ctx, op)
		return "", fmt.Errorf("%s: failed to check if user exists: %w", op, err)
	}

	return status, nil
}

// replaceSoftDeletedUser replaces a soft deleted user with the given user
func replaceSoftDeletedUser(ctx context.Context, tx pgx.Tx, user model.User) error {
	const (
		op = "user.storage.replaceSoftDeletedUser"

		query = `WITH update_deleted AS (
						UPDATE users SET deleted_at = NULL WHERE email = $1 RETURNING *
					)
					INSERT INTO users
						(id, email, password, updated_at)
						VALUES ($2, $3, $4, $5)`
	)

	_, err := tx.Exec(
		ctx,
		query,
		user.Email,
		user.ID,
		user.Password,
		user.UpdatedAt)
	if err != nil {
		RollbackOnError(&err, tx, ctx, op)
		return fmt.Errorf("%s: failed to replace soft deleted user: %w", op, err)
	}

	return nil
}

// insertUser inserts a new user
func insertUser(ctx context.Context, tx pgx.Tx, user model.User) error {
	const (
		op = "user.storage.insertNewUser"

		query = `INSERT INTO users
    							(id, email, password, updated_at)
								VALUES ($1, $2, $3, $4)`
	)

	_, err := tx.Exec(
		ctx,
		query,
		user.ID,
		user.Email,
		user.Password,
		user.UpdatedAt,
	)
	if err != nil {
		RollbackOnError(&err, tx, ctx, op)
		return fmt.Errorf("%s: failed to insert new user: %w", op, err)
	}

	return nil
}

// GetUser returns a user by ID
func (s *UserStorage) GetUser(ctx context.Context, id string) (model.GetUser, error) {
	const (
		op = "user.storage.GetUser"

		query = `SELECT id, email, updated_at
							FROM users WHERE id = $1 AND deleted_at IS NULL`
	)

	var user model.GetUser

	err := s.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
	}
	if err != nil {
		return user, fmt.Errorf("%s: failed to get user: %w", op, err)
	}

	return user, nil
}

// GetUsers returns a list of users
func (s *UserStorage) GetUsers(ctx context.Context, pgn model.Pagination) ([]*model.GetUser, error) {
	const (
		op = "user.storage.GetUsers"

		query = `SELECT id, email, updated_at
							FROM users WHERE deleted_at IS NULL ORDER BY id DESC LIMIT $1 OFFSET $2`
	)

	rows, err := s.Query(ctx, query, pgn.Limit, pgn.Offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	var users []*model.GetUser
	users, err = pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[model.GetUser])
	if err != nil {
		return nil, fmt.Errorf("%s: failed to collect rows: %w", op, err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("%s: no users found: %w", op, storage.ErrNoUsersFound)
	}

	return users, nil
}

// UpdateUser updates a user by ID
func (s *UserStorage) UpdateUser(ctx context.Context, user model.User) error {
	const op = "user.storage.UpdateUser"

	// Begin transaction
	tx, err := BeginTransaction(s.Pool, ctx, op)
	defer func() {
		RollbackOnError(&err, tx, ctx, op)
	}()

	// Check if the user email exists for a different user
	if err = checkEmailUniqueness(ctx, tx, user.Email, user.ID); err != nil {
		return err
	}

	// Prepare the dynamic update query based on the provided fields
	queryUpdate := "UPDATE users SET updated_at = $1"
	queryParams := []interface{}{user.UpdatedAt}

	if user.Email != "" {
		queryUpdate += ", email = $" + strconv.Itoa(len(queryParams)+1)
		queryParams = append(queryParams, user.Email)
	}
	if user.Password != "" {
		queryUpdate += ", password = $" + strconv.Itoa(len(queryParams)+1)
		queryParams = append(queryParams, user.Password)
	}

	// Add condition for the specific user ID
	queryUpdate += " WHERE id = $" + strconv.Itoa(len(queryParams)+1)
	queryParams = append(queryParams, user.ID)

	// Execute the update query
	_, err = tx.Exec(ctx, queryUpdate, queryParams...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute update query: %w", op, err)
	}

	CommitTransaction(&err, tx, ctx, op)

	return nil
}

// checkEmailUniqueness checks if the provided email already exists in the database for another user
func checkEmailUniqueness(ctx context.Context, tx pgx.Tx, email, id string) error {
	const (
		op = "user.storage.checkEmailUniqueness"

		query = `SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL`
	)

	var existingUserID string

	err := tx.QueryRow(ctx, query, email).Scan(&existingUserID)
	if !errors.Is(err, pgx.ErrNoRows) && existingUserID != id {
		return fmt.Errorf(
			"%s: email already exists in the database for another user: %w", op, storage.ErrUserAlreadyExists,
		)
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: failed to check email uniqueness: %w", op, err)
	}

	return nil
}

// DeleteUser deletes a user by ID
func (s *UserStorage) DeleteUser(ctx context.Context, id string) error {
	const (
		op = "user.storage.DeleteUser"

		query = `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	)

	result, err := s.Exec(ctx, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: user with this id not found %w", op, storage.ErrUserNotFound)
	}
	if err != nil {
		return fmt.Errorf("%s: failed to delete user: %w", op, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("%s: user with ID %s not found: %w", op, id, storage.ErrUserNotFound)
	}

	return nil
}
