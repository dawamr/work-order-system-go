package main

import (
	"log"

	"github.com/dawamr/work-order-system-go/config"
	"github.com/dawamr/work-order-system-go/database"
)

func main() {
	log.Println("=== Database Migration Tool ===")
	
	// Load configuration
	config.LoadConfig()
	
	// Connect to database
	database.ConnectDB()
	
	// Run migration
	database.MigrateDB()
	
	log.Println("=== Migration completed successfully! ===")
}
