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
	"syscall"
	"time"
)

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
	mux.HandleFunc("/api/v1/hello", helloHandler)

	fmt.Println("server env is " + cfg.Env)
	fmt.Println("server starting on :8080")

	handler := loggingMiddleware(mux)

	server := NewServer(":"+cfg.Port, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
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

func helloHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "hello, world!!!",
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
