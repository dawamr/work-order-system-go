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

	// Drop existing foreign key constraints if any
 	// Auto migrate models
 	err := DB.AutoMigrate(
 		&models.User{},
 		&models.WorkOrder{},
 		&models.WorkOrderProgress{},
 		&models.WorkOrderStatusHistory{},
 		&models.AuditLog{},
 	)

 	if err != nil {
 		log.Fatalf("Failed to migrate database: %v", err)
 	}

 	// Drop existing foreign key constraints if any
 	DB.Exec(`ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS fk_audit_logs_user`)

	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Check if constraint exists before adding it
	var constraintExists int64
	DB.Raw(`
		SELECT COUNT(1)
		FROM information_schema.table_constraints
		WHERE constraint_name = 'fk_audit_logs_user'
		AND table_name = 'audit_logs'
	`).Scan(&constraintExists)

	// Only add constraint if it doesn't exist
	if constraintExists == 0 {
		DB.Exec(`ALTER TABLE audit_logs
			ADD CONSTRAINT fk_audit_logs_user
			FOREIGN KEY (user_id)
			REFERENCES users(id)
			ON DELETE RESTRICT
			ON UPDATE CASCADE`)
	}

	log.Println("Database migration completed")
}
