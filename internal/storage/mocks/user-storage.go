package mocks

import (
	"context"
	"github.com/rshelekhov/reframed/internal/models"
	"github.com/stretchr/testify/mock"
)

type UserStorage struct {
	mock.Mock
}

func (u *UserStorage) CreateUser(ctx context.Context, user models.User) error {
	args := u.Called(ctx, user)
	return args.Error(0)
}

func (u *UserStorage) GetUserByID(ctx context.Context, id string) (models.User, error) {
	args := u.Called(ctx, id)
	result := args.Get(0)
	return result.(models.User), args.Error(1)
}

func (u *UserStorage) GetUsers(ctx context.Context, pgn models.Pagination) ([]models.User, error) {
	args := u.Called(ctx, pgn)
	return args.Get(0).([]models.User), args.Error(1)
}

func (u *UserStorage) UpdateUser(ctx context.Context, user models.User) error {
	//TODO implement me
	panic("implement me")
}

func (u *UserStorage) DeleteUser(ctx context.Context, id string) error {
	//TODO implement me
	panic("implement me")
}
