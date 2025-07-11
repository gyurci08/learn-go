package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	// Read database configuration from environment
	//dsn := os.Getenv("DATABASE_DSN")
	//if dsn == "" {
	//	log.Fatal("DATABASE_DSN environment variable not set")
	//}

	// Initialize database connection
	//db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	//if err != nil {
	//	log.Fatalf("Database connection failed: %v", err)
	//}

	// Set up HTTP router using standard library
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", helloHandler)

	// Start server
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
