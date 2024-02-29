package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rshelekhov/reframed/internal/model"
	"github.com/rshelekhov/reframed/internal/port"
	"github.com/rshelekhov/reframed/pkg/constants/le"
	"strconv"
	"time"
)

type TaskStorage struct {
	*pgxpool.Pool
	*Queries
}

func NewTaskStorage(pool *pgxpool.Pool) port.TaskStorage {
	return &TaskStorage{
		Pool:    pool,
		Queries: New(pool),
	}
}

// TODO: make all storage methods with custom struct instead of default types like this
func (s *TaskStorage) CreateTask(ctx context.Context, task model.Task) error {
	const op = "task.storage.CreateTask"

	taskParams := CreateTaskParams{
		ID:        task.ID,
		Title:     task.Title,
		StatusID:  int32(task.StatusID),
		ListID:    task.ListID,
		HeadingID: task.HeadingID,
		UserID:    task.UserID,
		UpdatedAt: task.UpdatedAt,
	}
	if task.Description != "" {
		taskParams.Description = pgtype.Text{
			String: task.Description,
			Valid:  true,
		}
	}
	if !task.StartDate.IsZero() {
		taskParams.StartDate = pgtype.Timestamptz{
			Time:  task.StartTime,
			Valid: true,
		}
	}
	if !task.Deadline.IsZero() {
		taskParams.Deadline = pgtype.Timestamptz{
			Time:  task.Deadline,
			Valid: true,
		}
	}

	if err := s.Queries.CreateTask(ctx, taskParams); err != nil {
		return fmt.Errorf("%s: failed to insert new task: %w", op, err)
	}
	return nil
}

func (s *TaskStorage) GetTaskStatusID(ctx context.Context, status model.StatusName) (int, error) {
	const op = "task.storage.GetTaskStatusID"

	statusID, err := s.Queries.GetTaskStatusID(ctx, status.String())

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, le.ErrTaskStatusNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get statusID: %w", op, err)
	}

	return int(statusID), nil
}

func (s *TaskStorage) GetTaskByID(ctx context.Context, taskID, userID string) (model.Task, error) {
	const op = "task.storage.GetTaskByID"

	task, err := s.Queries.GetTaskByID(ctx, GetTaskByIDParams{
		ID:     taskID,
		UserID: userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Task{}, le.ErrTaskNotFound
	}
	if err != nil {
		return model.Task{}, fmt.Errorf("%s: failed to get task: %w", op, err)
	}

	taskResp := model.Task{
		ID:        task.ID,
		Title:     task.Title,
		StatusID:  int(task.StatusID),
		ListID:    task.ListID,
		HeadingID: task.HeadingID,
		UpdatedAt: task.UpdatedAt,
		Overdue:   task.Overdue,
	}
	if task.Description.Valid {
		taskResp.Description = task.Description.String
	}
	if task.StartDate.Valid {
		taskResp.StartDate = task.StartDate.Time
	}
	if task.Deadline.Valid {
		taskResp.Deadline = task.Deadline.Time
	}
	if task.StartTime.Valid {
		taskResp.StartTime = task.StartTime.Time
	}
	if task.EndTime.Valid {
		taskResp.EndTime = task.EndTime.Time
	}

	if task.Tags != nil {
		tagsArray, ok := task.Tags.([]interface{})
		if ok {
			tags := make([]string, 0, len(tagsArray))
			for _, tag := range tagsArray {
				if t, ok := tag.(string); ok {
					tags = append(tags, t)
				}
			}
			taskResp.Tags = tags
		}
	}

	return taskResp, nil
}

func (s *TaskStorage) GetTasksByUserID(ctx context.Context, userID string, pgn model.Pagination) ([]model.Task, error) {
	const op = "task.storage.GetTasksByUserID"

	var afterID string
	if pgn.AfterID != "" {
		afterID = pgn.AfterID
	}

	tasks, err := s.Queries.GetTasksByUserID(ctx, GetTasksByUserIDParams{
		UserID:  userID,
		AfterID: afterID,
		Limit:   pgn.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tasks: %w", op, err)
	}

	var tasksResp []model.Task

	for _, task := range tasks {
		t := model.Task{
			ID:        task.ID,
			Title:     task.Title,
			StatusID:  int(task.StatusID),
			ListID:    task.ListID,
			HeadingID: task.HeadingID,
			UpdatedAt: task.UpdatedAt,
			Overdue:   task.Overdue,
		}
		if task.Description.Valid {
			t.Description = task.Description.String
		}
		if task.StartDate.Valid {
			t.StartDate = task.StartDate.Time
		}
		if task.Deadline.Valid {
			t.Deadline = task.Deadline.Time
		}
		if task.StartTime.Valid {
			t.StartTime = task.StartTime.Time
		}
		if task.EndTime.Valid {
			t.EndTime = task.EndTime.Time
		}

		if task.Tags != nil {
			tagsArray, ok := task.Tags.([]interface{})
			if ok {
				tags := make([]string, 0, len(tagsArray))
				for _, tag := range tagsArray {
					if t, ok := tag.(string); ok {
						tags = append(tags, t)
					}
				}
				t.Tags = tags
			}
		}

		tasksResp = append(tasksResp, t)
	}

	if len(tasks) == 0 {
		return nil, le.ErrNoTasksFound
	}

	return tasksResp, nil
}

func (s *TaskStorage) GetTasksByListID(ctx context.Context, listID, userID string) ([]model.Task, error) {
	const op = "task.storage.GetTasksByListID"

	tasks, err := s.Queries.GetTasksByListID(ctx, GetTasksByListIDParams{
		ListID: listID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tasks: %w", op, err)
	}

	var tasksResp []model.Task

	for _, task := range tasks {
		t := model.Task{
			ID:        task.ID,
			Title:     task.Title,
			StatusID:  int(task.StatusID),
			ListID:    task.ListID,
			HeadingID: task.HeadingID,
			UpdatedAt: task.UpdatedAt,
			Overdue:   task.Overdue,
		}
		if task.Description.Valid {
			t.Description = task.Description.String
		}
		if task.StartDate.Valid {
			t.StartDate = task.StartDate.Time
		}
		if task.Deadline.Valid {
			t.Deadline = task.Deadline.Time
		}
		if task.StartTime.Valid {
			t.StartTime = task.StartTime.Time
		}
		if task.EndTime.Valid {
			t.EndTime = task.EndTime.Time
		}

		if task.Tags != nil {
			tagsArray, ok := task.Tags.([]interface{})
			if ok {
				tags := make([]string, 0, len(tagsArray))
				for _, tag := range tagsArray {
					if t, ok := tag.(string); ok {
						tags = append(tags, t)
					}
				}
				t.Tags = tags
			}
		}

		tasksResp = append(tasksResp, t)
	}

	if len(tasks) == 0 {
		return nil, le.ErrNoTasksFound
	}

	return tasksResp, nil
}

func (s *TaskStorage) GetTasksGroupedByHeadings(ctx context.Context, listID, userID string) ([]model.TaskGroup, error) {
	const op = "task.storage.GetTasksGroupedByHeadings"

	groups, err := s.Queries.GetTasksGroupedByHeadings(ctx, GetTasksGroupedByHeadingsParams{
		ListID: listID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tasks groups: %w", op, err)
	}
	if len(groups) == 0 {
		return nil, le.ErrNoTasksFound
	}

	var taskGroups []model.TaskGroup

	for _, group := range groups {
		var taskGroup model.TaskGroup
		var tasks []model.TaskResponseData

		err = json.Unmarshal(group.Tasks, &tasks)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal tasks from postgres json object: %w", op, err)
		}

		taskGroup.HeadingID = group.HeadingID
		taskGroup.Tasks = tasks

		taskGroups = append(taskGroups, taskGroup)
	}

	return taskGroups, nil
}

func (s *TaskStorage) GetTasksForToday(ctx context.Context, userID string) ([]model.TaskGroup, error) {
	const op = "task.storage.GetTasksForToday"

	groups, err := s.Queries.GetTasksForToday(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tasks groups: %w", op, err)
	}
	if len(groups) == 0 {
		return nil, le.ErrNoTasksFound
	}

	var taskGroups []model.TaskGroup

	for _, group := range groups {
		var taskGroup model.TaskGroup
		var tasks []model.TaskResponseData

		err = json.Unmarshal(group.Tasks, &tasks)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal tasks from postgres json object: %w", op, err)
		}

		taskGroup.ListID = group.ListID
		taskGroup.Tasks = tasks

		taskGroups = append(taskGroups, taskGroup)
	}

	return taskGroups, nil
}

func (s *TaskStorage) GetUpcomingTasks(ctx context.Context, userID string, pgn model.Pagination) ([]model.TaskGroup, error) {
	const op = "task.storage.GetUpcomingTasks"

	var afterDate time.Time

	if pgn.AfterDate.IsZero() {
		afterDate = time.Now()
	} else {
		afterDate = pgn.AfterDate
	}

	groups, err := s.Queries.GetUpcomingTasks(ctx, GetUpcomingTasksParams{
		UserID: userID,
		AfterDate: pgtype.Timestamptz{
			Valid: true,
			Time:  afterDate,
		},
		Limit: pgn.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tasks groups: %w", op, err)
	}
	if len(groups) == 0 {
		return nil, le.ErrNoTasksFound
	}

	var taskGroups []model.TaskGroup

	for _, group := range groups {
		var taskGroup model.TaskGroup
		var tasks []model.TaskResponseData

		err = json.Unmarshal(group.Tasks, &tasks)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal tasks from postgres json object: %w", op, err)
		}

		taskGroup.StartDate = group.StartDate.Time
		taskGroup.Tasks = tasks

		taskGroups = append(taskGroups, taskGroup)
	}

	return taskGroups, nil
}

func (s *TaskStorage) GetOverdueTasks(ctx context.Context, userID string, pgn model.Pagination) ([]model.TaskGroup, error) {
	const op = "task.storage.GetOverdueTasks"

	groups, err := s.Queries.GetOverdueTasks(ctx, GetOverdueTasksParams{
		UserID:  userID,
		Limit:   pgn.Limit,
		AfterID: pgn.AfterID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tasks groups: %w", op, err)
	}
	if len(groups) == 0 {
		return nil, le.ErrNoTasksFound
	}

	var taskGroups []model.TaskGroup

	for _, group := range groups {
		var taskGroup model.TaskGroup
		var tasks []model.TaskResponseData

		err = json.Unmarshal(group.Tasks, &tasks)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal tasks from postgres json object: %w", op, err)
		}

		taskGroup.ListID = group.ListID
		taskGroup.Tasks = tasks

		taskGroups = append(taskGroups, taskGroup)
	}

	return taskGroups, nil
}

func (s *TaskStorage) GetTasksForSomeday(ctx context.Context, userID string, pgn model.Pagination) ([]model.TaskGroup, error) {
	const op = "task.storage.GetTasksForSomeday"

	var afterDate time.Time

	if pgn.AfterDate.IsZero() {
		afterDate = time.Now()
	} else {
		afterDate = pgn.AfterDate
	}

	groups, err := s.Queries.GetUpcomingTasks(ctx, GetUpcomingTasksParams{
		UserID: userID,
		AfterDate: pgtype.Timestamptz{
			Valid: true,
			Time:  afterDate,
		},
		Limit: pgn.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tasks groups: %w", op, err)
	}
	if len(groups) == 0 {
		return nil, le.ErrNoTasksFound
	}

	var taskGroups []model.TaskGroup

	for _, group := range groups {
		var taskGroup model.TaskGroup
		var tasks []model.TaskResponseData

		err = json.Unmarshal(group.Tasks, &tasks)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal tasks from postgres json object: %w", op, err)
		}

		if group.StartDate.Valid {
			taskGroup.StartDate = group.StartDate.Time
		}
		taskGroup.Tasks = tasks

		taskGroups = append(taskGroups, taskGroup)
	}

	return taskGroups, nil
}

func (s *TaskStorage) GetCompletedTasks(ctx context.Context, userID string, pgn model.Pagination) ([]model.TaskGroup, error) {
	const op = "task.storage.GetCompletedTasks"

	var afterDate time.Time

	if pgn.AfterDate.IsZero() {
		afterDate = time.Now()
	} else {
		afterDate = pgn.AfterDate
	}

	groups, err := s.Queries.GetCompletedTasks(ctx, GetCompletedTasksParams{
		UserID:      userID,
		Limit:       pgn.Limit,
		StatusTitle: model.StatusCompleted.String(),
		AfterDate: pgtype.Timestamptz{
			Valid: true,
			Time:  afterDate,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tasks groups: %w", op, err)
	}
	if len(groups) == 0 {
		return nil, le.ErrNoTasksFound
	}

	var taskGroups []model.TaskGroup

	for _, group := range groups {
		var taskGroup model.TaskGroup
		var tasks []model.TaskResponseData

		err = json.Unmarshal(group.Tasks, &tasks)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal tasks from postgres json object: %w", op, err)
		}

		// TODO: check the response for this field
		if group.Month.Valid {
			taskGroup.Month = group.Month.Months
		}
		taskGroup.Tasks = tasks

		taskGroups = append(taskGroups, taskGroup)
	}

	return taskGroups, nil
}

func (s *TaskStorage) GetArchivedTasks(ctx context.Context, userID string, pgn model.Pagination) ([]model.TaskGroup, error) {
	const op = "task.storage.GetArchivedTasks"

	var afterMonth time.Time

	if pgn.AfterDate.IsZero() {
		afterMonth = time.Now().Truncate(24 * time.Hour)
	} else {
		afterMonth = pgn.AfterDate
	}

	groups, err := s.Queries.GetArchivedTasks(ctx, GetArchivedTasksParams{
		UserID:      userID,
		Limit:       pgn.Limit,
		StatusTitle: model.StatusArchived.String(),
		AfterMonth: pgtype.Timestamptz{
			Valid: true,
			Time:  afterMonth,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tasks groups: %w", op, err)
	}
	if len(groups) == 0 {
		return nil, le.ErrNoTasksFound
	}

	var taskGroups []model.TaskGroup

	for _, group := range groups {
		var taskGroup model.TaskGroup
		var tasks []model.TaskResponseData

		err = json.Unmarshal(group.Tasks, &tasks)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal tasks from postgres json object: %w", op, err)
		}

		// TODO: check the response for this field
		if group.Month.Valid {
			taskGroup.Month = group.Month.Months
		}
		taskGroup.Tasks = tasks

		taskGroups = append(taskGroups, taskGroup)
	}
	return taskGroups, nil
}

func (s *TaskStorage) UpdateTask(ctx context.Context, task model.Task) error {
	const (
		op = "task.storage.UpdateTask"

		queryGetHeadingID = `
			SELECT heading_id
			FROM tasks
			WHERE id = $1
			  AND user_id = $2`
	)

	var headingID string

	err := s.QueryRow(ctx, queryGetHeadingID, task.ID, task.UserID).Scan(&headingID)
	if err != nil {
		return fmt.Errorf("%s: failed to get heading ID: %w", op, err)
	}

	// Prepare the dynamic update query based on the provided fields
	queryUpdate := "UPDATE tasks SET updated_at = $1"
	queryParams := []interface{}{task.UpdatedAt}

	// Add fields to the query
	if task.Title != "" {
		queryUpdate += ", title = $" + strconv.Itoa(len(queryParams)+1)
		queryParams = append(queryParams, task.Title)
	}
	if task.Description != "" {
		queryUpdate += ", description = $" + strconv.Itoa(len(queryParams)+1)
		queryParams = append(queryParams, task.Description)
	}
	if !task.StartDate.IsZero() {
		queryUpdate += ", start_date = $" + strconv.Itoa(len(queryParams)+1)
		queryParams = append(queryParams, task.StartDate)
	}
	if !task.Deadline.IsZero() {
		queryUpdate += ", deadline = $" + strconv.Itoa(len(queryParams)+1)
		queryParams = append(queryParams, task.Deadline)
	}
	if task.HeadingID != headingID {
		queryUpdate += ", heading_id = $" + strconv.Itoa(len(queryParams)+1)
		queryParams = append(queryParams, task.HeadingID)
	}

	// Add condition for the specific user ID
	queryUpdate += " WHERE id = $" + strconv.Itoa(len(queryParams)+1)
	queryParams = append(queryParams, task.ID)

	queryUpdate += " AND user_id = $" + strconv.Itoa(len(queryParams)+1)
	queryParams = append(queryParams, task.UserID)

	// Execute the update query
	result, err := s.Exec(ctx, queryUpdate, queryParams...)
	if err != nil {
		return fmt.Errorf("%s: failed to update task: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		return le.ErrTaskNotFound
	}

	return nil
}

func (s *TaskStorage) UpdateTaskTime(ctx context.Context, task model.Task) error {
	const op = "task.storage.UpdateTaskTime"

	// Get the statusID ID for the planned status
	var statusID string

	// Prepare the dynamic update query based on the provided fields
	queryUpdate := "UPDATE tasks SET updated_at = $1"
	queryParams := []interface{}{task.UpdatedAt}

	// Add time fields to the query
	if !task.StartTime.IsZero() && !task.EndTime.IsZero() {
		queryUpdate += ", start_time = $" + strconv.Itoa(len(queryParams)+1) + ", end_time = $" + strconv.Itoa(len(queryParams)+2)
		queryParams = append(queryParams, task.StartTime, task.EndTime)
	} else if task.StartTime.IsZero() && task.EndTime.IsZero() {
		queryUpdate += ", start_time = NULL, end_time = NULL"
	} else {
		return le.ErrInvalidTaskTimeRange
	}

	// Add statusID ID to the query
	queryUpdate += ", status_id = $" + strconv.Itoa(len(queryParams)+1)
	queryParams = append(queryParams, statusID)

	// Add condition for the specific user ID
	queryUpdate += " WHERE id = $" + strconv.Itoa(len(queryParams)+1)
	queryParams = append(queryParams, task.ID)

	queryUpdate += " AND user_id = $" + strconv.Itoa(len(queryParams)+1)
	queryParams = append(queryParams, task.UserID)

	// Execute the update query
	result, err := s.Exec(ctx, queryUpdate, queryParams...)
	if err != nil {
		return fmt.Errorf("%s: failed to update task: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		return le.ErrTaskNotFound
	}

	return nil
}

func (s *TaskStorage) MoveTaskToAnotherList(ctx context.Context, task model.Task) error {
	const op = "task.storage.MoveTaskToAnotherList"

	if err := s.Queries.MoveTaskToAnotherList(ctx, MoveTaskToAnotherListParams{
		ListID:    task.ListID,
		HeadingID: task.HeadingID,
		UpdatedAt: task.UpdatedAt,
		ID:        task.ID,
		UserID:    task.UserID,
	}); err != nil {
		return fmt.Errorf("%s: failed to move task: %w", op, err)
	}
	return nil
}

func (s *TaskStorage) MarkAsCompleted(ctx context.Context, task model.Task) error {
	const op = "task.storage.MarkAsCompleted"

	if err := s.Queries.MarkTaskAsCompleted(ctx, MarkTaskAsCompletedParams{
		StatusID:  int32(task.StatusID),
		UpdatedAt: task.UpdatedAt,
		ID:        task.ID,
		UserID:    task.UserID,
	}); err != nil {
		return fmt.Errorf("%s: failed to update task: %w", op, err)
	}
	return nil
}

func (s *TaskStorage) MarkAsArchived(ctx context.Context, task model.Task) error {
	const op = "task.storage.MarkAsArchived"

	if err := s.Queries.MarkTaskAsArchived(ctx, MarkTaskAsArchivedParams{
		StatusID: int32(task.StatusID),
		DeletedAt: pgtype.Timestamptz{
			Valid: true,
			Time:  task.DeletedAt,
		},
		ID:     task.ID,
		UserID: task.UserID,
	}); err != nil {
		return fmt.Errorf("%s: failed to update task: %w", op, err)
	}
	return nil
}
