package handler

import (
	"net/http"

	"billing-app/internal/models"
	"billing-app/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InventoryHandler struct{}

func (h *InventoryHandler) ListProducts(c *gin.Context) {
	var products []models.Product
	if err := database.DB.Preload("Brand").Preload("Category").Where("is_active = ?", true).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	c.JSON(http.StatusOK, products)
}

type CreateProductRequest struct {
	Name              string  `json:"name" binding:"required"`
	BrandName         string  `json:"brand_name" binding:"required"`
	CategoryID        *uint   `json:"category_id"`
	Description       string  `json:"description"`
	UnitPrice         float64 `json:"unit_price" binding:"required"`
	LowStockThreshold int     `json:"low_stock_threshold"`
	Barcode           string  `json:"barcode"`
	OpeningStock      int     `json:"opening_stock"`
}

func (h *InventoryHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find or Create Brand
	var brand models.Brand
	if err := database.DB.FirstOrCreate(&brand, models.Brand{Name: req.BrandName}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process brand"})
		return
	}

	userID := c.GetUint("userID")

	tx := database.DB.Begin()

	product := models.Product{
		Name:              req.Name,
		BrandID:           brand.ID,
		CategoryID:        req.CategoryID,
		Description:       req.Description,
		UnitPrice:         req.UnitPrice,
		LowStockThreshold: req.LowStockThreshold,
		CurrentStock:      req.OpeningStock, // Set initial stock
		Barcode:           req.Barcode,
		IsActive:          true,
	}

	if err := tx.Create(&product).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	// Create Stock Entry if Opening Stock is provided
	if req.OpeningStock > 0 {
		entry := models.StockEntry{
			ProductID:     product.ID,
			QuantityAdded: req.OpeningStock,
			AddedBy:       userID,
		}
		if err := tx.Create(&entry).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log opening stock"})
			return
		}
	}

	tx.Commit()

	c.JSON(http.StatusCreated, product)
}

type AddStockRequest struct {
	ProductID int `json:"product_id" binding:"required"`
	Quantity  int `json:"quantity" binding:"required"`
}

func (h *InventoryHandler) AddStock(c *gin.Context) {
	var req AddStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")

	// Start Transaction
	tx := database.DB.Begin()

	// Update Product Stock
	if err := tx.Model(&models.Product{}).Where("id = ?", req.ProductID).Update("current_stock", gorm.Expr("current_stock + ?", req.Quantity)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock"})
		return
	}

	// Create Stock Entry
	entry := models.StockEntry{
		ProductID:     uint(req.ProductID),
		QuantityAdded: req.Quantity,
		AddedBy:       userID,
	}

	if err := tx.Create(&entry).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log stock entry"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Stock added successfully"})
}

func (h *InventoryHandler) ListBrands(c *gin.Context) {
	var brands []models.Brand
	if err := database.DB.Find(&brands).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch brands"})
		return
	}
	c.JSON(http.StatusOK, brands)
}

func (h *InventoryHandler) GetLowStockAlerts(c *gin.Context) {
	var products []models.Product
	if err := database.DB.Preload("Brand").Preload("Category").Where("current_stock <= low_stock_threshold AND is_active = ?", true).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}
	c.JSON(http.StatusOK, products)
}

// Category Handlers
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func (h *InventoryHandler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := models.Category{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := database.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, category)
}

func (h *InventoryHandler) ListCategories(c *gin.Context) {
	var categories []models.Category
	if err := database.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	c.JSON(http.StatusOK, categories)
}
