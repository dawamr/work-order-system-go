package controllers

import (
	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/models"
	"github.com/gofiber/fiber/v2"
)

type AuditLogListResponse struct {
	Error     bool            `json:"error"`
	AuditLogs []models.AuditLog `json:"audit_logs"`
	Pagination Pagination     `json:"pagination"`
}

// GetAuditLogs returns a paginated list of audit logs
func GetAuditLogs(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	entityType := c.Query("entity_type")
	entityID := c.QueryInt("entity_id", 0)
	action := c.Query("action")

	offset := (page - 1) * limit

	// Build query with proper User preloading
	query := database.DB.Model(&models.AuditLog{}).
		Preload("User"). // Use Preload instead of Joins
		Order("created_at DESC")

	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
	if entityID > 0 {
		query = query.Where("entity_id = ?", entityID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}

	var count int64
	query.Count(&count)

	var auditLogs []models.AuditLog
	if err := query.Offset(offset).Limit(limit).Find(&auditLogs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: true,
			Msg:   "Error fetching audit logs",
		})
	}

	return c.JSON(AuditLogListResponse{
		Error:     false,
		AuditLogs: auditLogs,
		Pagination: Pagination{
			Total: count,
			Page:  page,
			Limit: limit,
			Pages: (count + int64(limit) - 1) / int64(limit),
		},
	})
}
