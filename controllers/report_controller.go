package controllers

import (
	"log"
	"sort"
	"strings"
	"time"

	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// WorkOrderDashboard represents a summary of work orders by status
type WorkOrderDashboard struct {
	Status    models.WorkOrderStatus `json:"status"`
	Count     int64                  `json:"count"`
}

type WorkOrderSummary struct {
	WorkOrderNumber string `json:"work_order_number"`
	ProductName     string `json:"product_name"`
	TotalWO         int64  `json:"total_wo"`
	Percentage      int64  `json:"percentage"`
	TargetQty       int64  `json:"target_qty"`
	AchievedQty     int64  `json:"achieved_qty"`
	Achievement     int64  `json:"achievement"`
	Pending         int64  `json:"pending"`
	InProgress      int64  `json:"in_progress"`
	Completed       int64  `json:"completed"`
	Cancelled       int64  `json:"cancelled"`
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
type DashboardResponse struct {
	Error   bool              `json:"error"`
	Summary []WorkOrderDashboard `json:"summary"`
}

type SummaryResponse struct {
	Error   bool              `json:"error"`
	Summary []WorkOrderSummary `json:"summary"`
}

// PerformanceResponse represents an operator performance response
type PerformanceResponse struct {
	Error        bool                  `json:"error"`
	Performances []OperatorPerformance `json:"performances"`
}

// @Summary Get work order Dashboard
// @Description Get a dashboard report of work orders by status
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} DashboardResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /reports/dashboard [get]
func GetWorkOrderDashboard(c *fiber.Ctx) error {
	role := c.Locals("role").(models.Role)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Build base query
	baseQuery := database.DB.Model(&models.WorkOrder{})

	// Apply date filters if provided
	if startDate != "" {
		startTime, err := time.Parse(time.DateOnly, startDate)
		if err == nil {
			baseQuery = baseQuery.Where("created_at >= ?", startTime)
		}
	}
	if endDate != "" {
		endTime, err := time.Parse(time.DateOnly, endDate)
		if err == nil {
			// Add one day to include the end date
			endTime = endTime.Add(24 * time.Hour)
			baseQuery = baseQuery.Where("created_at < ?", endTime)
		}
	}
	if role != models.RoleProductionManager {
		baseQuery = baseQuery.Where("operator_id = ?", c.Locals("user_id").(uint))
	}

	// Get summary by status using cloned queries
	var pendingSummary WorkOrderDashboard
	var inProgressSummary WorkOrderDashboard
	var completedSummary WorkOrderDashboard
	var totalWorkOrderCount WorkOrderDashboard

	// Clone base query for each status count
	baseQuery.Session(&gorm.Session{}).Where("status = ?", models.StatusPending).Count(&pendingSummary.Count)
	baseQuery.Session(&gorm.Session{}).Where("status = ?", models.StatusInProgress).Count(&inProgressSummary.Count)
	baseQuery.Session(&gorm.Session{}).Where("status = ?", models.StatusCompleted).Count(&completedSummary.Count)
	baseQuery.Session(&gorm.Session{}).Count(&totalWorkOrderCount.Count)

	pendingSummary.Status = models.StatusPending
	inProgressSummary.Status = models.StatusInProgress
	completedSummary.Status = models.StatusCompleted
	totalWorkOrderCount.Status = "total"

	summaries := []WorkOrderDashboard{pendingSummary, inProgressSummary, completedSummary, totalWorkOrderCount}

	return c.Status(fiber.StatusOK).JSON(DashboardResponse{
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
// @Router /reports/performance [get]
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
	if err := database.DB.Where("role = ?", models.RoleOperator).Find(&operators).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching operators",
		})
	}

	// Prepare performance data
	var performances []OperatorPerformance

	for _, operator := range operators {
		performance := OperatorPerformance{
			OperatorID: operator.ID,
			Username:   operator.Username,
		}

		// Build base query for this operator
		baseQuery := database.DB.Model(&models.WorkOrder{}).Where("operator_id = ?", operator.ID)

		// Apply date filters if provided
		if startDate != "" {
			startTime, err := time.Parse(time.DateOnly, startDate)
			if err == nil {
				baseQuery = baseQuery.Where("production_deadline >= ?", startTime)
			}
		}
		if endDate != "" {
			endTime, err := time.Parse(time.DateOnly, endDate)
			if err == nil {
				// Add one day to include the end date
				endTime = endTime.Add(24 * time.Hour)
				baseQuery = baseQuery.Where("production_deadline < ?", endTime)
			}
		}

		// Get total assigned work orders
		baseQuery.Session(&gorm.Session{}).Count(&performance.Assigned)

		// Get work orders in progress
		baseQuery.Session(&gorm.Session{}).
			Where("status = ?", models.StatusInProgress).
			Count(&performance.InProgress)

		// Get completed work orders
		baseQuery.Session(&gorm.Session{}).
			Where("status = ?", models.StatusCompleted).
			Count(&performance.Completed)

		// Get total quantity of completed work orders
		var totalQuantity int64
		if err := baseQuery.Session(&gorm.Session{}).
			Where("status = ?", models.StatusCompleted).
			Select("COALESCE(SUM(quantity), 0)").
			Row().Scan(&totalQuantity); err != nil {
			log.Printf("Error calculating total quantity for operator %d: %v", operator.ID, err)
		}
		performance.TotalQuantity = totalQuantity

		performances = append(performances, performance)
	}

	// performance sort by completed descending
	sort.Slice(performances, func(i, j int) bool {
		return performances[i].Completed > performances[j].Completed
	})

	// Return performance data
	return c.Status(fiber.StatusOK).JSON(PerformanceResponse{
		Error:        false,
		Performances: performances,
	})
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

	// Build base query
	baseQuery := database.DB.Model(&models.WorkOrder{})

	// Apply date filters if provided
	if startDate != "" {
		startTime, err := time.Parse(time.DateOnly, startDate)
		if err == nil {
			baseQuery = baseQuery.Where("production_deadline >= ?", startTime)
		}
	} else {
		baseQuery = baseQuery.Where("production_deadline >= ?", time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.Now().Location()))
	}
	if endDate != "" {
		endTime, err := time.Parse(time.DateOnly, endDate)
		if err == nil {
			// Add one day to include the end date
			endTime = endTime.Add(24 * time.Hour)
			baseQuery = baseQuery.Where("production_deadline < ?", endTime)
		}
	} else {
		baseQuery = baseQuery.Where("production_deadline < ?", time.Date(time.Now().Year(), 12, 31, 23, 59, 59, 0, time.Now().Location()))
	}

	// Get distinct product names
	var productNames []string
	if err := baseQuery.Session(&gorm.Session{}).
		Distinct("product_name").
		Pluck("product_name", &productNames).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching product names",
		})
	}

	// Get total count of work orders for percentage calculation
	var totalWorkOrders int64
	baseQuery.Session(&gorm.Session{}).Count(&totalWorkOrders)

	// Prepare summaries
	summaries := []WorkOrderSummary{}

	// For each product, calculate metrics
	for _, productName := range productNames {
		summary := WorkOrderSummary{
			ProductName: productName,
		}

		// Get all work order numbers for this product
		var workOrderNumbers []string
		if err := baseQuery.Session(&gorm.Session{}).
			Where("product_name = ?", productName).
			Distinct("work_order_number").
			Pluck("work_order_number", &workOrderNumbers).Error; err != nil {
			log.Printf("Error fetching work order numbers for %s: %v", productName, err)
			workOrderNumbers = []string{}
		}

		// Join work order numbers as a comma-separated list
		summary.WorkOrderNumber = strings.Join(workOrderNumbers, ", ")

		// Count total work orders for this product
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ?", productName).
			Count(&summary.TotalWO)

		// Calculate percentage of total work orders
		if totalWorkOrders > 0 {
			summary.Percentage = int64(float64(summary.TotalWO) / float64(totalWorkOrders) * 100)
		}

		// Get target quantity
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ?", productName).
			Select("COALESCE(SUM(target_quantity), 0)").
			Row().Scan(&summary.TargetQty)

		// Get achieved quantity (completed work orders)
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND status = ?", productName, models.StatusCompleted).
			Select("COALESCE(SUM(quantity), 0)").
			Row().Scan(&summary.AchievedQty)

		// Calculate achievement percentage
		if summary.TargetQty > 0 {
			summary.Achievement = int64(float64(summary.AchievedQty) / float64(summary.TargetQty) * 100)
		}

		// Count work orders by status
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND status = ?", productName, models.StatusPending).
			Count(&summary.Pending)

		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND status = ?", productName, models.StatusInProgress).
			Count(&summary.InProgress)

		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND status = ?", productName, models.StatusCompleted).
			Count(&summary.Completed)

		// For cancelled, we need to check if there's a "cancelled" status in your system
		// Since it's not defined in the models, we'll use a placeholder query
		// You might need to adjust this based on how cancelled orders are tracked
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND deleted_at IS NOT NULL", productName).
			Count(&summary.Cancelled)

		summaries = append(summaries, summary)
	}

	// Add a total summary row
	if len(summaries) > 0 {
		totalSummary := WorkOrderSummary{
			ProductName: "Total",
			TotalWO:     totalWorkOrders,
			Percentage:  100, // 100%
		}

		// Calculate totals
		for _, summary := range summaries {
			totalSummary.TargetQty += summary.TargetQty
			totalSummary.AchievedQty += summary.AchievedQty
			totalSummary.Pending += summary.Pending
			totalSummary.InProgress += summary.InProgress
			totalSummary.Completed += summary.Completed
			totalSummary.Cancelled += summary.Cancelled
		}

		// Calculate overall achievement percentage
		if totalSummary.TargetQty > 0 {
			totalSummary.Achievement = int64(float64(totalSummary.AchievedQty) / float64(totalSummary.TargetQty) * 100)
		}

		summaries = append(summaries, totalSummary)
	}

	return c.Status(fiber.StatusOK).JSON(SummaryResponse{
		Error:   false,
		Summary: summaries,
	})
}

// @Summary Get work order summary by operator
// @Description Get a summary report of work orders by status for a specific operator (Production Manager only)
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param operator_id path string true "Operator ID"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} SummaryResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /reports/summary/{operator_id} [get]
func GetWorkOrderSummaryByOperator(c *fiber.Ctx) error {
	role := c.Locals("role").(models.Role)
	if role != models.RoleProductionManager {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "Only Production Manager can view reports",
		})
	}

	// Get operator ID from path parameter
	operatorID := c.Params("operator_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Build base query
	baseQuery := database.DB.Model(&models.WorkOrder{}).Where("operator_id = ?", operatorID)

	// Apply date filters if provided
	if startDate != "" {
		startTime, err := time.Parse(time.DateOnly, startDate)
		if err == nil {
			baseQuery = baseQuery.Where("created_at >= ?", startTime)
		}
	} else {
		baseQuery = baseQuery.Where("created_at >= ?", time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.Now().Location()))
	}
	if endDate != "" {
		endTime, err := time.Parse(time.DateOnly, endDate)
		if err == nil {
			// Add one day to include the end date
			endTime = endTime.Add(24 * time.Hour)
			baseQuery = baseQuery.Where("created_at < ?", endTime)
		}
	} else {
		baseQuery = baseQuery.Where("created_at < ?", time.Date(time.Now().Year(), 12, 31, 23, 59, 59, 0, time.Now().Location()))
	}

	// Get distinct product names
	var productNames []string
	if err := baseQuery.Session(&gorm.Session{}).
		Distinct("product_name").
		Pluck("product_name", &productNames).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching product names",
		})
	}

	// Get total count of work orders for percentage calculation
	var totalWorkOrders int64
	baseQuery.Session(&gorm.Session{}).Count(&totalWorkOrders)

	// Prepare summaries
	summaries := []WorkOrderSummary{}

	// For each product, calculate metrics
	for _, productName := range productNames {
		summary := WorkOrderSummary{
			ProductName: productName,
		}

		// Get all work order numbers for this product
		var workOrderNumbers []string
		if err := baseQuery.Session(&gorm.Session{}).
			Where("product_name = ?", productName).
			Distinct("work_order_number").
			Pluck("work_order_number", &workOrderNumbers).Error; err != nil {
			log.Printf("Error fetching work order numbers for %s: %v", productName, err)
			workOrderNumbers = []string{}
		}

		// Join work order numbers as a comma-separated list
		summary.WorkOrderNumber = strings.Join(workOrderNumbers, ", ")

		// Count total work orders for this product
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ?", productName).
			Count(&summary.TotalWO)

		// Calculate percentage of total work orders
		if totalWorkOrders > 0 {
			summary.Percentage = int64(float64(summary.TotalWO) / float64(totalWorkOrders) * 100)
		}

		// Get target quantity
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ?", productName).
			Select("COALESCE(SUM(target_quantity), 0)").
			Row().Scan(&summary.TargetQty)

		// Get achieved quantity (completed work orders)
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND status = ?", productName, models.StatusCompleted).
			Select("COALESCE(SUM(quantity), 0)").
			Row().Scan(&summary.AchievedQty)

		// Calculate achievement percentage
		if summary.TargetQty > 0 {
			summary.Achievement = int64(float64(summary.AchievedQty) / float64(summary.TargetQty) * 100)
		}

		// Count work orders by status
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND status = ?", productName, models.StatusPending).
			Count(&summary.Pending)

		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND status = ?", productName, models.StatusInProgress).
			Count(&summary.InProgress)

		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND status = ?", productName, models.StatusCompleted).
			Count(&summary.Completed)

		// For cancelled, we need to check if there's a "cancelled" status in your system
		baseQuery.Session(&gorm.Session{}).
			Where("product_name = ? AND deleted_at IS NOT NULL", productName).
			Count(&summary.Cancelled)

		summaries = append(summaries, summary)
	}

	// Add a total summary row
	if len(summaries) > 0 {
		totalSummary := WorkOrderSummary{
			ProductName: "Total",
			TotalWO:     totalWorkOrders,
			Percentage:  100, // 100%
		}

		// Calculate totals
		for _, summary := range summaries {
			totalSummary.TargetQty += summary.TargetQty
			totalSummary.AchievedQty += summary.AchievedQty
			totalSummary.Pending += summary.Pending
			totalSummary.InProgress += summary.InProgress
			totalSummary.Completed += summary.Completed
			totalSummary.Cancelled += summary.Cancelled
		}

		// Calculate overall achievement percentage
		if totalSummary.TargetQty > 0 {
			totalSummary.Achievement = int64(float64(totalSummary.AchievedQty) / float64(totalSummary.TargetQty) * 100)
		}

		summaries = append(summaries, totalSummary)
	}

	return c.Status(fiber.StatusOK).JSON(SummaryResponse{
		Error:   false,
		Summary: summaries,
	})
}
