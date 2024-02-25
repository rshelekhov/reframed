// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: list.sql

package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createList = `-- name: CreateList :exec
INSERT INTO lists (id, title, user_id, updated_at)
VALUES ($1, $2, $3, $4)
`

type CreateListParams struct {
	ID        string             `db:"id"`
	Title     string             `db:"title"`
	UserID    string             `db:"user_id"`
	UpdatedAt pgtype.Timestamptz `db:"updated_at"`
}

func (q *Queries) CreateList(ctx context.Context, arg CreateListParams) error {
	_, err := q.db.Exec(ctx, createList,
		arg.ID,
		arg.Title,
		arg.UserID,
		arg.UpdatedAt,
	)
	return err
}

const deleteList = `-- name: DeleteList :exec
UPDATE lists
SET deleted_at = $1
WHERE id = $2
  AND user_id = $3
`

type DeleteListParams struct {
	DeletedAt pgtype.Timestamptz `db:"deleted_at"`
	ID        string             `db:"id"`
	UserID    string             `db:"user_id"`
}

func (q *Queries) DeleteList(ctx context.Context, arg DeleteListParams) error {
	_, err := q.db.Exec(ctx, deleteList, arg.DeletedAt, arg.ID, arg.UserID)
	return err
}

const getListByID = `-- name: GetListByID :one
SELECT id, title, user_id, updated_at
FROM lists
WHERE id = $1
  AND user_id = $2
  AND deleted_at IS NULL
`

type GetListByIDParams struct {
	ID     string `db:"id"`
	UserID string `db:"user_id"`
}

type GetListByIDRow struct {
	ID        string             `db:"id"`
	Title     string             `db:"title"`
	UserID    string             `db:"user_id"`
	UpdatedAt pgtype.Timestamptz `db:"updated_at"`
}

func (q *Queries) GetListByID(ctx context.Context, arg GetListByIDParams) (GetListByIDRow, error) {
	row := q.db.QueryRow(ctx, getListByID, arg.ID, arg.UserID)
	var i GetListByIDRow
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.UserID,
		&i.UpdatedAt,
	)
	return i, err
}

const getListsByUserID = `-- name: GetListsByUserID :many
SELECT id, title, updated_at
FROM lists
WHERE user_id = $1
  AND deleted_at IS NULL
ORDER BY id
`

type GetListsByUserIDRow struct {
	ID        string             `db:"id"`
	Title     string             `db:"title"`
	UpdatedAt pgtype.Timestamptz `db:"updated_at"`
}

func (q *Queries) GetListsByUserID(ctx context.Context, userID string) ([]GetListsByUserIDRow, error) {
	rows, err := q.db.Query(ctx, getListsByUserID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetListsByUserIDRow{}
	for rows.Next() {
		var i GetListsByUserIDRow
		if err := rows.Scan(&i.ID, &i.Title, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateList = `-- name: UpdateList :exec
UPDATE lists
SET title = $1,	updated_at = $2
WHERE id = $3
  AND user_id = $4
`

type UpdateListParams struct {
	Title     string             `db:"title"`
	UpdatedAt pgtype.Timestamptz `db:"updated_at"`
	ID        string             `db:"id"`
	UserID    string             `db:"user_id"`
}

func (q *Queries) UpdateList(ctx context.Context, arg UpdateListParams) error {
	_, err := q.db.Exec(ctx, updateList,
		arg.Title,
		arg.UpdatedAt,
		arg.ID,
		arg.UserID,
	)
	return err
}