package handler

import (
	"fmt"
	"net/http"
	"time"

	"billing-app/config"
	"billing-app/internal/models"
	"billing-app/pkg/database"

	"github.com/gin-gonic/gin"
)

type BillingHandler struct{}

// Helper to generate bill number: B-YYYYMMDD-SEQ
func generateBillNo() string {
	prefix := config.AppConfig.Defaults.BillerPrefix
	dateStr := time.Now().Format("20060102")

	var lastBill models.Bill
	database.DB.Order("id desc").First(&lastBill)

	newID := lastBill.ID + 1 // Simple increment strategy for now
	return fmt.Sprintf("%s-%s-%05d", prefix, dateStr, newID)
}

type BillItemRequest struct {
	ProductID uint    `json:"product_id" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required"`
	UnitPrice float64 `json:"unit_price" binding:"required"`
	Total     float64 `json:"total" binding:"required"`
}

type CreateBillRequest struct {
	CustomerID     *uint             `json:"customer_id"`
	TotalAmount    float64           `json:"total_amount" binding:"required"`
	DiscountAmount float64           `json:"discount_amount"`
	GSTAmount      float64           `json:"gst_amount"`
	NetPayable     float64           `json:"net_payable" binding:"required"`
	PaymentMode    string            `json:"payment_mode" binding:"required"`
	Items          []BillItemRequest `json:"items" binding:"required"`
}

func (h *BillingHandler) CreateBill(c *gin.Context) {
	var req CreateBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")
	billNo := generateBillNo()

	tx := database.DB.Begin()

	bill := models.Bill{
		BillNo:         billNo,
		BillDate:       time.Now(),
		CustomerID:     req.CustomerID,
		UserID:         userID,
		TotalAmount:    req.TotalAmount,
		DiscountAmount: req.DiscountAmount,
		GSTAmount:      req.GSTAmount,
		NetPayable:     req.NetPayable,
		PaymentMode:    req.PaymentMode,
		Status:         "PAID",
	}

	if err := tx.Create(&bill).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bill record"})
		return
	}

	for _, itemReq := range req.Items {
		// Check Stock
		var product models.Product
		if err := tx.Where("id = ?", itemReq.ProductID).First(&product).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Product ID %d not found", itemReq.ProductID)})
			return
		}

		if product.CurrentStock < itemReq.Quantity {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Insufficient stock for %s", product.Name)})
			return
		}

		// Deduct Stock
		if err := tx.Model(&models.Product{}).Where("id = ?", itemReq.ProductID).Update("current_stock", product.CurrentStock-itemReq.Quantity).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock"})
			return
		}

		// Add Bill Item
		billItem := models.BillItem{
			BillID:    bill.ID,
			ProductID: itemReq.ProductID,
			Quantity:  itemReq.Quantity,
			UnitPrice: itemReq.UnitPrice,
			Total:     itemReq.Total,
		}
		if err := tx.Create(&billItem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add bill item"})
			return
		}
	}

	tx.Commit()
	c.JSON(http.StatusCreated, gin.H{"message": "Bill created successfully", "bill_no": billNo, "bill_id": bill.ID})
}

func (h *BillingHandler) ListBills(c *gin.Context) {
	page := 1
	limit := 10

	if c.Query("page") != "" {
		fmt.Sscanf(c.Query("page"), "%d", &page)
	}
	if c.Query("limit") != "" {
		fmt.Sscanf(c.Query("limit"), "%d", &limit)
	}

	offset := (page - 1) * limit

	var bills []models.Bill
	var total int64

	database.DB.Model(&models.Bill{}).Count(&total)

	if err := database.DB.Preload("Customer").Preload("User").Preload("Items").Preload("Items.Product").Order("bill_date desc").Limit(limit).Offset(offset).Find(&bills).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bills"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  bills,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

type CreateCustomerRequest struct {
	Name            string  `json:"name" binding:"required"`
	Mobile          string  `json:"mobile" binding:"required"`
	Address         string  `json:"address"`
	WhatsappOptIn   bool    `json:"whatsapp_opt_in"`
	DiscountPercent float64 `json:"discount_percent"`
}

func (h *BillingHandler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer := models.Customer{
		Name:            req.Name,
		Mobile:          req.Mobile,
		Address:         req.Address,
		WhatsappOptIn:   req.WhatsappOptIn,
		DiscountPercent: req.DiscountPercent,
	}

	if err := database.DB.Create(&customer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer (Mobile might be duplicate)"})
		return
	}

	c.JSON(http.StatusCreated, customer)
}

func (h *BillingHandler) SearchCustomers(c *gin.Context) {
	query := c.Query("q")
	customers := []models.Customer{} // Initialize as empty slice
	if query == "" {
		database.DB.Limit(20).Find(&customers)
	} else {
		database.DB.Where("name LIKE ? OR mobile LIKE ?", "%"+query+"%", "%"+query+"%").Find(&customers)
	}
	c.JSON(http.StatusOK, customers)
}

func (h *BillingHandler) GetNextBillNo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"next_bill_no": generateBillNo()})
}

func (h *BillingHandler) MyTodaySales(c *gin.Context) {
	userID := c.GetUint("userID")
	// Calculate start of day
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var bills []models.Bill
	if err := database.DB.Where("user_id = ? AND bill_date >= ? AND bill_date < ?", userID, startOfDay, endOfDay).
		Order("bill_date desc").Find(&bills).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sales data"})
		return
	}

	var total float64
	hourlySales := make([]float64, 24)

	for _, bill := range bills {
		total += bill.NetPayable
		hour := bill.BillDate.Hour()
		if hour >= 0 && hour < 24 {
			hourlySales[hour] += bill.NetPayable
		}
	}

	// Recent 5 bills
	var recentBills []models.Bill
	if len(bills) > 5 {
		recentBills = bills[:5]
	} else {
		recentBills = bills
	}

	c.JSON(http.StatusOK, gin.H{
		"sales":        total,
		"hourly_sales": hourlySales,
		"recent_bills": recentBills,
	})
}

func (h *BillingHandler) GetGlobalDiscount(c *gin.Context) {
	var discount models.Discount
	if err := database.DB.Where("is_active = ?", true).Last(&discount).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"percentage": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"percentage": discount.Percentage})
}

func (h *BillingHandler) GetDiscountRules(c *gin.Context) {
	var rules []models.DiscountRule
	if err := database.DB.Where("is_active = ?", true).Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rules"})
		return
	}
	c.JSON(http.StatusOK, rules)
}
