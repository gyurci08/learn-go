package main

import (
	"context"
	"errors"
	"learn-go/internal/api"
	"learn-go/internal/config"
	"learn-go/internal/db"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	config.LoadEnv()
	dsn := config.GetDSN()
	port := config.GetPort()

	dbConn, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("ERROR: Database connection failed: %v", err)
	}

	// Check DB connection with ping
	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Fatalf("ERROR: Could not get generic DB object: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("ERROR: Database ping failed: %v", err)
	}
	log.Println("INFO: Database connection OK")

	// Auto-migrate HelloWorld schema
	if err := dbConn.AutoMigrate(&db.HelloWorld{}); err != nil {
		log.Fatalf("ERROR: AutoMigrate failed: %v", err)
	}

	// Set up HTTP router
	mux := http.NewServeMux()
	mux.Handle("GET /hello", api.ListHelloHandler(dbConn))
	mux.Handle("POST /hello", api.AddHelloHandler(dbConn))
	mux.Handle("DELETE /hello/", api.DeleteHelloHandler(dbConn)) // DELETE /hello/1
	mux.Handle("GET /health", api.HealthHandler(dbConn))

	// Wrap router with logging middleware
	loggedMux := api.LoggingMiddleware(mux)

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
