package routes

import (
	"github.com/dawamr/work-order-system-go/controllers"
	"github.com/dawamr/work-order-system-go/middleware"
	"github.com/dawamr/work-order-system-go/models"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes sets up all the routes for the application
func SetupRoutes(app *fiber.App) {
	// Public routes
	auth := app.Group("/api/auth")
	auth.Post("/login", controllers.Login)
	auth.Post("/register", controllers.Register)

	// Protected routes
	api := app.Group("/api", middleware.Protected())

	// Work Order routes
	workOrders := api.Group("/work-orders")

	// Routes for both Production Manager and Operator
	workOrders.Get("/:id", controllers.GetWorkOrderByID)
	workOrders.Get("/:id/progress", controllers.GetWorkOrderProgress)
	workOrders.Get("/:id/history", controllers.GetWorkOrderStatusHistory)

	// Routes for Production Manager only
	workOrders.Post("/", middleware.RoleAuthorization(models.RoleProductionManager), controllers.CreateWorkOrder)
	workOrders.Get("/", middleware.RoleAuthorization(models.RoleProductionManager), controllers.GetWorkOrders)
	workOrders.Put("/:id", middleware.RoleAuthorization(models.RoleProductionManager), controllers.UpdateWorkOrder)

	// Routes for Operator only
	workOrders.Get("/assigned", middleware.RoleAuthorization(models.RoleOperator), controllers.GetAssignedWorkOrders)
	workOrders.Put("/:id/status", controllers.UpdateWorkOrderStatus)
	workOrders.Post("/:id/progress", controllers.CreateWorkOrderProgress)

	// Report routes (Production Manager only)
	reports := api.Group("/reports", middleware.RoleAuthorization(models.RoleProductionManager))
	reports.Get("/summary", controllers.GetWorkOrderSummary)
	reports.Get("/operators", controllers.GetOperatorPerformance)
}
