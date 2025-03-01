package controllers

import (
	"fmt"
	"time"

	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

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

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid request body",
		})
	}

	fmt.Println(c.BodyParser(&req))
	fmt.Println(req)
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
		query = query.Where("work_order_number LIKE ? OR product_name LIKE ?",
			"%"+search+"%", "%"+search+"%")
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

	// Check if operator exists if operator ID is provided
	if req.OperatorID != 0 {
		var operator models.User
		result := database.DB.Where("id = ? AND role = ?", req.OperatorID, models.RoleOperator).First(&operator)
		if result.Error != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: true,
				Msg:   "Operator not found",
			})
		}
		workOrder.OperatorID = req.OperatorID
	}

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
		oldStatus := workOrder.Status
		workOrder.Status = req.Status

		// Create status history if status changed
		if oldStatus != req.Status {
			statusHistory := models.WorkOrderStatusHistory{
				WorkOrderID: workOrder.ID,
				Status:      req.Status,
				Quantity:    workOrder.Quantity,
			}

			if err := database.DB.Create(&statusHistory).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
					Error: true,
					Msg:   "Error creating status history",
				})
			}
		}
	}

	// Save work order to database
	if err := database.DB.Save(&workOrder).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error updating work order",
		})
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

	// Check if user is the assigned operator or a production manager
	if role == models.RoleOperator && workOrder.OperatorID != userID {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: true,
			Msg:   "You are not assigned to this work order",
		})
	}

	// Validate status transition
	if !isValidStatusTransition(workOrder.Status, req.Status) {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid status transition",
		})
	}

	// Update work order status
	workOrder.Status = req.Status

	// Update quantity if provided
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

	// Create status history
	statusHistory := models.WorkOrderStatusHistory{
		WorkOrderID: workOrder.ID,
		Status:      req.Status,
		Quantity:    workOrder.Quantity,
	}

	if err := database.DB.Create(&statusHistory).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error creating status history",
		})
	}

	// Return updated work order
	return c.Status(fiber.StatusOK).JSON(WorkOrderResponse{
		Error:     false,
		WorkOrder: workOrder,
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
