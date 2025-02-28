package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config stores all configuration of the application
// The values are read by viper from a config file or environment variables
type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	JWTSecret      string
	TokenExpiresIn int
}

// AppConfig holds the application configuration
var AppConfig Config

// LoadConfig reads configuration from environment variables or .env file
func LoadConfig() {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Set default values
	AppConfig = Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "postgres"),
		DBName:         getEnv("DB_NAME", "workorder"),
		JWTSecret:      getEnv("JWT_SECRET", "your-secret-key"),
		TokenExpiresIn: getEnvAsInt("TOKEN_EXPIRES_IN", 24), // hours
	}

	log.Println("Configuration loaded successfully")
}

// Helper function to read an environment variable or return a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Helper function to read an environment variable as integer or return a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
