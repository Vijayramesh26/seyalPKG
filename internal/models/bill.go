package models

import (
	"time"
)

type Bill struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	BillNo         string     `gorm:"size:50;unique;not null" json:"bill_no"`
	OrderNo        string     `gorm:"size:50" json:"order_no"` // Optional reference
	BillDate       time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"bill_date"`
	CustomerID     *uint      `json:"customer_id"` // Nullable
	Customer       *Customer  `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	UserID         uint       `json:"user_id"`
	User           User       `gorm:"foreignKey:UserID" json:"user"`
	TotalAmount    float64    `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	DiscountAmount float64    `gorm:"type:decimal(10,2);default:0.00" json:"discount_amount"`
	GSTAmount      float64    `gorm:"type:decimal(10,2);default:0.00" json:"gst_amount"`
	NetPayable     float64    `gorm:"type:decimal(10,2);not null" json:"net_payable"`
	PaymentMode    string     `gorm:"type:enum('CASH', 'ONLINE', 'CARD');default:'CASH'" json:"payment_mode"`
	Status         string     `gorm:"type:enum('PAID', 'CANCELLED');default:'PAID'" json:"status"`
	Items          []BillItem `gorm:"foreignKey:BillID" json:"items"`
}

type BillItem struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	BillID    uint    `json:"bill_id"`
	ProductID uint    `json:"product_id"`
	Product   Product `gorm:"foreignKey:ProductID" json:"product"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `gorm:"type:decimal(10,2);not null" json:"unit_price"`
	Total     float64 `gorm:"type:decimal(10,2);not null" json:"total"`
}
