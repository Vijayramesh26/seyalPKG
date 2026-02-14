package models

import (
	"time"
)

type Customer struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"size:100;not null" json:"name"`
	Mobile          string    `gorm:"size:15;unique;not null" json:"mobile"`
	Address         string    `gorm:"type:text" json:"address"`
	WhatsappOptIn   bool      `gorm:"default:false" json:"whatsapp_opt_in"`
	DiscountPercent float64   `gorm:"type:decimal(5,2);default:0.00" json:"discount_percent"`
	CreatedAt       time.Time `json:"created_at"`
}

type CustomerOrder struct {
	ID             uint        `gorm:"primaryKey" json:"id"`
	OrderNo        string      `gorm:"size:50;unique;not null" json:"order_no"`
	CustomerID     uint        `json:"customer_id"`
	Customer       Customer    `gorm:"foreignKey:CustomerID" json:"customer"`
	OrderDate      time.Time   `gorm:"default:CURRENT_TIMESTAMP" json:"order_date"`
	Status         string      `gorm:"type:enum('PENDING', 'COMPLETED', 'CANCELLED');default:'PENDING'" json:"status"`
	TotalEstimated float64     `gorm:"type:decimal(10,2)" json:"total_estimated"`
	Items          []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
}

type OrderItem struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	OrderID   uint    `json:"order_id"`
	ProductID uint    `json:"product_id"`
	Product   Product `gorm:"foreignKey:ProductID" json:"product"`
	Quantity  int     `json:"quantity"`
}

type Discount struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"size:50" json:"name"`
	Percentage float64   `gorm:"type:decimal(5,2);not null" json:"percentage"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type DiscountRule struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	MinAmount  float64 `json:"min_amount"`
	MaxAmount  float64 `json:"max_amount"` // 0 means infinity
	Percentage float64 `json:"percentage"`
	IsActive   bool    `json:"is_active" gorm:"default:true"`
}
