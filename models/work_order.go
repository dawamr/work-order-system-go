package models

import (
	"time"

	"gorm.io/gorm"
)

// WorkOrderStatus type for work order status
type WorkOrderStatus string

const (
	// StatusPending represents a pending work order
	StatusPending WorkOrderStatus = "pending"
	// StatusInProgress represents a work order in progress
	StatusInProgress WorkOrderStatus = "in_progress"
	// StatusCompleted represents a completed work order
	StatusCompleted WorkOrderStatus = "completed"
)

// WorkOrder represents a work order in the system
type WorkOrder struct {
	ID                 uint            `gorm:"primaryKey" json:"id"`
	WorkOrderNumber    string          `gorm:"size:20;uniqueIndex;not null" json:"work_order_number"`
	ProductName        string          `gorm:"size:100;not null" json:"product_name"`
	Quantity           int             `gorm:"not null" json:"quantity"`
	ProductionDeadline time.Time       `json:"production_deadline"`
	Status             WorkOrderStatus `gorm:"size:20;not null;default:'pending'" json:"status"`
	OperatorID         uint            `json:"operator_id"`
	Operator           User            `gorm:"foreignKey:OperatorID" json:"operator"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	DeletedAt          gorm.DeletedAt  `gorm:"index" json:"-"`
}

// WorkOrderProgress represents progress updates for a work order
type WorkOrderProgress struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	WorkOrderID      uint           `gorm:"not null" json:"work_order_id"`
	WorkOrder        WorkOrder      `gorm:"foreignKey:WorkOrderID" json:"work_order"`
	ProgressDesc     string         `gorm:"size:500;not null" json:"progress_description"`
	ProgressQuantity int            `json:"progress_quantity"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// WorkOrderStatusHistory represents the history of status changes for a work order
type WorkOrderStatusHistory struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	WorkOrderID uint            `gorm:"not null" json:"work_order_id"`
	WorkOrder   WorkOrder       `gorm:"foreignKey:WorkOrderID" json:"work_order"`
	Status      WorkOrderStatus `gorm:"size:20;not null" json:"status"`
	Quantity    int             `json:"quantity"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `gorm:"index" json:"-"`
}
