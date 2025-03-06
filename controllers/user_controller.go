package controllers

import (
	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/models"
	"github.com/gofiber/fiber/v2"
)

// OperatorResponse represents a response containing a list of operators
type OperatorResponse struct {
	Error     bool          `json:"error"`
	Operators []models.User `json:"operators"`
}

// @Summary Get all operators
// @Description Get a list of all operators in the system
// @Tags operators
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} OperatorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /operators [get]
func GetOperators(c *fiber.Ctx) error {
	// Fetch all users with operator role from database
	var operators []models.User
	result := database.DB.Where("role = ?", models.RoleOperator).Find(&operators)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching operators",
		})
	}

	// Return operators list
	return c.Status(fiber.StatusOK).JSON(OperatorResponse{
		Error:     false,
		Operators: operators,
	})
}
