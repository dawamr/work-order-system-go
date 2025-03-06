package services

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/models"
	"gorm.io/gorm"
)

// AuditLogService handles audit logging operations
type AuditLogService struct{}

// CreateLog creates a new audit log entry
func (s *AuditLogService) CreateLog(userID uint, action models.ActionType, entityType string, entityID uint, oldValues, newValues interface{}, note string) error {
	var oldValuesJSON, newValuesJSON models.JSON

	// Get user data
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("error fetching user data: %v", err)
	}

	if oldValues != nil {
		// Extract only the changed fields
		changes := s.GetChangedFields(oldValues, newValues)
		if len(changes) > 0 {
			// Format old values
			oldData := make(map[string]interface{})
			for field, value := range changes {
				if changeMap, ok := value.(map[string]interface{}); ok {
					oldData[field] = changeMap["old"]
				}
			}

			data, err := json.Marshal(oldData)
			if err != nil {
				return fmt.Errorf("error marshaling old values: %v", err)
			}
			oldValuesJSON = models.JSON(data)
		}
	}

	if newValues != nil {
		// Format new values
		changes := s.GetChangedFields(oldValues, newValues)
		if len(changes) > 0 {
			// Format new values
			newData := make(map[string]interface{})
			for field, value := range changes {
				if changeMap, ok := value.(map[string]interface{}); ok {
					newData[field] = changeMap["new"]
				}
			}

			data, err := json.Marshal(newData)
			if err != nil {
				return fmt.Errorf("error marshaling new values: %v", err)
			}
			newValuesJSON = models.JSON(data)
		}
	}

	log := models.AuditLog{
		UserID:     userID,
		User:       user,  // Include complete user data
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		OldValues:  oldValuesJSON,
		NewValues:  newValuesJSON,
		Note:       note,
	}

	if err := database.DB.Create(&log).Error; err != nil {
		return fmt.Errorf("error creating audit log: %v", err)
	}

	return nil
}

// GetChangedFields compares old and new structs and returns changed fields
func (s *AuditLogService) GetChangedFields(old, new interface{}) map[string]interface{} {
	changes := make(map[string]interface{})

	if old == nil || new == nil {
		return changes
	}

	oldVal := reflect.ValueOf(old)
	newVal := reflect.ValueOf(new)

	// Handle pointer types
	if oldVal.Kind() == reflect.Ptr {
		oldVal = oldVal.Elem()
	}
	if newVal.Kind() == reflect.Ptr {
		newVal = newVal.Elem()
	}

	// Ensure both values are structs
	if oldVal.Kind() != reflect.Struct || newVal.Kind() != reflect.Struct {
		return changes
	}

	// Fields to ignore in audit log
	ignoredFields := map[string]bool{
		"updated_at": true,
		"created_at": true,
		"deleted_at": true,
	}

	for i := 0; i < oldVal.NumField(); i++ {
		field := oldVal.Type().Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip if field should not be logged
		if field.Tag.Get("audit") == "-" {
			continue
		}

		// Get JSON field name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Remove omitempty and other options from json tag
		jsonTag = strings.Split(jsonTag, ",")[0]

		// Skip ignored fields
		if ignoredFields[jsonTag] {
			continue
		}

		oldField := oldVal.Field(i)
		newField := newVal.Field(i)

		// Handle special types (time.Time, etc)
		oldInterface := s.normalizeValue(oldField.Interface())
		newInterface := s.normalizeValue(newField.Interface())

		// Compare values
		if !reflect.DeepEqual(oldInterface, newInterface) {
			// For time.Time, compare normalized values to avoid timezone issues
			if _, ok := oldField.Interface().(time.Time); ok {
				oldTime, _ := time.Parse(time.RFC3339, oldInterface.(string))
				newTime, _ := time.Parse(time.RFC3339, newInterface.(string))

				// Compare times in UTC
				if oldTime.UTC().Equal(newTime.UTC()) {
					continue
				}
			}

			changes[jsonTag] = map[string]interface{}{
				"old": oldInterface,
				"new": newInterface,
			}
		}
	}

	return changes
}

// normalizeValue handles special types for comparison
func (s *AuditLogService) normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case time.Time:
		return val.Format(time.RFC3339)
	case *time.Time:
		if val == nil {
			return nil
		}
		return val.Format(time.RFC3339)
	case gorm.DeletedAt:
		if val.Valid {
			return val.Time.Format(time.RFC3339)
		}
		return nil
	default:
		return v
	}
}
