package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// HelloWorld is a simple GORM model for demonstration.
type HelloWorld struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Message string `gorm:"type:varchar(255)" json:"message"`
}

var db *gorm.DB // Global DB handle

// ErrorResponse is a unified error response structure
type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("INFO: No .env file found or error loading .env file")
	}

	// Get required environment variables
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("ERROR: DATABASE_DSN environment variable not set")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("INFO: PORT not set, using default :8080")
	}

	// Connect to database
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("ERROR: Database connection failed: %v", err)
	}

	// Check DB connection with ping
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("ERROR: Could not get generic DB object: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("ERROR: Database ping failed: %v", err)
	}
	log.Println("INFO: Database connection OK")

	// Auto-migrate HelloWorld schema
	if err := db.AutoMigrate(&HelloWorld{}); err != nil {
		log.Fatalf("ERROR: AutoMigrate failed: %v", err)
	}

	// Set up HTTP router
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", listHelloHandler)
	mux.HandleFunc("POST /hello", addHelloHandler)
	mux.HandleFunc("DELETE /hello/", deleteHelloHandler) // DELETE /hello/1
	mux.HandleFunc("GET /health", healthHandler)

	// Wrap router with logging middleware
	loggedMux := loggingMiddleware(mux)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: loggedMux,
	}

	// Graceful shutdown setup
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		log.Println("INFO: Shutting down server gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("ERROR: HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("INFO: Server starting on :%s", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("ERROR: Server failed: %v", err)
	}

	<-idleConnsClosed
	log.Println("INFO: Server stopped")
}

// Middleware for logging every request
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DEBUG: %s %s endpoint accessed from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// List all HelloWorld messages
func listHelloHandler(w http.ResponseWriter, r *http.Request) {
	var hellos []HelloWorld
	if err := db.Find(&hellos).Error; err != nil {
		writeError(w, "DB error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, hellos, http.StatusOK)
}

// Add a new HelloWorld message
func addHelloHandler(w http.ResponseWriter, r *http.Request) {
	var h HelloWorld
	if err := json.NewDecoder(r.Body).Decode(&h); err != nil {
		writeError(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if h.Message == "" {
		writeError(w, "Message is required", http.StatusBadRequest)
		return
	}
	if err := db.Create(&h).Error; err != nil {
		writeError(w, "DB error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, h, http.StatusCreated)
}

// Delete a HelloWorld message by ID
func deleteHelloHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/hello/")
	if idStr == "" {
		writeError(w, "ID required", http.StatusBadRequest)
		return
	}
	if err := db.Delete(&HelloWorld{}, idStr).Error; err != nil {
		writeError(w, "DB error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := db.DB()
	if err != nil || sqlDB.Ping() != nil {
		writeError(w, "DB not available", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// Helper to write error responses as JSON
func writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

// Helper to write any data as JSON
func writeJSON(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
