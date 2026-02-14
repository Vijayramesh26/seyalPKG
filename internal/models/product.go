package models

import (
	"time"

	"gorm.io/gorm"
)

type Brand struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;unique;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Products  []Product `json:"-"`
}

type Category struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;unique;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	Products    []Product `json:"-"`
}

type Product struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Name              string         `gorm:"size:150;not null" json:"name"`
	BrandID           uint           `json:"brand_id"`
	Brand             Brand          `gorm:"foreignKey:BrandID" json:"brand"`
	CategoryID        *uint          `json:"category_id"`
	Category          *Category      `gorm:"foreignKey:CategoryID" json:"category"`
	Description       string         `gorm:"type:text" json:"description"`
	UnitPrice         float64        `gorm:"type:decimal(10,2);not null" json:"unit_price"`
	CurrentStock      int            `gorm:"default:0" json:"current_stock"`
	LowStockThreshold int            `gorm:"default:10" json:"low_stock_threshold"`
	Barcode           string         `gorm:"size:50;index" json:"barcode"`
	IsActive          bool           `gorm:"default:true" json:"is_active"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

type StockEntry struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProductID     uint      `json:"product_id"`
	Product       Product   `gorm:"foreignKey:ProductID" json:"product"`
	QuantityAdded int       `json:"quantity_added"`
	AddedBy       uint      `json:"added_by"`
	User          User      `gorm:"foreignKey:AddedBy" json:"user"`
	EntryDate     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"entry_date"`
}
