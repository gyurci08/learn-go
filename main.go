package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type CreateUserRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}
type FakeDatabase struct {
	Users []User `json:"users"`
}

func (db *FakeDatabase) GetUser(ctx context.Context, id int) (User, error) {
	for _, user := range db.Users {
		if user.Id == id {
			return user, nil
		}
	}
	return User{}, fmt.Errorf("user not found")
}

func (db *FakeDatabase) CreateUser(ctx context.Context, user User) (User, error) {
	newUser := User{
		Id:   len(db.Users) + 1,
		Name: user.Name,
		Age:  user.Age,
	}
	db.Users = append(db.Users, newUser)
	return newUser, nil
}

type Config struct {
	Port string
	Env  string
}

type Server struct {
	httpServer *http.Server
}

func NewServer(address string, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    address,
			Handler: handler,
		},
	}
}

func (s *Server) start(errChan chan<- error) {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()
}

func (s *Server) stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Run() error {
	errChan := make(chan error, 1)
	s.start(errChan)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		slog.Info("stop signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return s.stop(ctx)
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("error: invalid env")
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/live", liveHandler)
	mux.HandleFunc("/ready", readyHandler)

	fmt.Println("server env is " + cfg.Env)

	db := &FakeDatabase{
		Users: []User{
			{Id: 1, Name: "John Doe", Age: 25},
		},
	}

	mux.HandleFunc("/api/v1/user", userHandler(db))

	fmt.Println("server starting on :8080")
	handler := loggingMiddleware(mux)
	server := NewServer(":"+cfg.Port, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}

func userHandler(db *FakeDatabase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// 1) GET: user lekérdezése id alapján
			strId := r.URL.Query().Get("id")
			if strId == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error": "missing id query parameter",
				})
				return
			}

			id, err := strconv.Atoi(strId)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error": "invalid id, must be an integer",
				})
				return
			}

			user, err := db.GetUser(r.Context(), id)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{
					"error": "user not found",
				})
				return
			}

			writeJSON(w, http.StatusOK, user)

		case http.MethodPost:
			// 2) POST: új user létrehozása JSON body-ból
			var req CreateUserRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error": "invalid JSON body",
				})
				return
			}

			// esetleg minimális validáció
			if req.Name == "" || req.Age <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error": "name and age are required",
				})
				return
			}

			newUser, err := db.CreateUser(r.Context(), User{
				Name: req.Name,
				Age:  req.Age,
			})
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "could not create user",
				})
				return
			}

			// 201 Created + létrehozott user
			writeJSON(w, http.StatusCreated, newUser)

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
		}
	}
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "ok",
	}

	writeJSON(w, http.StatusOK, response)
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "ready",
	}

	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Println("error writing response ", err)
	}
}

func loadConfig() (Config, error) {
	port := os.Getenv("PORT")
	env := os.Getenv("ENV")

	if port == "" {
		port = "8080"
	}
	if env == "" {
		env = "development"
	}

	return Config{
		Port: port,
		Env:  env,
	}, nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		slog.Info("request", "latency", time.Since(start))
	})
}
