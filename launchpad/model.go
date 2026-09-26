package main

import "time"

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Status      string    `json:"status"`
	OwnerID     int64     `json:"owner_id"`
	Budget      *float64  `json:"budget"`
	CreatedAt   time.Time `json:"created_at"`
}

type Launch struct {
	ID        int64      `json:"id"`
	ProjectID int64      `json:"project_id"`
	Title     string     `json:"title"`
	Status    string     `json:"status"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	CreatedAt time.Time  `json:"created_at"`
}

type Task struct {
	ID         int64      `json:"id"`
	LaunchID   int64      `json:"launch_id"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	AssigneeID *int64     `json:"assignee_id"`
	DueDate    *time.Time `json:"due_date"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Comment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
