package main

import (
	"log"
	"os"

	"github.com/dawamr/work-order-system-go/config"
	"github.com/dawamr/work-order-system-go/database"
	_ "github.com/dawamr/work-order-system-go/docs" // Import generated Swagger docs
	"github.com/dawamr/work-order-system-go/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title Work Order System API
// @version 1.0
// @description This is the API documentation for the Work Order System
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email your.email@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize database connection
	database.ConnectDB()
	database.MigrateDB()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			return c.Status(code).JSON(fiber.Map{
				"error": true,
				"msg":   err.Error(),
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE",
	}))

	// Health check endpoint (for hosting platform verification)
	app.Get("/kaithheathcheck", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"message": "Application is running",
			"service": "Work Order System API",
		})
	})
	app.Get("/kaithhealth", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"message": "Application is running",
			"service": "Work Order System API",
		})
	})

	// Swagger UI route
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Setup routes
	routes.SetupRoutes(app)

	// Get port from environment variable or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start the server
	log.Printf("Starting server on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
