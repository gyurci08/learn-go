package main

import (
	"encoding/json"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// HelloWorld is a simple GORM model for demonstration.
type HelloWorld struct {
	ID      uint   `gorm:"primaryKey"`
	Message string `gorm:"type:varchar(255)"`
}

func main() {
	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading .env file")
	}

	// Get required environment variable
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("DATABASE_DSN environment variable not set")
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Auto-migrate HelloWorld schema
	if err := db.AutoMigrate(&HelloWorld{}); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	// Set up HTTP router
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", helloHandler)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Message string `json:"message"`
	}{
		Message: "Hello, World!",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
