package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var allowedUserRoles = map[string]bool{
	"producer": true, "marketer": true, "methodist": true, "lead": true,
}
var allowedProjectStatuses = map[string]bool{
	"search": true, "in_progress": true, "launched": true, "archived": true, "cancelled": true,
}
var allowedLaunchStatuses = map[string]bool{
	"planned": true, "running": true, "finished": true, "cancelled": true,
}
var allowedTaskStatuses = map[string]bool{
	"todo": true, "in_progress": true, "done": true,
}

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// handleStoreError maps a store error to an HTTP response without leaking
// raw database details to the client.
func handleStoreError(w http.ResponseWriter, err error, notFoundMsg string) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, notFoundMsg)
		return
	}
	switch pgErrorCode(err) {
	case pgUniqueViolation:
		writeError(w, http.StatusUnprocessableEntity, "already exists")
	case pgForeignKeyViolation:
		writeError(w, http.StatusUnprocessableEntity, "related record does not exist")
	default:
		writeError(w, http.StatusInternalServerError, "database error")
	}
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// queryInt64 reads an optional numeric query param; 0 means "not provided".
func queryInt64(r *http.Request, name string) (int64, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

func parseDate(raw string) (*time.Time, bool) {
	if raw == "" {
		return nil, true
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, false
	}
	return &t, true
}

// ---------- users ----------

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	u, err := h.store.GetUser(r.Context(), id)
	if err != nil {
		handleStoreError(w, err, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	if req.Role == "" {
		req.Role = "producer"
	}

	if req.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusUnprocessableEntity, "email is not valid")
		return
	}
	if !allowedUserRoles[req.Role] {
		writeError(w, http.StatusUnprocessableEntity, "unknown role")
		return
	}

	u, err := h.store.CreateUser(r.Context(), req.Name, req.Email, req.Role)
	if err != nil {
		handleStoreError(w, err, "user not found")
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

type updateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Role  *string `json:"role"`
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == nil && req.Email == nil && req.Role == nil {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			writeError(w, http.StatusUnprocessableEntity, "name cannot be empty")
			return
		}
		req.Name = &trimmed
	}
	if req.Email != nil && !strings.Contains(*req.Email, "@") {
		writeError(w, http.StatusUnprocessableEntity, "email is not valid")
		return
	}
	if req.Role != nil && !allowedUserRoles[*req.Role] {
		writeError(w, http.StatusUnprocessableEntity, "unknown role")
		return
	}

	u, err := h.store.UpdateUser(r.Context(), id, req.Name, req.Email, req.Role)
	if err != nil {
		handleStoreError(w, err, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	if err := h.store.DeleteUser(r.Context(), id); err != nil {
		handleStoreError(w, err, "user not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- projects ----------

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && !allowedProjectStatuses[status] {
		writeError(w, http.StatusBadRequest, "unknown status filter")
		return
	}
	projects, err := h.store.ListProjects(r.Context(), status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (h *Handler) getProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	p, err := h.store.GetProject(r.Context(), id)
	if err != nil {
		handleStoreError(w, err, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type createProjectRequest struct {
	Title       string   `json:"title"`
	Description *string  `json:"description"`
	OwnerID     int64    `json:"owner_id"`
	Budget      *float64 `json:"budget"`
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	if req.OwnerID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "owner_id is required")
		return
	}
	if req.Budget != nil && *req.Budget < 0 {
		writeError(w, http.StatusUnprocessableEntity, "budget cannot be negative")
		return
	}

	p, err := h.store.CreateProject(r.Context(), req.Title, req.Description, req.OwnerID, req.Budget)
	if err != nil {
		handleStoreError(w, err, "project not found")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

type updateProjectRequest struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Status      *string  `json:"status"`
	OwnerID     *int64   `json:"owner_id"`
	Budget      *float64 `json:"budget"`
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == nil && req.Description == nil && req.Status == nil && req.OwnerID == nil && req.Budget == nil {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			writeError(w, http.StatusUnprocessableEntity, "title cannot be empty")
			return
		}
		req.Title = &trimmed
	}
	if req.Status != nil && !allowedProjectStatuses[*req.Status] {
		writeError(w, http.StatusUnprocessableEntity, "unknown status")
		return
	}
	if req.OwnerID != nil && *req.OwnerID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "owner_id must be positive")
		return
	}
	if req.Budget != nil && *req.Budget < 0 {
		writeError(w, http.StatusUnprocessableEntity, "budget cannot be negative")
		return
	}

	p, err := h.store.UpdateProject(r.Context(), id, req.Title, req.Description, req.Status, req.OwnerID, req.Budget)
	if err != nil {
		handleStoreError(w, err, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	if err := h.store.DeleteProject(r.Context(), id); err != nil {
		handleStoreError(w, err, "project not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- launches ----------

func (h *Handler) listLaunches(w http.ResponseWriter, r *http.Request) {
	projectID, ok := queryInt64(r, "project_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "project_id must be a positive number")
		return
	}
	launches, err := h.store.ListLaunches(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, launches)
}

func (h *Handler) getLaunch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	l, err := h.store.GetLaunch(r.Context(), id)
	if err != nil {
		handleStoreError(w, err, "launch not found")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

type createLaunchRequest struct {
	ProjectID int64  `json:"project_id"`
	Title     string `json:"title"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func (h *Handler) createLaunch(w http.ResponseWriter, r *http.Request) {
	var req createLaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	if req.ProjectID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "project_id is required")
		return
	}
	startDate, ok := parseDate(req.StartDate)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "start_date must be YYYY-MM-DD")
		return
	}
	endDate, ok := parseDate(req.EndDate)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "end_date must be YYYY-MM-DD")
		return
	}

	l, err := h.store.CreateLaunch(r.Context(), req.ProjectID, req.Title, startDate, endDate)
	if err != nil {
		handleStoreError(w, err, "launch not found")
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

type updateLaunchRequest struct {
	Title     *string `json:"title"`
	Status    *string `json:"status"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

func (h *Handler) updateLaunch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}

	var req updateLaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == nil && req.Status == nil && req.StartDate == nil && req.EndDate == nil {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			writeError(w, http.StatusUnprocessableEntity, "title cannot be empty")
			return
		}
		req.Title = &trimmed
	}
	if req.Status != nil && !allowedLaunchStatuses[*req.Status] {
		writeError(w, http.StatusUnprocessableEntity, "unknown status")
		return
	}
	var startDate, endDate *time.Time
	if req.StartDate != nil {
		d, ok := parseDate(*req.StartDate)
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "start_date must be YYYY-MM-DD")
			return
		}
		startDate = d
	}
	if req.EndDate != nil {
		d, ok := parseDate(*req.EndDate)
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "end_date must be YYYY-MM-DD")
			return
		}
		endDate = d
	}

	l, err := h.store.UpdateLaunch(r.Context(), id, req.Title, req.Status, startDate, endDate)
	if err != nil {
		handleStoreError(w, err, "launch not found")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (h *Handler) deleteLaunch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	if err := h.store.DeleteLaunch(r.Context(), id); err != nil {
		handleStoreError(w, err, "launch not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- tasks ----------

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	launchID, ok := queryInt64(r, "launch_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "launch_id must be a positive number")
		return
	}
	assigneeID, ok := queryInt64(r, "assignee_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "assignee_id must be a positive number")
		return
	}
	tasks, err := h.store.ListTasks(r.Context(), launchID, assigneeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	t, err := h.store.GetTask(r.Context(), id)
	if err != nil {
		handleStoreError(w, err, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

type createTaskRequest struct {
	LaunchID   int64  `json:"launch_id"`
	Title      string `json:"title"`
	AssigneeID *int64 `json:"assignee_id"`
	DueDate    string `json:"due_date"`
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	if req.LaunchID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "launch_id is required")
		return
	}
	if req.AssigneeID != nil && *req.AssigneeID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "assignee_id must be positive")
		return
	}
	dueDate, ok := parseDate(req.DueDate)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "due_date must be YYYY-MM-DD")
		return
	}

	t, err := h.store.CreateTask(r.Context(), req.LaunchID, req.Title, req.AssigneeID, dueDate)
	if err != nil {
		handleStoreError(w, err, "task not found")
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

type updateTaskRequest struct {
	Title      *string `json:"title"`
	Status     *string `json:"status"`
	AssigneeID *int64  `json:"assignee_id"`
	DueDate    *string `json:"due_date"`
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == nil && req.Status == nil && req.AssigneeID == nil && req.DueDate == nil {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			writeError(w, http.StatusUnprocessableEntity, "title cannot be empty")
			return
		}
		req.Title = &trimmed
	}
	if req.Status != nil && !allowedTaskStatuses[*req.Status] {
		writeError(w, http.StatusUnprocessableEntity, "unknown status")
		return
	}
	if req.AssigneeID != nil && *req.AssigneeID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "assignee_id must be positive")
		return
	}
	var dueDate *time.Time
	if req.DueDate != nil {
		d, ok := parseDate(*req.DueDate)
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "due_date must be YYYY-MM-DD")
			return
		}
		dueDate = d
	}

	t, err := h.store.UpdateTask(r.Context(), id, req.Title, req.Status, req.AssigneeID, dueDate)
	if err != nil {
		handleStoreError(w, err, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	if err := h.store.DeleteTask(r.Context(), id); err != nil {
		handleStoreError(w, err, "task not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- comments ----------

func (h *Handler) listComments(w http.ResponseWriter, r *http.Request) {
	taskID, ok := queryInt64(r, "task_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "task_id must be a positive number")
		return
	}
	comments, err := h.store.ListComments(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, comments)
}

type createCommentRequest struct {
	TaskID int64  `json:"task_id"`
	UserID int64  `json:"user_id"`
	Text   string `json:"text"`
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		writeError(w, http.StatusUnprocessableEntity, "text is required")
		return
	}
	if req.TaskID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "task_id is required")
		return
	}
	if req.UserID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "user_id is required")
		return
	}

	c, err := h.store.CreateComment(r.Context(), req.TaskID, req.UserID, req.Text)
	if err != nil {
		handleStoreError(w, err, "comment not found")
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	if err := h.store.DeleteComment(r.Context(), id); err != nil {
		handleStoreError(w, err, "comment not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
