package main

import (
	"context"
	"errors"
	"gorm.io/gorm"
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
	// Load environment and configuration
	config.LoadEnv()

	dsn := config.GetDSN()
	port := config.GetPort()

	dbConn := mustConnectDB(dsn)
	mustPingDB(dbConn)
	mustMigrateDB(dbConn)
	insertTestDataIfEmpty(dbConn)

	// Set up HTTP server
	server := setupServer(port, dbConn)

	// Start server with graceful shutdown
	startServer(server)
}

// mustConnectDB returns a DB connection or exits on failure
func mustConnectDB(dsn string) *gorm.DB {
	dbConn, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("ERROR: Database connection failed: %v", err)
	}
	return dbConn
}

// mustPingDB checks DB connection
func mustPingDB(dbConn *gorm.DB) {
	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Fatalf("ERROR: Could not get generic DB object: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("ERROR: Database ping failed: %v", err)
	}
	log.Println("INFO: Database connection OK")
}

// mustMigrateDB auto-migrates HelloWorld schema
func mustMigrateDB(dbConn *gorm.DB) {
	if err := dbConn.AutoMigrate(&db.HelloWorld{}); err != nil {
		log.Fatalf("ERROR: AutoMigrate failed: %v", err)
	}
}

// insertTestDataIfEmpty inserts test data if table is empty
func insertTestDataIfEmpty(dbConn *gorm.DB) {
	var count int64
	dbConn.Model(&db.HelloWorld{}).Count(&count)
	if count == 0 {
		testData := []db.HelloWorld{
			{Message: "Hello, World!"},
			{Message: "Test message"},
		}
		if err := dbConn.Create(&testData).Error; err != nil {
			log.Fatalf("ERROR: Test data insertion failed: %v", err)
		}
		log.Println("INFO: Test data inserted into HelloWorld table")
	}
}

// setupServer prepares HTTP server and routes
func setupServer(port string, dbConn *gorm.DB) *http.Server {
	mux := http.NewServeMux()
	// Order matters for correct routing
	mux.Handle("GET /hello", api.ListHelloHandler(dbConn))
	mux.Handle("GET /hello/", api.GetHelloHandler(dbConn))
	mux.Handle("POST /hello", api.AddHelloHandler(dbConn))
	mux.Handle("DELETE /hello/", api.DeleteHelloHandler(dbConn))
	mux.Handle("GET /health", api.HealthHandler(dbConn))

	loggedMux := api.LoggingMiddleware(mux)
	return &http.Server{
		Addr:    ":" + port,
		Handler: loggedMux,
	}
}

// startServer runs server and handles graceful shutdown
func startServer(server *http.Server) {
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

	log.Printf("INFO: Server starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("ERROR: Server failed: %v", err)
	}

	<-idleConnsClosed
	log.Println("INFO: Server stopped")
}
