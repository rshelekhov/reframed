package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rshelekhov/reframed/src/le"
	"github.com/rshelekhov/reframed/src/models"
)

type ListStorage struct {
	*pgxpool.Pool
}

func NewListStorage(pool *pgxpool.Pool) *ListStorage {
	return &ListStorage{Pool: pool}
}

func (s ListStorage) CreateList(ctx context.Context, list models.List) error {
	const (
		op = "list.storage.CreateList"

		// TODO: UPDATE IF EXISTS
		query = `INSERT INTO lists (id, title, user_id, updated_at) VALUES ($1, $2, $3, $4)`
	)

	_, err := s.Exec(ctx, query, list.ID, list.Title, list.UserID, list.UpdatedAt)
	if err != nil {
		return fmt.Errorf("%s: failed to insert new list: %w", op, err)
	}

	return nil
}

func (s ListStorage) GetListByID(ctx context.Context, id string) (models.List, error) {
	//TODO implement me
	panic("implement me")
}

func (s ListStorage) GetLists(ctx context.Context, userID string, pgn models.Pagination) ([]models.List, error) {
	const (
		op = "list.storage.GetLists"

		query = `SELECT id, title, updated_at
					FROM lists WHERE user_id = $1 AND deleted_at IS NULL ORDER BY id DESC LIMIT $2 OFFSET $3`
	)

	rows, err := s.Query(ctx, query, userID, pgn.Limit, pgn.Offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	var lists []models.List

	for rows.Next() {
		list := models.List{}

		err = rows.Scan(&list.ID, &list.Title, &list.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}

		lists = append(lists, list)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: failed to iterate rows: %w", op, err)
	}

	if len(lists) == 0 {
		return nil, le.ErrNoListsFound
	}

	return lists, nil
}

func (s ListStorage) UpdateList(ctx context.Context, list models.List) error {
	//TODO implement me
	panic("implement me")
}

func (s ListStorage) DeleteList(ctx context.Context, id string) error {
	//TODO implement me
	panic("implement me")
}
