package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// pgErrorCode returns the Postgres SQLSTATE code if err wraps one, otherwise "".
func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// ---------- users ----------

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, email, role, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) GetUser(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, name, email, role, created_at FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) CreateUser(ctx context.Context, name, email, role string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (name, email, role) VALUES ($1, $2, $3)
		 RETURNING id, name, email, role, created_at`,
		name, email, role).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) UpdateUser(ctx context.Context, id int64, name, email, role *string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`UPDATE users SET
			name = COALESCE($1, name),
			email = COALESCE($2, email),
			role = COALESCE($3, role)
		 WHERE id = $4
		 RETURNING id, name, email, role, created_at`,
		name, email, role, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- projects ----------

func (s *Store) ListProjects(ctx context.Context, status string) ([]Project, error) {
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = s.db.Query(ctx,
			`SELECT id, title, description, status, owner_id, budget, created_at
			 FROM projects ORDER BY id`)
	} else {
		rows, err = s.db.Query(ctx,
			`SELECT id, title, description, status, owner_id, budget, created_at
			 FROM projects WHERE status = $1 ORDER BY id`, status)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []Project{}
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Status, &p.OwnerID, &p.Budget, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *Store) GetProject(ctx context.Context, id int64) (Project, error) {
	var p Project
	err := s.db.QueryRow(ctx,
		`SELECT id, title, description, status, owner_id, budget, created_at
		 FROM projects WHERE id = $1`, id).
		Scan(&p.ID, &p.Title, &p.Description, &p.Status, &p.OwnerID, &p.Budget, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	if err != nil {
		return Project{}, err
	}
	return p, nil
}

func (s *Store) CreateProject(ctx context.Context, title string, description *string, ownerID int64, budget *float64) (Project, error) {
	var p Project
	err := s.db.QueryRow(ctx,
		`INSERT INTO projects (title, description, owner_id, budget)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, title, description, status, owner_id, budget, created_at`,
		title, description, ownerID, budget).
		Scan(&p.ID, &p.Title, &p.Description, &p.Status, &p.OwnerID, &p.Budget, &p.CreatedAt)
	if err != nil {
		return Project{}, err
	}
	return p, nil
}

func (s *Store) UpdateProject(ctx context.Context, id int64, title, description *string, status *string, ownerID *int64, budget *float64) (Project, error) {
	var p Project
	err := s.db.QueryRow(ctx,
		`UPDATE projects SET
			title = COALESCE($1, title),
			description = COALESCE($2, description),
			status = COALESCE($3, status),
			owner_id = COALESCE($4, owner_id),
			budget = COALESCE($5, budget)
		 WHERE id = $6
		 RETURNING id, title, description, status, owner_id, budget, created_at`,
		title, description, status, ownerID, budget, id).
		Scan(&p.ID, &p.Title, &p.Description, &p.Status, &p.OwnerID, &p.Budget, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	if err != nil {
		return Project{}, err
	}
	return p, nil
}

func (s *Store) DeleteProject(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- launches ----------

func (s *Store) ListLaunches(ctx context.Context, projectID int64) ([]Launch, error) {
	var rows pgx.Rows
	var err error
	if projectID == 0 {
		rows, err = s.db.Query(ctx,
			`SELECT id, project_id, title, status, start_date, end_date, created_at
			 FROM launches ORDER BY id`)
	} else {
		rows, err = s.db.Query(ctx,
			`SELECT id, project_id, title, status, start_date, end_date, created_at
			 FROM launches WHERE project_id = $1 ORDER BY id`, projectID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	launches := []Launch{}
	for rows.Next() {
		var l Launch
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Title, &l.Status, &l.StartDate, &l.EndDate, &l.CreatedAt); err != nil {
			return nil, err
		}
		launches = append(launches, l)
	}
	return launches, rows.Err()
}

func (s *Store) GetLaunch(ctx context.Context, id int64) (Launch, error) {
	var l Launch
	err := s.db.QueryRow(ctx,
		`SELECT id, project_id, title, status, start_date, end_date, created_at
		 FROM launches WHERE id = $1`, id).
		Scan(&l.ID, &l.ProjectID, &l.Title, &l.Status, &l.StartDate, &l.EndDate, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Launch{}, ErrNotFound
	}
	if err != nil {
		return Launch{}, err
	}
	return l, nil
}

func (s *Store) CreateLaunch(ctx context.Context, projectID int64, title string, startDate, endDate *time.Time) (Launch, error) {
	var l Launch
	err := s.db.QueryRow(ctx,
		`INSERT INTO launches (project_id, title, start_date, end_date)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, project_id, title, status, start_date, end_date, created_at`,
		projectID, title, startDate, endDate).
		Scan(&l.ID, &l.ProjectID, &l.Title, &l.Status, &l.StartDate, &l.EndDate, &l.CreatedAt)
	if err != nil {
		return Launch{}, err
	}
	return l, nil
}

func (s *Store) UpdateLaunch(ctx context.Context, id int64, title, status *string, startDate, endDate *time.Time) (Launch, error) {
	var l Launch
	err := s.db.QueryRow(ctx,
		`UPDATE launches SET
			title = COALESCE($1, title),
			status = COALESCE($2, status),
			start_date = COALESCE($3, start_date),
			end_date = COALESCE($4, end_date)
		 WHERE id = $5
		 RETURNING id, project_id, title, status, start_date, end_date, created_at`,
		title, status, startDate, endDate, id).
		Scan(&l.ID, &l.ProjectID, &l.Title, &l.Status, &l.StartDate, &l.EndDate, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Launch{}, ErrNotFound
	}
	if err != nil {
		return Launch{}, err
	}
	return l, nil
}

func (s *Store) DeleteLaunch(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM launches WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- tasks ----------

func (s *Store) ListTasks(ctx context.Context, launchID, assigneeID int64) ([]Task, error) {
	var rows pgx.Rows
	var err error
	switch {
	case launchID != 0 && assigneeID != 0:
		rows, err = s.db.Query(ctx,
			`SELECT id, launch_id, title, status, assignee_id, due_date, created_at
			 FROM tasks WHERE launch_id = $1 AND assignee_id = $2 ORDER BY id`,
			launchID, assigneeID)
	case launchID != 0:
		rows, err = s.db.Query(ctx,
			`SELECT id, launch_id, title, status, assignee_id, due_date, created_at
			 FROM tasks WHERE launch_id = $1 ORDER BY id`, launchID)
	case assigneeID != 0:
		rows, err = s.db.Query(ctx,
			`SELECT id, launch_id, title, status, assignee_id, due_date, created_at
			 FROM tasks WHERE assignee_id = $1 ORDER BY id`, assigneeID)
	default:
		rows, err = s.db.Query(ctx,
			`SELECT id, launch_id, title, status, assignee_id, due_date, created_at
			 FROM tasks ORDER BY id`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.LaunchID, &t.Title, &t.Status, &t.AssigneeID, &t.DueDate, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *Store) GetTask(ctx context.Context, id int64) (Task, error) {
	var t Task
	err := s.db.QueryRow(ctx,
		`SELECT id, launch_id, title, status, assignee_id, due_date, created_at
		 FROM tasks WHERE id = $1`, id).
		Scan(&t.ID, &t.LaunchID, &t.Title, &t.Status, &t.AssigneeID, &t.DueDate, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) CreateTask(ctx context.Context, launchID int64, title string, assigneeID *int64, dueDate *time.Time) (Task, error) {
	var t Task
	err := s.db.QueryRow(ctx,
		`INSERT INTO tasks (launch_id, title, assignee_id, due_date)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, launch_id, title, status, assignee_id, due_date, created_at`,
		launchID, title, assigneeID, dueDate).
		Scan(&t.ID, &t.LaunchID, &t.Title, &t.Status, &t.AssigneeID, &t.DueDate, &t.CreatedAt)
	if err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) UpdateTask(ctx context.Context, id int64, title, status *string, assigneeID *int64, dueDate *time.Time) (Task, error) {
	var t Task
	err := s.db.QueryRow(ctx,
		`UPDATE tasks SET
			title = COALESCE($1, title),
			status = COALESCE($2, status),
			assignee_id = COALESCE($3, assignee_id),
			due_date = COALESCE($4, due_date)
		 WHERE id = $5
		 RETURNING id, launch_id, title, status, assignee_id, due_date, created_at`,
		title, status, assigneeID, dueDate, id).
		Scan(&t.ID, &t.LaunchID, &t.Title, &t.Status, &t.AssigneeID, &t.DueDate, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) DeleteTask(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- comments ----------

func (s *Store) ListComments(ctx context.Context, taskID int64) ([]Comment, error) {
	var rows pgx.Rows
	var err error
	if taskID == 0 {
		rows, err = s.db.Query(ctx,
			`SELECT id, task_id, user_id, text, created_at FROM comments ORDER BY id`)
	} else {
		rows, err = s.db.Query(ctx,
			`SELECT id, task_id, user_id, text, created_at
			 FROM comments WHERE task_id = $1 ORDER BY id`, taskID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Text, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (s *Store) CreateComment(ctx context.Context, taskID, userID int64, text string) (Comment, error) {
	var c Comment
	err := s.db.QueryRow(ctx,
		`INSERT INTO comments (task_id, user_id, text)
		 VALUES ($1, $2, $3)
		 RETURNING id, task_id, user_id, text, created_at`,
		taskID, userID, text).
		Scan(&c.ID, &c.TaskID, &c.UserID, &c.Text, &c.CreatedAt)
	if err != nil {
		return Comment{}, err
	}
	return c, nil
}

func (s *Store) DeleteComment(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM comments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
