package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("INFO: No .env file found or error loading .env file")
	}
}

func GetDSN() string {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("ERROR: DATABASE_DSN environment variable not set")
	}
	return dsn
}

func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("INFO: PORT not set, using default :8080")
	}
	return port
}
