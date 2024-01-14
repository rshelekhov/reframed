package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/rshelekhov/reframed/internal/logger"
	"github.com/rshelekhov/reframed/internal/model"
	"github.com/rshelekhov/reframed/internal/storage"
	"github.com/segmentio/ksuid"
	"log/slog"
	"net/http"
	"time"
)

type UserHandler struct {
	Storage storage.UserStorage
	Logger  logger.Interface
}

//go:generate go run github.com/vektra/mockery/v2@v2.40.1 --name=UserCreater
type UserCreater interface {
	CreateUser(ctx context.Context, user *model.User) (string, error)
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.CreateUser"

		log := logger.LogWithRequest(h.Logger, op, r)

		// user := &model.CreateUser{}
		user := &model.User{}

		// Decode the request body
		err := decodeJSON(w, r, log, user)
		if err != nil {
			return
		}

		// Validate the request
		err = validateData(w, r, log, user)
		if err != nil {
			return
		}

		id := ksuid.New().String()
		now := time.Now().UTC()

		newUser := model.User{
			ID:        id,
			Email:     user.Email,
			Password:  user.Password,
			UpdatedAt: &now,
		}

		// Create the user
		err = h.Storage.CreateUser(r.Context(), newUser)
		if errors.Is(err, storage.ErrUserAlreadyExists) {
			log.Error(fmt.Sprintf("%v", storage.ErrUserAlreadyExists), slog.String("email", *user.Email))
			responseError(w, r, http.StatusConflict, fmt.Sprintf("%v", storage.ErrUserAlreadyExists))
			return
		}
		if err != nil {
			log.Error("failed to create user", logger.Err(err))
			responseError(w, r, http.StatusInternalServerError, "failed to create user")
			return
		}

		log.Info("UserUsecase created", slog.Any("user_id", id))
		responseSuccess(w, r, http.StatusCreated, "user created", model.User{ID: id})
	}
}

//go:generate go run github.com/vektra/mockery/v2@v2.40.1 --name=UserIDGetter
type UserIDGetter interface {
	GetUserByID(ctx context.Context, id string) (model.User, error)
}

// GetUserByID get a user by ID
func (h *UserHandler) GetUserByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.GetUserByID"

		log := logger.LogWithRequest(h.Logger, op, r)

		id, err := GetID(w, r, log)
		if err != nil {
			return
		}

		user, err := h.Storage.GetUserByID(r.Context(), id)
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Error(fmt.Sprintf("%v", storage.ErrUserNotFound), slog.String("user_id", id))
			responseError(w, r, http.StatusNotFound, fmt.Sprintf("%v", storage.ErrUserNotFound))
			return
		}
		if err != nil {
			log.Error("failed to get user", logger.Err(err))
			responseError(w, r, http.StatusInternalServerError, "failed to get user")
			return
		}

		log.Info("UserUsecase received", slog.Any("user", user))
		responseSuccess(w, r, http.StatusOK, "user received", user)
	}
}

//go:generate go run github.com/vektra/mockery/v2@v2.40.1 --name=UsersGetter
type UsersGetter interface {
	GetUsers(ctx context.Context, pgn model.Pagination) ([]model.User, error)
}

// GetUsers get a list of users
func (h *UserHandler) GetUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.GetUsers"

		log := logger.LogWithRequest(h.Logger, op, r)

		pagination, err := parseLimitAndOffset(r)
		if err != nil {
			log.Error("failed to parse limit and offset", logger.Err(err))
			responseError(w, r, http.StatusBadRequest, "failed to parse limit and offset")
			return
		}

		users, err := h.Storage.GetUsers(r.Context(), pagination)
		if errors.Is(err, storage.ErrNoUsersFound) {
			log.Error(fmt.Sprintf("%v", storage.ErrNoUsersFound))
			responseError(w, r, http.StatusNotFound, fmt.Sprintf("%v", storage.ErrNoUsersFound))
			return
		}
		if err != nil {
			log.Error("failed to get users", logger.Err(err))
			responseError(w, r, http.StatusInternalServerError, "failed to get users")
			return
		}

		log.Info(
			"users found",
			slog.Int("count", len(users)),
			slog.Int("limit", pagination.Limit),
			slog.Int("offset", pagination.Offset),
		)

		responseSuccess(w, r, http.StatusOK, "users found", users)
	}
}

//go:generate go run github.com/vektra/mockery/v2@v2.40.1 --name=UserUpdater
type UserUpdater interface {
	UpdateUser(ctx context.Context, id string, user *model.UpdateUser) error
}

// UpdateUser updates a user by ID
func (h *UserHandler) UpdateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.UpdateUser"

		log := logger.LogWithRequest(h.Logger, op, r)

		user := &model.UpdateUser{}

		id, err := GetID(w, r, log)
		if err != nil {
			return
		}

		// Decode the request body
		err = decodeJSON(w, r, log, user)
		if err != nil {
			return
		}

		// Validate the request
		err = validateData(w, r, log, user)
		if err != nil {
			return
		}

		now := time.Now().UTC()

		updatedUser := model.User{
			ID:        id,
			Email:     &user.Email,
			Password:  &user.Password,
			UpdatedAt: &now,
		}

		err = h.Storage.UpdateUser(r.Context(), updatedUser)
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Error(fmt.Sprintf("%v", storage.ErrUserNotFound), slog.String("user_id", id))
			responseError(w, r, http.StatusNotFound, fmt.Sprintf("%v", storage.ErrUserNotFound))
			return
		}
		if errors.Is(err, storage.ErrEmailAlreadyTaken) {
			log.Error(fmt.Sprintf("%v", storage.ErrEmailAlreadyTaken), slog.String("email", user.Email))
			responseError(w, r, http.StatusConflict, fmt.Sprintf("%v", storage.ErrEmailAlreadyTaken))
			return
		}
		if errors.Is(err, storage.ErrNoChangesDetected) {
			log.Error(fmt.Sprintf("%v", storage.ErrNoChangesDetected), slog.String("user_id", id))
			responseError(w, r, http.StatusBadRequest, fmt.Sprintf("%v", storage.ErrNoChangesDetected))
			return
		}
		if errors.Is(err, storage.ErrNoPasswordChangesDetected) {
			log.Error(fmt.Sprintf("%v", storage.ErrNoPasswordChangesDetected), slog.String("user_id", id))
			responseError(w, r, http.StatusBadRequest, fmt.Sprintf("%v", storage.ErrNoPasswordChangesDetected))
			return
		}
		if err != nil {
			log.Error("failed to update user", logger.Err(err))
			responseError(w, r, http.StatusInternalServerError, "failed to update user")
			return
		}

		log.Info("UserUsecase updated", slog.String("user_id", id))
		responseSuccess(w, r, http.StatusOK, "user updated", model.User{ID: id})
	}
}

//go:generate go run github.com/vektra/mockery/v2@v2.40.1 --name=UserDeleter
type UserDeleter interface {
	DeleteUser(ctx context.Context, id string) error
}

// DeleteUser deletes a user by ID
func (h *UserHandler) DeleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.DeleteUser"

		log := logger.LogWithRequest(h.Logger, op, r)

		id, err := GetID(w, r, log)
		if err != nil {
			return
		}

		err = h.Storage.DeleteUser(r.Context(), id)
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Error(fmt.Sprintf("%v", storage.ErrUserNotFound), slog.String("user_id", id))
			responseError(w, r, http.StatusNotFound, fmt.Sprintf("%v", storage.ErrUserNotFound))
			return
		}
		if err != nil {
			log.Error("failed to delete user", logger.Err(err))
			responseError(w, r, http.StatusInternalServerError, "failed to delete user")
			return
		}

		log.Info("user deleted", slog.String("user_id", id))

		responseSuccess(w, r, http.StatusOK, "user deleted", model.User{ID: id})
	}
}
