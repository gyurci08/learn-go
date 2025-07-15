package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	envDSN      = "DATABASE_DSN"
	envPort     = "PORT"
	defaultPort = 8080
)

// LoadEnv loads environment variables from one or more dot-env files.
func LoadEnv(files ...string) {
	if err := godotenv.Load(files...); err != nil {
		log.Println("INFO: No .env file found or error loading .env file")
	}
}

// GetDSN fetches the DATABASE_DSN environment variable (required).
func GetDSN() string {
	dsn := os.Getenv(envDSN)
	if dsn == "" {
		log.Fatal("ERROR: " + envDSN + " environment variable not set")
	}
	return dsn
}

// GetPort returns the server port as string for HTTP listeners.
// Defaults to 8080 if unset; logs this fact.
func GetPort() string {
	port := os.Getenv(envPort)
	if port == "" {
		log.Printf("INFO: %s not set, using default :%d", envPort, defaultPort)
		return strconv.Itoa(defaultPort)
	}
	return port
}
