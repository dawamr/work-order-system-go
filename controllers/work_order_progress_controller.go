package controllers

import (
	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CreateProgressRequest represents the create progress request body
type CreateProgressRequest struct {
	ProgressDesc     string `json:"progress_description" validate:"required"`
	ProgressQuantity int    `json:"progress_quantity" validate:"required,min=0"`
}

// ProgressResponse represents a progress entry response
type ProgressResponse struct {
	Error    bool                    `json:"error"`
	Progress models.WorkOrderProgress `json:"progress"`
}

// ProgressListResponse represents a list of progress entries
type ProgressListResponse struct {
	Error    bool                      `json:"error"`
	Progress []models.WorkOrderProgress `json:"progress"`
}

// StatusHistoryResponse represents a list of status history entries
type StatusHistoryResponse struct {
	Error   bool                          `json:"error"`
	History []models.WorkOrderStatusHistory `json:"history"`
}

// CreateWorkOrderProgress creates a new progress entry for a work order
// @Summary Create progress entry
// @Description Create a new progress entry for a work order
// @Tags progress
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work order ID"
// @Param request body CreateProgressRequest true "Progress details"
// @Success 201 {object} ProgressResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /work-orders/{id}/progress [post]
func CreateWorkOrderProgress(c *fiber.Ctx) error {
	// Get user ID and role from context
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(models.Role)

	// Get work order ID from URL
	workOrderID := c.Params("id")

	// Parse request body
	var req CreateProgressRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid request body",
		})
	}

	// Get work order from database
	var workOrder models.WorkOrder
	result := database.DB.First(&workOrder, workOrderID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: true,
				Msg:   "Work order not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching work order",
		})
	}

	// Check if user is the assigned operator or a production manager
	if role == models.RoleOperator && workOrder.OperatorID != userID {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "You are not assigned to this work order",
		})
	}

	// Check if work order is in progress
	if workOrder.Status != models.StatusInProgress {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Work order must be in progress to add progress updates",
		})
	}

	// Create progress entry
	progress := models.WorkOrderProgress{
		WorkOrderID:      workOrder.ID,
		ProgressDesc:     req.ProgressDesc,
		ProgressQuantity: req.ProgressQuantity,
	}

	// Save progress to database
	if err := database.DB.Create(&progress).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error creating progress entry",
		})
	}

	// Return progress
	return c.Status(fiber.StatusCreated).JSON(ProgressResponse{
		Error:    false,
		Progress: progress,
	})
}

// GetWorkOrderProgress gets all progress entries for a work order
// @Summary Get work order progress
// @Description Get all progress entries for a work order
// @Tags progress
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work order ID"
// @Success 200 {object} ProgressListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /work-orders/{id}/progress [get]
func GetWorkOrderProgress(c *fiber.Ctx) error {
	// Get user ID and role from context
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(models.Role)

	// Get work order ID from URL
	workOrderID := c.Params("id")

	// Get work order from database
	var workOrder models.WorkOrder
	result := database.DB.First(&workOrder, workOrderID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: true,
				Msg:   "Work order not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching work order",
		})
	}

	// Check if user is the assigned operator or a production manager
	if role == models.RoleOperator && workOrder.OperatorID != userID {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "You are not assigned to this work order",
		})
	}

	// Get progress entries
	var progress []models.WorkOrderProgress
	result = database.DB.Where("work_order_id = ?", workOrder.ID).Order("created_at DESC").Find(&progress)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching progress entries",
		})
	}

	// Return progress entries
	return c.Status(fiber.StatusOK).JSON(ProgressListResponse{
		Error:    false,
		Progress: progress,
	})
}

// GetWorkOrderStatusHistory gets the status history for a work order
// @Summary Get work order status history
// @Description Get the status history for a work order
// @Tags progress
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work order ID"
// @Success 200 {object} StatusHistoryResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /work-orders/{id}/history [get]
func GetWorkOrderStatusHistory(c *fiber.Ctx) error {
	// Get user ID and role from context
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(models.Role)

	// Get work order ID from URL
	workOrderID := c.Params("id")

	// Get work order from database
	var workOrder models.WorkOrder
	result := database.DB.First(&workOrder, workOrderID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: true,
				Msg:   "Work order not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching work order",
		})
	}

	// Check if user is the assigned operator or a production manager
	if role == models.RoleOperator && workOrder.OperatorID != userID {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "You are not assigned to this work order",
		})
	}

	// Get status history
	var history []models.WorkOrderStatusHistory
	result = database.DB.Where("work_order_id = ?", workOrder.ID).Order("created_at ASC").Find(&history)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching status history",
		})
	}

	// Return status history
	return c.Status(fiber.StatusOK).JSON(StatusHistoryResponse{
		Error:   false,
		History: history,
	})
}
