package controllers

import (
	"log"

	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/middleware"
	"github.com/dawamr/work-order-system-go/models"
	"github.com/gofiber/fiber/v2"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Error bool `json:"error"`
	Token string `json:"token"`
	User  struct {
		ID       uint        `json:"id"`
		Username string      `json:"username"`
		Role     models.Role `json:"role"`
	} `json:"user"`
}

// RegisterRequest represents the register request body
type RegisterRequest struct {
	Username string      `json:"username" validate:"required,min=3,max=50"`
	Password string      `json:"password" validate:"required,min=6"`
	Role     models.Role `json:"role" validate:"required,oneof=production_manager operator"`
}

// RegisterResponse represents the register response
type RegisterResponse struct {
	Error bool `json:"error"`
	Token string `json:"token"`
	User  struct {
		ID       uint        `json:"id"`
		Username string      `json:"username"`
		Role     models.Role `json:"role"`
	} `json:"user"`
}

// ErrorResponse represents the error response
type ErrorResponse struct {
	Error bool   `json:"error"`
	Msg   string `json:"msg"`
}

// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
func Login(c *fiber.Ctx) error {
	// Parse request body
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid request body",
		})
	}

	// Find user by username
	var user models.User
	result := database.DB.Where("username = ?", req.Username).First(&user)
	log.Println(result.Error != nil)
	if result.Error != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid credentials",
		})
	}

	// Check password
	if err := user.CheckPassword(req.Password); err != nil {
		log.Println(err)
		log.Println(user.Password)
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid credentials",
		})
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(&user)
	log.Println(token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error generating token",
		})
	}

	// Return token and user info
	return c.Status(fiber.StatusOK).JSON(LoginResponse{
		Error: false,
		Token: token,
		User: struct {
			ID       uint        `json:"id"`
			Username string      `json:"username"`
			Role     models.Role `json:"role"`
		}{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

// @Summary Register new user
// @Description Register a new user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func Register(c *fiber.Ctx) error {
	// Parse request body
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Invalid request body",
		})
	}

	// Check if username already exists
	var existingUser models.User
	result := database.DB.Where("username = ?", req.Username).First(&existingUser)
	if result.Error == nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: true,
			Msg:   "Username already exists",
		})
	}

	// Create new user
	user := models.User{
		Username: req.Username,
		Password: req.Password,
		Role:     req.Role,
	}

	// Save user to database
	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error creating user",
		})
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error generating token",
		})
	}

	// Return token and user info
	return c.Status(fiber.StatusCreated).JSON(RegisterResponse{
		Error: false,
		Token: token,
		User: struct {
			ID       uint        `json:"id"`
			Username string      `json:"username"`
			Role     models.Role `json:"role"`
		}{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}
