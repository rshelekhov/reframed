package service

import (
	"github.com/go-playground/validator"
	"github.com/rshelekhov/reframed/internal/model"
	"github.com/rshelekhov/reframed/internal/storage"
)

type service struct {
	storage  storage.Storage
	validate *validator.Validate
}

// New creates a new service layer
func New(storage storage.Storage, v *validator.Validate) Service {
	return &service{storage, v}
}

// Service is the common interface for all services
type Service interface {
	UserService
}

// UserService defines the user use-cases
type UserService interface {
	CreateUser(user *model.CreateUser) (string, error)
	GetUser(id string) (model.GetUser, error)
	GetUsers(model.Pagination) ([]model.GetUser, error)
	UpdateUser(id string, user *model.UpdateUser) error
	DeleteUser(id string) error
	GetUserRoles() ([]model.GetRole, error)
}
