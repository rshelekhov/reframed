package user

import (
	"fmt"
	"github.com/rshelekhov/remedi/internal/lib/api/models"
	"github.com/segmentio/ksuid"
	"time"
)

type Service interface {
	CreateUser(user CreateUser) (string, error)
	GetUser(id string) (GetUser, error)
	GetUsers(models.Pagination) ([]GetUser, error)
	UpdateUser(id string) (string, error)
	DeleteUser(id string) error
}

type userService struct {
	storage Storage
}

// NewService creates a new service
func NewService(storage Storage) Service {
	return &userService{storage}
}

// CreateUser creates a new user
func (s *userService) CreateUser(user CreateUser) (string, error) {
	const op = "user.service.CreateUser"

	id := ksuid.New()

	entity := User{
		ID:        id.String(),
		Email:     user.Email,
		Password:  user.Password,
		RoleID:    user.RoleID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
		UpdatedAt: time.Now().UTC(),
	}

	err := s.storage.CreateUser(entity)
	if err != nil {
		return "", fmt.Errorf("%s: failed to create user: %w", op, err)
	}

	return entity.ID, nil
}

// GetUser returns a user by ID
func (s *userService) GetUser(id string) (GetUser, error) {
	// const op = "user.service.GetUser"
	return s.storage.GetUser(id)
}

// GetUsers returns a list of users
func (s *userService) GetUsers(pgn models.Pagination) ([]GetUser, error) {
	// const op = "user.service.GetUsers"
	return s.storage.GetUsers(pgn)
}

// UpdateUser updates a user by ID
func (s *userService) UpdateUser(id string) (string, error) {
	const op = "user.service.UpdateUser"

	err := s.storage.UpdateUser(id)
	if err != nil {
		return "", fmt.Errorf("%s: failed to update user: %w", op, err)
	}

	return id, nil
}

// DeleteUser deletes a user by ID
func (s *userService) DeleteUser(id string) error {
	// const op = "user.service.DeleteUser"
	return s.storage.DeleteUser(id)
}
