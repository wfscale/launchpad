package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func mustEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	ctx := context.Background()

	dsn := mustEnv("DATABASE_URL", "postgres://app:secret@localhost:5432/app?sslmode=disable")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("cannot create pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("cannot reach database: %v", err)
	}

	h := NewHandler(NewStore(pool))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /users", h.listUsers)
	mux.HandleFunc("POST /users", h.createUser)
	mux.HandleFunc("GET /users/{id}", h.getUser)
	mux.HandleFunc("PATCH /users/{id}", h.updateUser)
	mux.HandleFunc("DELETE /users/{id}", h.deleteUser)

	mux.HandleFunc("GET /projects", h.listProjects)
	mux.HandleFunc("POST /projects", h.createProject)
	mux.HandleFunc("GET /projects/{id}", h.getProject)
	mux.HandleFunc("PATCH /projects/{id}", h.updateProject)
	mux.HandleFunc("DELETE /projects/{id}", h.deleteProject)

	mux.HandleFunc("GET /launches", h.listLaunches)
	mux.HandleFunc("POST /launches", h.createLaunch)
	mux.HandleFunc("GET /launches/{id}", h.getLaunch)
	mux.HandleFunc("PATCH /launches/{id}", h.updateLaunch)
	mux.HandleFunc("DELETE /launches/{id}", h.deleteLaunch)

	mux.HandleFunc("GET /tasks", h.listTasks)
	mux.HandleFunc("POST /tasks", h.createTask)
	mux.HandleFunc("GET /tasks/{id}", h.getTask)
	mux.HandleFunc("PATCH /tasks/{id}", h.updateTask)
	mux.HandleFunc("DELETE /tasks/{id}", h.deleteTask)

	mux.HandleFunc("GET /comments", h.listComments)
	mux.HandleFunc("POST /comments", h.createComment)
	mux.HandleFunc("DELETE /comments/{id}", h.deleteComment)

	addr := ":" + mustEnv("PORT", "8080")
	log.Printf("listening on %s", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      withLogging(withCORS(mux)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
