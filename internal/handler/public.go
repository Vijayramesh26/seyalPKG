package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"billing-app/config"
	"billing-app/internal/models"
	"billing-app/pkg/database"

	"github.com/gin-gonic/gin"
)

type PublicHandler struct{}

func (h *PublicHandler) GetSiteInfo(c *gin.Context) {
	c.JSON(http.StatusOK, config.AppConfig.Site)
}

func (h *PublicHandler) GetPublicConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"company_name":    config.AppConfig.Defaults.CompanyName,
		"company_logo":    config.AppConfig.Defaults.CompanyLogo,
		"company_address": config.AppConfig.Defaults.CompanyAddress,
		"company_phone":   config.AppConfig.Defaults.CompanyPhone,
	})
}

func (h *PublicHandler) ListPublicProducts(c *gin.Context) {
	var products []models.Product
	// Show all active products (including out of stock)
	if err := database.DB.Preload("Brand").Preload("Category").Where("is_active = ?", true).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	c.JSON(http.StatusOK, products)
}

type SubmitOrderRequest struct {
	CustomerMobile string            `json:"customer_mobile" binding:"required"`
	CustomerName   string            `json:"customer_name" binding:"required"`
	Address        string            `json:"address"`
	Items          []BillItemRequest `json:"items" binding:"required"` // Reuse BillItemRequest structure
}

// Generate Order No: ORD-YYYYMMDD-SEQ
func generateOrderNo() string {
	dateStr := time.Now().Format("20060102")
	var lastOrder models.CustomerOrder
	database.DB.Order("id desc").First(&lastOrder)
	newID := lastOrder.ID + 1
	return fmt.Sprintf("ORD-%s-%04d", dateStr, newID)
}

func (h *PublicHandler) SubmitOrder(c *gin.Context) {
	var req SubmitOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find or Create Customer
	var customer models.Customer
	if err := database.DB.Where("mobile = ?", req.CustomerMobile).First(&customer).Error; err != nil {
		// New Customer
		customer = models.Customer{
			Name:    req.CustomerName,
			Mobile:  req.CustomerMobile,
			Address: req.Address,
		}
		if err := database.DB.Create(&customer).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process customer info"})
			return
		}
	} else {
		// Existing Customer - Update details if changed
		if customer.Name != req.CustomerName || customer.Address != req.Address {
			customer.Name = req.CustomerName
			customer.Address = req.Address
			database.DB.Save(&customer)
		}
	}

	tx := database.DB.Begin()

	order := models.CustomerOrder{
		OrderNo:    generateOrderNo(),
		CustomerID: customer.ID,
		Status:     "PENDING",
		OrderDate:  time.Now(),
	}

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Build detailed message
	var msgBuilder strings.Builder
	// Initial message header
	msgBuilder.WriteString(fmt.Sprintf("Hello *%s*, your order *%s* is placed successfully! 🛒\n\n*Items Ordered:*\n", customer.Name, order.OrderNo))

	var totalEstimated float64
	for _, itemReq := range req.Items {
		// Verify Product Price/Existence
		var product models.Product
		if err := database.DB.First(&product, itemReq.ProductID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product"})
			return
		}

		itemTotal := product.UnitPrice * float64(itemReq.Quantity)
		totalEstimated += itemTotal

		// Add item to message
		msgBuilder.WriteString(fmt.Sprintf("• %s x %d - ₹%.2f\n", product.Name, itemReq.Quantity, itemTotal))

		orderItem := models.OrderItem{
			OrderID:   order.ID,
			ProductID: itemReq.ProductID,
			Quantity:  itemReq.Quantity,
		}
		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add order item"})
			return
		}
	}

	// Update total
	tx.Model(&order).Update("total_estimated", totalEstimated)
	tx.Commit()

	// Finalize WhatsApp Message
	msgBuilder.WriteString(fmt.Sprintf("\n*Total Amount:* ₹%.2f\n", totalEstimated))
	if customer.Address != "" {
		msgBuilder.WriteString(fmt.Sprintf("*Delivery Address:* %s\n", customer.Address))
	}
	msgBuilder.WriteString("\nThank you for shopping with us! 🙏")

	// URL Encode the message
	encodedMsg := url.QueryEscape(msgBuilder.String())

	// Format mobile number (ensure 91 prefix for India if 10 digits)
	targetMobile := customer.Mobile
	if len(targetMobile) == 10 {
		targetMobile = "91" + targetMobile
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Order placed successfully",
		"order_no":     order.OrderNo,
		"whatsapp_url": fmt.Sprintf("https://wa.me/%s?text=%s", targetMobile, encodedMsg),
	})
}
