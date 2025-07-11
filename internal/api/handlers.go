package api

import (
	"encoding/json"
	"gorm.io/gorm"
	"learn-go/internal/db"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type DataResponse struct {
	Data interface{} `json:"data"`
}

// List all HelloWorld messages
func ListHelloHandler(dbConn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var hellos []db.HelloWorld
		if err := dbConn.Find(&hellos).Error; err != nil {
			WriteError(w, "DB error", http.StatusInternalServerError)
			return
		}
		WriteJSON(w, hellos, http.StatusOK)
	}
}

// Add a new HelloWorld message
func AddHelloHandler(dbConn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var h db.HelloWorld
		if err := json.NewDecoder(r.Body).Decode(&h); err != nil {
			WriteError(w, "Invalid input", http.StatusBadRequest)
			return
		}
		if h.Message == "" {
			WriteError(w, "Message is required", http.StatusBadRequest)
			return
		}
		if err := dbConn.Create(&h).Error; err != nil {
			WriteError(w, "DB error", http.StatusInternalServerError)
			return
		}
		WriteJSON(w, h, http.StatusCreated)
	}
}

// Delete a HelloWorld message by ID
func DeleteHelloHandler(dbConn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/hello/")
		if idStr == "" {
			WriteError(w, "ID required", http.StatusBadRequest)
			return
		}
		if err := dbConn.Delete(&db.HelloWorld{}, idStr).Error; err != nil {
			WriteError(w, "DB error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// Health check endpoint
func HealthHandler(dbConn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sqlDB, err := dbConn.DB()
		if err != nil || sqlDB.Ping() != nil {
			WriteError(w, "DB not available", http.StatusServiceUnavailable)
			return
		}
		WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	}
}

// Helper to write error responses as JSON
func WriteError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

// Helper to write any data as JSON
func WriteJSON(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(DataResponse{Data: data})
}
