package main

import (
	"log"

	"github.com/dawamr/work-order-system-go/config"
	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/utils/seeder"
)

func main() {
	log.Println("=== Database Seeder Tool ===")
	log.Println("WARNING: This will DELETE all existing data!")
	
	// Load configuration
	config.LoadConfig()
	
	// Connect to database
	database.ConnectDB()
	
	// Run migration first (to ensure tables exist)
	database.MigrateDB()
	
	// Run seeder
	seeder.SeedAll()
	
	log.Println("=== Seeding completed successfully! ===")
}
