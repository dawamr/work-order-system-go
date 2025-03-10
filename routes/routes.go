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

	// Api for list all operators
	operators := api.Group("/operators")
	operators.Get("/", controllers.GetOperators)

	// Work Order routes
	workOrders := api.Group("/work-orders")

	// Definisikan route statis terlebih dahulu
	workOrders.Get("/assigned", middleware.RoleAuthorization(models.RoleOperator), controllers.GetAssignedWorkOrders)

	// Kemudian definisikan route dengan parameter
	workOrders.Get("/:id", controllers.GetWorkOrderByID)
	workOrders.Get("/:id/progress", controllers.GetWorkOrderProgress)

	// Routes for Production Manager only
	workOrders.Post("/", middleware.RoleAuthorization(models.RoleProductionManager), controllers.CreateWorkOrder)
	workOrders.Get("/", middleware.RoleAuthorization(models.RoleProductionManager), controllers.GetWorkOrders)
	workOrders.Put("/:id", middleware.RoleAuthorization(models.RoleProductionManager), controllers.UpdateWorkOrder)
	workOrders.Delete("/:id", middleware.RoleAuthorization(models.RoleProductionManager), controllers.DeleteWorkOrder)
	// Work order logs
	workOrders.Get("/:id/logs", controllers.GetWorkOrderLogs)
	workOrders.Post("/:id/logs", controllers.CreateWorkOrderLog)

	// Routes for Operator only
	workOrders.Put("/:id/status", controllers.UpdateWorkOrderStatus)
	workOrders.Post("/:id/progress", controllers.CreateWorkOrderProgress)

	// Report routes (Production Manager only)
	reports := api.Group("/reports")
	reports.Get("/dashboard", controllers.GetWorkOrderDashboard)
	reports.Get("/performance", middleware.RoleAuthorization(models.RoleProductionManager), controllers.GetOperatorPerformance)
	reports.Get("/summary", middleware.RoleAuthorization(models.RoleProductionManager), controllers.GetWorkOrderSummary)
	reports.Get("/summary/:operator_id", middleware.RoleAuthorization(models.RoleProductionManager), controllers.GetWorkOrderSummaryByOperator)

	// Audit log routes (Production Manager only)
	auditLogs := api.Group("/audit-logs", middleware.RoleAuthorization(models.RoleProductionManager))
	auditLogs.Get("/", controllers.GetAuditLogs)
}
