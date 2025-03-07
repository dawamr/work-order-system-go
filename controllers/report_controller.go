package controllers

import (
	"time"

	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/models"
	"github.com/gofiber/fiber/v2"
)

// WorkOrderSummary represents a summary of work orders by status
type WorkOrderSummary struct {
	Status    models.WorkOrderStatus `json:"status"`
	Count     int64                  `json:"count"`
	Quantity  int64                  `json:"quantity"`
	Completed int64                  `json:"completed"`
}

// OperatorPerformance represents an operator's performance metrics
type OperatorPerformance struct {
	OperatorID   uint   `json:"operator_id"`
	Username     string `json:"username"`
	Assigned     int64  `json:"assigned"`
	InProgress   int64  `json:"in_progress"`
	Completed    int64  `json:"completed"`
	TotalQuantity int64 `json:"total_quantity"`
}

// SummaryResponse represents a work order summary response
type SummaryResponse struct {
	Error   bool              `json:"error"`
	Summary []WorkOrderSummary `json:"summary"`
}

// PerformanceResponse represents an operator performance response
type PerformanceResponse struct {
	Error        bool                  `json:"error"`
	Performances []OperatorPerformance `json:"performances"`
}

// @Summary Get work order summary
// @Description Get a summary report of work orders by status (Production Manager only)
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} SummaryResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /reports/summary [get]
func GetWorkOrderSummary(c *fiber.Ctx) error {
	// Only Production Manager can view reports
	role := c.Locals("role").(models.Role)
	if role != models.RoleProductionManager {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "Only Production Manager can view reports",
		})
	}

	// Get query parameters for date range
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Build query
	query := database.DB.Model(&models.WorkOrder{})

	// Apply date filters if provided
	if startDate != "" {
		startTime, err := time.Parse("2006-01-02", startDate)
		if err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if endDate != "" {
		endTime, err := time.Parse("2006-01-02", endDate)
		if err == nil {
			// Add one day to include the end date
			endTime = endTime.Add(24 * time.Hour)
			query = query.Where("created_at < ?", endTime)
		}
	}

	// Get summary by status
	var summaries []WorkOrderSummary

	// Pending
	var pendingSummary WorkOrderSummary
	pendingSummary.Status = models.StatusPending
	query.Where("status = ?", models.StatusPending).Count(&pendingSummary.Count)
	query.Where("status = ?", models.StatusPending).Select("COALESCE(SUM(quantity), 0)").Row().Scan(&pendingSummary.Quantity)
	pendingSummary.Completed = 0
	summaries = append(summaries, pendingSummary)

	// In Progress
	var inProgressSummary WorkOrderSummary
	inProgressSummary.Status = models.StatusInProgress
	query.Where("status = ?", models.StatusInProgress).Count(&inProgressSummary.Count)
	query.Where("status = ?", models.StatusInProgress).Select("COALESCE(SUM(quantity), 0)").Row().Scan(&inProgressSummary.Quantity)
	inProgressSummary.Completed = 0
	summaries = append(summaries, inProgressSummary)

	// Completed
	var completedSummary WorkOrderSummary
	completedSummary.Status = models.StatusCompleted
	query.Where("status = ?", models.StatusCompleted).Count(&completedSummary.Count)
	query.Where("status = ?", models.StatusCompleted).Select("COALESCE(SUM(quantity), 0)").Row().Scan(&completedSummary.Quantity)
	completedSummary.Completed = completedSummary.Quantity
	summaries = append(summaries, completedSummary)

	// Return summary
	return c.Status(fiber.StatusOK).JSON(SummaryResponse{
		Error:   false,
		Summary: summaries,
	})
}

// @Summary Get operator performance
// @Description Get performance metrics for operators (Production Manager only)
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} PerformanceResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /reports/operators [get]
func GetOperatorPerformance(c *fiber.Ctx) error {
	// Only Production Manager can view reports
	role := c.Locals("role").(models.Role)
	if role != models.RoleProductionManager {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "Only Production Manager can view reports",
		})
	}

	// Get query parameters for date range
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Get all operators
	var operators []models.User
	database.DB.Where("role = ?", models.RoleOperator).Find(&operators)

	// Prepare performance data
	var performances []OperatorPerformance

	for _, operator := range operators {
		performance := OperatorPerformance{
			OperatorID: operator.ID,
			Username:   operator.Username,
		}

		// Build query
		query := database.DB.Model(&models.WorkOrder{}).Where("operator_id = ?", operator.ID)

		// Apply date filters if provided
		if startDate != "" {
			startTime, err := time.Parse("2006-01-02", startDate)
			if err == nil {
				query = query.Where("created_at >= ?", startTime)
			}
		}
		if endDate != "" {
			endTime, err := time.Parse("2006-01-02", endDate)
			if err == nil {
				// Add one day to include the end date
				endTime = endTime.Add(24 * time.Hour)
				query = query.Where("created_at < ?", endTime)
			}
		}

		// Get counts by status
		query.Count(&performance.Assigned)
		query.Where("status = ?", models.StatusInProgress).Count(&performance.InProgress)
		query.Where("status = ?", models.StatusCompleted).Count(&performance.Completed)

		// Get total quantity
		query.Where("status = ?", models.StatusCompleted).Select("COALESCE(SUM(quantity), 0)").Row().Scan(&performance.TotalQuantity)

		performances = append(performances, performance)
	}

	// Return performance data
	return c.Status(fiber.StatusOK).JSON(PerformanceResponse{
		Error:        false,
		Performances: performances,
	})
}
