package api

import (
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"learn-go/internal/db"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// errorResponse is the JSON envelope for errors.
type errorResponse struct {
	Error string `json:"error"`
}

// dataResponse is the JSON envelope for data.
type dataResponse struct {
	Data interface{} `json:"data"`
}

// writeJSON sends a JSON response with the given HTTP status code.
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		log.Printf("Could not encode JSON response: %v", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
	}
}

// writeError logs and returns a standardized error response.
func writeError(w http.ResponseWriter, code int, userMsg string, err error) {
	if err != nil {
		log.Printf("HTTP %d: %s | Internal error: %v", code, userMsg, err)
	} else {
		log.Printf("HTTP %d: %s", code, userMsg)
	}
	writeJSON(w, code, errorResponse{Error: userMsg})
}

// extractID extracts a positive integer ID from the URL, after the given prefix.
func extractID(r *http.Request, prefix string) (int, error) {
	idStr := strings.TrimPrefix(r.URL.Path, prefix)
	idStr = strings.Trim(idStr, "/")
	return strconv.Atoi(idStr)
}

// ListHelloHandler handles GET /hello. Returns all HelloWorld messages.
func ListHelloHandler(dbConn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var hellos []db.HelloWorld
		if err := dbConn.WithContext(r.Context()).Find(&hellos).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "DB error", err)
			return
		}
		writeJSON(w, http.StatusOK, dataResponse{Data: hellos})
	}
}

// GetHelloHandler handles GET /hello/{id}. Returns a single HelloWorld message.
func GetHelloHandler(dbConn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := extractID(r, "/hello/")
		if err != nil || id < 1 {
			writeError(w, http.StatusBadRequest, "ID must be a positive integer", err)
			return
		}
		var h db.HelloWorld
		if err := dbConn.WithContext(r.Context()).First(&h, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeError(w, http.StatusNotFound, "Not found", nil)
			} else {
				writeError(w, http.StatusInternalServerError, "DB error", err)
			}
			return
		}
		writeJSON(w, http.StatusOK, dataResponse{Data: h})
	}
}

// AddHelloHandler handles POST /hello. Adds a new HelloWorld message.
func AddHelloHandler(dbConn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var h db.HelloWorld
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&h); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON payload", err)
			return
		}
		if strings.TrimSpace(h.Message) == "" {
			writeError(w, http.StatusBadRequest, "Message is required", nil)
			return
		}
		if err := dbConn.WithContext(r.Context()).Create(&h).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "DB error", err)
			return
		}
		writeJSON(w, http.StatusCreated, dataResponse{Data: h})
	}
}

// DeleteHelloHandler handles DELETE /hello/{id}. Deletes a HelloWorld message.
func DeleteHelloHandler(dbConn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := extractID(r, "/hello/")
		if err != nil || id < 1 {
			writeError(w, http.StatusBadRequest, "ID must be a positive integer", err)
			return
		}
		res := dbConn.WithContext(r.Context()).Delete(&db.HelloWorld{}, id)
		if err := res.Error; err != nil {
			writeError(w, http.StatusInternalServerError, "DB error", err)
			return
		}
		if res.RowsAffected == 0 {
			writeError(w, http.StatusNotFound, "Not found", nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// HealthHandler handles GET /health. Simple health check with DB ping.
func HealthHandler(dbConn *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sqlDB, err := dbConn.DB()
		if err != nil || sqlDB.PingContext(r.Context()) != nil {
			writeError(w, http.StatusServiceUnavailable, "DB not available", err)
			return
		}
		writeJSON(w, http.StatusOK, dataResponse{Data: map[string]string{"status": "ok"}})
	}
}
