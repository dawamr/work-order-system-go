package models

import (
	"errors"
	"fmt"
	"time"

	"database/sql/driver"

	"gorm.io/gorm"
)

// ActionType represents the type of action performed
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionUpdate ActionType = "update"
	ActionDelete ActionType = "delete"
	ActionCustom ActionType = "custom"
)

// AuditLog represents a log entry for model changes
type AuditLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint          `gorm:"not null" json:"user_id"`
	User       User          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:UserID;references:ID" json:"user"`
	Action     ActionType    `gorm:"size:20;not null" json:"action"`
	EntityID   uint         `gorm:"not null" json:"entity_id"`
	EntityType string       `gorm:"size:50;not null" json:"entity_type"`
	OldValues  JSON         `gorm:"type:jsonb" json:"old_values,omitempty"`
	NewValues  JSON         `gorm:"type:jsonb" json:"new_values,omitempty"`
	Note       string       `gorm:"type:text" json:"note,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// JSON is a wrapper for handling JSON data in GORM
type JSON []byte

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*j = append((*j)[0:0], v...)
	case string:
		*j = append((*j)[0:0], []byte(v)...)
	default:
		return fmt.Errorf("invalid type for JSON: %T", value)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// MarshalJSON implements json.Marshaler interface
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSON: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}
