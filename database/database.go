package database

import (
	"fmt"
	"log"

	"github.com/dawamr/work-order-system-go/config"
	"github.com/dawamr/work-order-system-go/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the database instance
var DB *gorm.DB

// ConnectDB connects to the database
func ConnectDB() {
	var err error

	// Construct DSN (Data Source Name)
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.AppConfig.DBHost,
		config.AppConfig.DBPort,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBName,
	)

	// Connect to the database
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection established")
}

// MigrateDB performs database migration
func MigrateDB() {
	log.Println("Running database migrations...")

	// Auto migrate models
	err := DB.AutoMigrate(
		&models.User{},
		&models.WorkOrder{},
		&models.WorkOrderProgress{},
		&models.WorkOrderStatusHistory{},
	)

	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database migration completed")
}
