package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rshelekhov/reframed/internal/model"
	"github.com/rshelekhov/reframed/internal/port"
	"github.com/rshelekhov/reframed/pkg/constants/le"
)

type TagStorage struct {
	*pgxpool.Pool
	se port.StorageExecutor
}

func NewTagStorage(pool *pgxpool.Pool, executor port.StorageExecutor) *TagStorage {
	return &TagStorage{Pool: pool, se: executor}
}

func (s *TagStorage) ExecSQL(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return s.se.ExecSQL(ctx, sql, arguments...)
}

func (s *TagStorage) CreateTag(ctx context.Context, tag model.Tag) error {
	const (
		op = "tag.repository.CreateTag"

		query = `
			INSERT INTO tags (id, title, user_id, updated_at)
			VALUES ($1, LOWER($2), $3, $4)`
	)

	_, err := s.ExecSQL(
		ctx,
		query,
		tag.ID,
		tag.Title,
		tag.UserID,
		tag.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: failed to insert tag: %w", op, err)
	}

	return nil
}

func (s *TagStorage) LinkTagsToTask(ctx context.Context, taskID string, tags []string) error {
	const (
		op = "tag.repository.LinkTagsToTask"

		query = `
			INSERT INTO tasks_tags (task_id, tag_id)
			VALUES ($1, (SELECT id
			      			FROM tags
			      			WHERE title = LOWER($2))
			)`
	)

	for _, tag := range tags {
		_, err := s.ExecSQL(ctx, query, taskID, tag)
		if err != nil {
			return fmt.Errorf("%s: failed to link tag to task: %w", op, err)
		}
	}

	return nil
}

func (s *TagStorage) UnlinkTagsFromTask(ctx context.Context, taskID string, tags []string) error {
	const (
		op = "tag.repository.UnlinkTagsFromTask"

		query = `
			DELETE FROM tasks_tags
			WHERE task_id = $1 AND tag_id =
			(
				SELECT id
				FROM tags
				WHERE title = lower($2)
			)`
	)

	for _, tag := range tags {
		_, err := s.ExecSQL(ctx, query, taskID, tag)
		if err != nil {
			return fmt.Errorf("%s: failed to unlink tag from task: %w", op, err)
		}
	}

	return nil
}

func (s *TagStorage) GetTagIDByTitle(ctx context.Context, title, userID string) (string, error) {
	const (
		op = "tag.repository.GetTagByTitle"

		query = `
			SELECT id
			FROM tags
			WHERE title = $1
			  AND user_id = $2
			  AND deleted_at IS NULL`
	)

	var tagID string

	err := s.QueryRow(ctx, query, title, userID).Scan(&tagID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", le.ErrTagNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: failed to check if tag exists: %w", op, err)
	}

	return tagID, nil
}

func (s *TagStorage) GetTagsByUserID(ctx context.Context, userID string) ([]model.Tag, error) {
	const (
		op = "tag.repository.GetTagsByUserID"

		query = `
			SELECT id, title, updated_at
			FROM tags
			WHERE user_id = $1
			  AND deleted_at IS NULL`
	)

	rows, err := s.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	var tags []model.Tag

	for rows.Next() {
		tag := model.Tag{}

		err = rows.Scan(
			&tag.ID,
			&tag.Title,
			&tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan tag: %w", op, err)
		}

		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: failed to iterate rows: %w", op, err)
	}

	if len(tags) == 0 {
		return nil, le.ErrNoTagsFound
	}

	return tags, nil
}

func (s *TagStorage) GetTagsByTaskID(ctx context.Context, taskID string) ([]model.Tag, error) {
	const (
		op = "tag.repository.GetTagsByTaskID"

		query = `
			SELECT tags.id, tags.title, tags.updated_at
			FROM tags
				JOIN tasks_tags
				    ON tags.id = tasks_tags.tag_id
			WHERE tasks_tags.task_id = $1
			  AND tags.deleted_at IS NULL`
	)

	rows, err := s.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	var tags []model.Tag

	for rows.Next() {
		tag := model.Tag{}

		err = rows.Scan(&tag)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan tag: %w", op, err)
		}

		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: failed to iterate rows: %w", op, err)
	}

	return tags, nil
}
