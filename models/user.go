package models

import (
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Role type for user roles
type Role string

const (
	// RoleProductionManager represents the Production Manager role
	RoleProductionManager Role = "production_manager"
	// RoleOperator represents the Operator role
	RoleOperator Role = "operator"
)

// User represents a user in the system
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"size:100;not null" json:"-"` // Password is not exposed in JSON
	Role      Role           `gorm:"size:20;not null;index" json:"role"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeSave is a GORM hook that hashes the password before saving
func (u *User) BeforeSave(tx *gorm.DB) error {
	if u.Password == "" {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword compares the provided password with the stored hash
func (u *User) CheckPassword(password string) error {
	log.Println(u.Password)
	log.Println(password)
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}
