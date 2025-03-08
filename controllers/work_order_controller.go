package controllers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/models"
	"github.com/dawamr/work-order-system-go/services"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var auditService = services.AuditLogService{}

// CreateWorkOrderRequest represents the create work order request body
type CreateWorkOrderRequest struct {
	ProductName        string    `json:"product_name" validate:"required"`
	Quantity           int       `json:"quantity" validate:"required,min=1"`
	ProductionDeadline time.Time `json:"production_deadline" validate:"required"`
	OperatorID         uint      `json:"operator_id" validate:"required"`
}

// UpdateWorkOrderRequest represents the update work order request body
type UpdateWorkOrderRequest struct {
	ProductName        string             `json:"product_name"`
	Quantity           int                `json:"quantity" validate:"omitempty,min=1"`
	ProductionDeadline time.Time          `json:"production_deadline"`
	Status             models.WorkOrderStatus `json:"status"`
	OperatorID         uint               `json:"operator_id"`
}

// UpdateWorkOrderStatusRequest represents the update work order status request body
type UpdateWorkOrderStatusRequest struct {
	Status   models.WorkOrderStatus `json:"status" validate:"required,oneof=pending in_progress completed"`
	Quantity int                `json:"quantity" validate:"omitempty,min=0"`
}

// WorkOrderResponse represents a work order response
type WorkOrderResponse struct {
	Error     bool         `json:"error"`
	WorkOrder models.WorkOrder `json:"work_order"`
}

// WorkOrderListResponse represents a paginated list of work orders
type WorkOrderListResponse struct {
	Error      bool             `json:"error"`
	WorkOrders []models.WorkOrder `json:"work_orders"`
	Pagination struct {
		Total int64 `json:"total"`
		Page  int   `json:"page"`
		Limit int   `json:"limit"`
		Pages int64 `json:"pages"`
	} `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Pages int64 `json:"pages"`
}

// CreateWorkOrderLogRequest represents the request body for creating a work order log
type CreateWorkOrderLogRequest struct {
	Note   string              `json:"note" validate:"required"`
	Status models.WorkOrderStatus `json:"status,omitempty"`
}

// GenerateWorkOrderNumber generates a unique work order number
func GenerateWorkOrderNumber() string {
	// Format: WO-YYYYMMDD-XXX
	date := time.Now().Format("20060102")

	// Get the latest work order number for today
	var latestWorkOrder models.WorkOrder
	result := database.DB.Where("work_order_number LIKE ?", fmt.Sprintf("WO-%s-%%", date)).
		Order("work_order_number DESC").
		First(&latestWorkOrder)

	var sequence int
	if result.Error != nil {
		// No work orders for today yet
		sequence = 1
	} else {
		// Extract the sequence number from the latest work order number
		fmt.Sscanf(latestWorkOrder.WorkOrderNumber, fmt.Sprintf("WO-%s-%%03d", date), &sequence)
		sequence++
	}

	// Format the work order number
	return fmt.Sprintf("WO-%s-%03d", date, sequence)
}

// @Summary Create work order
// @Description Create a new work order (Production Manager only)
// @Tags work-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateWorkOrderRequest true "Work order details"
// @Success 201 {object} WorkOrderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /work-orders [post]
func CreateWorkOrder(c *fiber.Ctx) error {

	// Only Production Manager can create work orders
	role := c.Locals("role").(models.Role)
	if role != models.RoleProductionManager {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "Only Production Manager can create work orders",
		})
	}

	// Parse request body
	var req CreateWorkOrderRequest

	fmt.Println(c.BodyParser(&req))
	fmt.Println(req)
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid request body",
		})
	}


	// Check if operator exists
	var operator models.User
	result := database.DB.Where("id = ? AND role = ?", req.OperatorID, models.RoleOperator).First(&operator)
	if result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Operator not found",
		})
	}

	// Generate work order number
	workOrderNumber := GenerateWorkOrderNumber()

	// Create work order
	workOrder := models.WorkOrder{
		WorkOrderNumber:    workOrderNumber,
		ProductName:        req.ProductName,
		Quantity:           req.Quantity,
		ProductionDeadline: req.ProductionDeadline,
		Status:             models.StatusPending,
		OperatorID:         req.OperatorID,
	}

	// Save work order to database
	if err := database.DB.Create(&workOrder).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error creating work order",
		})
	}

	// Create initial status history
	statusHistory := models.WorkOrderStatusHistory{
		WorkOrderID: workOrder.ID,
		Status:      models.StatusPending,
		Quantity:    0,
	}

	if err := database.DB.Create(&statusHistory).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error creating status history",
		})
	}

	// Return work order
	return c.Status(fiber.StatusCreated).JSON(WorkOrderResponse{
		Error:     false,
		WorkOrder: workOrder,
	})
}

// @Summary Get all work orders
// @Description Get a paginated list of all work orders (Production Manager only)
// @Tags work-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param status query string false "Filter by status (pending/in_progress/completed)"
// @Success 200 {object} WorkOrderListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /work-orders [get]
func GetWorkOrders(c *fiber.Ctx) error {
	// Get query parameters
	status := c.Query("status")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	operatorID := c.QueryInt("operator_id", 0) // filter by work_orders.operator_id
	search := c.Query("search") // search by work_orders.work_order_number, work_orders.product_name
	deadline := c.Query("deadline") // filter by work_orders.production_deadline

	// Calculate offset
	offset := (page - 1) * limit

	// Build query
	query := database.DB.Model(&models.WorkOrder{}).Preload("Operator")

	// Apply status filter if provided
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Apply operator filter if provided
	if operatorID > 0 {
		query = query.Where("operator_id = ?", operatorID)
	}

	// Apply search if provided
	if search != "" {
		// search with UPPERCASE
		search = strings.ToUpper(search)
		if strings.HasPrefix(search, "WO-") {
			query = query.Where("UPPER(work_order_number) LIKE ?", "%"+search+"%")
		} else {
			query = query.Where("UPPER(product_name) LIKE ?", "%"+search+"%")
		}
	}

	// Apply deadline filter if provided
	if deadline != "" {
		// Assume deadline is in YYYY-MM-DD format
		query = query.Where("DATE(production_deadline) = ?", deadline)
	}

	// Get total count
	var count int64
	query.Count(&count)

	// Get work orders with pagination
	var workOrders []models.WorkOrder
	result := query.Offset(offset).Limit(limit).Order("work_order_number DESC").Find(&workOrders)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching work orders",
		})
	}

	// Return work orders with pagination info
	return c.Status(fiber.StatusOK).JSON(WorkOrderListResponse{
		Error:      false,
		WorkOrders: workOrders,
		Pagination: Pagination{
			Total:  count,
			Page:   page,
			Limit:  limit,
			Pages:  (count + int64(limit) - 1) / int64(limit),
		},
	})
}

// @Summary Get assigned work orders
// @Description Get work orders assigned to the current operator
// @Tags work-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param status query string false "Filter by status (pending/in_progress/completed)"
// @Success 200 {object} WorkOrderListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /work-orders/assigned [get]
func GetAssignedWorkOrders(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id").(uint)

	// Get query parameters
	status := c.Query("status")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	// Calculate offset
	offset := (page - 1) * limit

	// Build query
	query := database.DB.Model(&models.WorkOrder{}).Where("operator_id = ?", userID)

	// Apply status filter if provided
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	var count int64
	query.Count(&count)

	// Get work orders with pagination
	var workOrders []models.WorkOrder
	result := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&workOrders)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching work orders",
		})
	}

	// Return work orders with pagination info
	return c.Status(fiber.StatusOK).JSON(WorkOrderListResponse{
		Error:      false,
		WorkOrders: workOrders,
		Pagination: Pagination{
			Total:  count,
			Page:   page,
			Limit:  limit,
			Pages:  (count + int64(limit) - 1) / int64(limit),
		},
	})
}

// @Summary Get work order by ID
// @Description Get a work order by its ID
// @Tags work-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work order ID"
// @Success 200 {object} WorkOrderResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /work-orders/{id} [get]
func GetWorkOrderByID(c *fiber.Ctx) error {
	// Get work order ID from URL
	id := c.Params("id")

	// Get work order from database
	var workOrder models.WorkOrder
	result := database.DB.Preload("Operator").First(&workOrder, id)
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

	// Return work order
	return c.Status(fiber.StatusOK).JSON(WorkOrderResponse{
		Error:     false,
		WorkOrder: workOrder,
	})
}

// @Summary Update work order
// @Description Update a work order (Production Manager only)
// @Tags work-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work order ID"
// @Param request body UpdateWorkOrderRequest true "Work order update details"
// @Success 200 {object} WorkOrderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /work-orders/{id} [put]
func UpdateWorkOrder(c *fiber.Ctx) error {
	// Only Production Manager can update work orders
	role := c.Locals("role").(models.Role)
	if role != models.RoleProductionManager {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "Only Production Manager can update work orders",
		})
	}

	// Get work order ID from URL
	id := c.Params("id")

	// Parse request body
	var req UpdateWorkOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid request body",
		})
	}

	// Get work order from database
	var oldWorkOrder models.WorkOrder
	result := database.DB.First(&oldWorkOrder, id)
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

	// Buat salinan untuk audit log
	workOrder := oldWorkOrder

	// Update work order fields if provided
	if req.ProductName != "" {
		workOrder.ProductName = req.ProductName
	}
	if req.Quantity > 0 {
		workOrder.Quantity = req.Quantity
	}
	if !req.ProductionDeadline.IsZero() {
		workOrder.ProductionDeadline = req.ProductionDeadline
	}
	if req.Status != "" {
		workOrder.Status = req.Status
	}
	if req.OperatorID != 0 {
		workOrder.OperatorID = req.OperatorID
	}

	// Save work order to database
	if err := database.DB.Save(&workOrder).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error updating work order",
		})
	}

	// Create audit log after successful update
	userID := c.Locals("user_id").(uint)
	if err := auditService.CreateLog(
		userID,
		models.ActionUpdate,
		"WorkOrder",
		workOrder.ID,
		oldWorkOrder,  // old values
		workOrder,     // new values
		fmt.Sprintf("Work order %s updated", workOrder.WorkOrderNumber),
	); err != nil {
		log.Printf("Error creating audit log: %v", err)
	}

	// Return updated work order
	return c.Status(fiber.StatusOK).JSON(WorkOrderResponse{
		Error:     false,
		WorkOrder: workOrder,
	})
}

// @Summary Update work order status
// @Description Update a work order status (Operator only)
// @Tags work-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work order ID"
// @Param request body UpdateWorkOrderStatusRequest true "Status update details"
// @Success 200 {object} WorkOrderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /work-orders/{id}/status [put]
func UpdateWorkOrderStatus(c *fiber.Ctx) error {
	// Get user ID and role from context
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(models.Role)

	// Tambahkan validasi role
	if role == models.RoleOperator {
		// Check if user is the assigned operator
		var workOrder models.WorkOrder
		if err := database.DB.First(&workOrder, c.Params("id")).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: true,
				Msg:   "Work order not found",
			})
		}

		if workOrder.OperatorID != userID {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Error: true,
				Msg:   "You are not assigned to this work order",
			})
		}
	} else if role != models.RoleProductionManager {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "Unauthorized to update work order status",
		})
	}

	// Get work order ID from URL
	id := c.Params("id")

	// Parse request body
	var req UpdateWorkOrderStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid request body",
		})
	}

	// Get work order from database
	var oldWorkOrder models.WorkOrder
	result := database.DB.First(&oldWorkOrder, id)
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

	// Buat salinan untuk update
	workOrder := oldWorkOrder

	// Update work order status
	workOrder.Status = req.Status
	if req.Quantity > 0 {
		workOrder.Quantity = req.Quantity
	}

	// Save work order to database
	if err := database.DB.Save(&workOrder).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error updating work order status",
		})
	}

	// Create audit log after successful update
	if err := auditService.CreateLog(
		userID,
		models.ActionUpdate,
		"WorkOrder",
		workOrder.ID,
		oldWorkOrder,  // old values
		workOrder,     // new values
		fmt.Sprintf("Work order %s status updated from %s to %s",
			workOrder.WorkOrderNumber,
			oldWorkOrder.Status,
			workOrder.Status),
	); err != nil {
		log.Printf("Error creating audit log: %v", err)
	}

	// Return updated work order
	return c.Status(fiber.StatusOK).JSON(WorkOrderResponse{
		Error:     false,
		WorkOrder: workOrder,
	})
}

// @Summary Delete work order
// @Description Delete a work order (Production Manager only)
// @Tags work-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work order ID"
// @Success 200 {object} WorkOrderResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /work-orders/{id} [delete]
func DeleteWorkOrder(c *fiber.Ctx) error {
	// Only Production Manager can delete work orders
	role := c.Locals("role").(models.Role)
	if role != models.RoleProductionManager {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "Only Production Manager can delete work orders",
		})
	}

	// Get work order ID from URL
	id := c.Params("id")

	// Get work order from database
	var workOrder models.WorkOrder
	result := database.DB.First(&workOrder, id)
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

	// Delete work order from database
	if err := database.DB.Delete(&workOrder).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error deleting work order",
		})
	}

	// Create audit log after successful delete
	userID := c.Locals("user_id").(uint)
	if err := auditService.CreateLog(
		userID,
		models.ActionDelete,
		"WorkOrder",
		workOrder.ID,
		workOrder,  // capture state before deletion
		nil,        // no new values for deletion
		fmt.Sprintf("Work order %s deleted", workOrder.WorkOrderNumber),
	); err != nil {
		log.Printf("Error creating audit log: %v", err)
	}

	// Return success response
	return c.Status(fiber.StatusOK).JSON(WorkOrderResponse{
		Error:     false,
		WorkOrder: workOrder,
	})
}

func GetWorkOrderLogs(c *fiber.Ctx) error {
	id := c.Params("id")

	var logs []models.AuditLog
	if err := database.DB.
		Preload("User"). // Add preload for User
		Where("entity_type = ? AND entity_id = ?", "WorkOrder", id).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching audit logs",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"error": false,
		"logs":  logs,
	})
}


// CreateWorkOrderLog creates a custom log entry for a work order
func CreateWorkOrderLog(c *fiber.Ctx) error {
	id := c.Params("id")

	var req CreateWorkOrderLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid request body",
		})
	}

	// Validate required fields
	if req.Note == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Note is required",
		})
	}

	// Get work order from database
	var workOrder models.WorkOrder
	result := database.DB.First(&workOrder, id)
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

	// If status is provided, validate the transition
	if req.Status != "" {
		if !isValidStatusTransition(workOrder.Status, req.Status) {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: true,
				Msg:   "Invalid status transition",
			})
		}

		// Create a copy of work order for new values
		newWorkOrder := workOrder
		newWorkOrder.Status = req.Status

		// Create audit log with status change
		userID := c.Locals("user_id").(uint)
		if err := auditService.CreateLog(
			userID,
			models.ActionCustom,
			"WorkOrder",
			workOrder.ID,
			workOrder,     // old state
			newWorkOrder,  // new state with updated status
			req.Note,      // use provided note
		); err != nil {
			log.Printf("Error creating audit log: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error: true,
				Msg:   "Error creating work order log",
			})
		}

		// Update work order status
		workOrder.Status = req.Status
		if err := database.DB.Save(&workOrder).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error: true,
				Msg:   "Error updating work order status",
			})
		}
	} else {
		// Create audit log without status change
		userID := c.Locals("user_id").(uint)
		if err := auditService.CreateLog(
			userID,
			models.ActionCustom,
			"WorkOrder",
			workOrder.ID,
			nil,         // no old values
			nil,         // no new values
			req.Note,    // use provided note
		); err != nil {
			log.Printf("Error creating audit log: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error: true,
				Msg:   "Error creating work order log",
			})
		}
	}

	// Return success response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"error": false,
		"message": "Work order log created successfully",
		"work_order": workOrder,
	})
}

// Helper function to validate status transitions
func isValidStatusTransition(from, to models.WorkOrderStatus) bool {
	switch from {
	case models.StatusPending:
		return to == models.StatusInProgress
	case models.StatusInProgress:
		return to == models.StatusCompleted
	default:
		return false
	}
}
