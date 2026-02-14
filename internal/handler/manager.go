package handler

import (
	"net/http"
	"time"

	"billing-app/internal/models"
	"billing-app/pkg/database"

	"github.com/gin-gonic/gin"
)

type ManagerHandler struct{}

func (h *ManagerHandler) GetSalesReport(c *gin.Context) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var bills []models.Bill
	query := database.DB.Preload("Items").Preload("User")

	if startDateStr != "" && endDateStr != "" {
		// Parse dates assuming YYYY-MM-DD
		startDate, _ := time.Parse("2006-01-02", startDateStr)
		endDate, _ := time.Parse("2006-01-02", endDateStr)
		// Set end date to end of day
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		query = query.Where("bill_date BETWEEN ? AND ?", startDate, endDate)
	}

	if err := query.Find(&bills).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sales report"})
		return
	}

	// Calculate Summary
	var totalRevenue float64
	var totalTransactions int
	var productsSold int

	for _, bill := range bills {
		totalRevenue += bill.NetPayable
		totalTransactions++
		for _, item := range bill.Items {
			productsSold += item.Quantity
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"total_revenue":      totalRevenue,
			"total_transactions": totalTransactions,
			"products_sold":      productsSold,
		},
		"transactions": bills,
	})
}

func (h *ManagerHandler) ListCustomerOrders(c *gin.Context) {
	status := c.Query("status")
	var orders []models.CustomerOrder

	query := database.DB.Preload("Customer").Preload("Items.Product").Preload("Items.Product.Brand").Order("order_date desc")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Explicitly preload Product then Brand to ensure deep nesting works
	if err := query.Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *ManagerHandler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"` // PENDING, COMPLETED, CANCELLED
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&models.CustomerOrder{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order status updated"})
}

func (h *ManagerHandler) SetGlobalDiscount(c *gin.Context) {
	var req struct {
		Percentage float64 `json:"percentage" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Disable previous active discounts
	database.DB.Model(&models.Discount{}).Where("is_active = ?", true).Update("is_active", false)

	// Add new discount
	discount := models.Discount{
		Name:       "Standard Manager Discount",
		Percentage: req.Percentage,
		IsActive:   true,
	}

	if err := database.DB.Create(&discount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set discount"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Global discount updated"})
}

func (h *ManagerHandler) GetGlobalDiscount(c *gin.Context) {
	var discount models.Discount
	if err := database.DB.Where("is_active = ?", true).Last(&discount).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"percentage": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"percentage": discount.Percentage})
}

func (h *ManagerHandler) UpdateCustomerDiscount(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		DiscountPercent float64 `json:"discount_percent" binding:"gte=0,lte=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discount percentage"})
		return
	}

	if err := database.DB.Model(&models.Customer{}).Where("id = ?", id).Update("discount_percent", req.DiscountPercent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer discount"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer discount updated"})
}

func (h *ManagerHandler) GetCustomers(c *gin.Context) {
	type CustomerWithStats struct {
		models.Customer
		TotalSpend float64 `json:"total_spend"`
		Rank       int     `json:"rank"`
	}

	var customers []models.Customer
	if err := database.DB.Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customers"})
		return
	}

	// Calculate Total Spend for each customer
	var customerStats []CustomerWithStats
	for _, cust := range customers {
		var total float64
		// Assumes bills table has customer_id and net_payable
		// database.DB.Model(&models.Bill{}).Where("customer_id = ?", cust.ID).Select("sum(net_payable)").Scan(&total)
		// Scan into *float64 to handle NULLs if no bills exist (Scan treats NULL as 0 for float64 destination usually, but let's be safe with a check or just simple scan)
		database.DB.Table("bills").Where("customer_id = ?", cust.ID).Select("COALESCE(SUM(net_payable), 0)").Scan(&total)

		customerStats = append(customerStats, CustomerWithStats{
			Customer:   cust,
			TotalSpend: total,
		})
	}

	// Sort by Total Spend Descending
	for i := range customerStats {
		for j := i + 1; j < len(customerStats); j++ {
			if customerStats[i].TotalSpend < customerStats[j].TotalSpend {
				customerStats[i], customerStats[j] = customerStats[j], customerStats[i]
			}
		}
	}

	// Assign Ranks
	for i := range customerStats {
		customerStats[i].Rank = i + 1
	}

	c.JSON(http.StatusOK, customerStats)
}

func (h *ManagerHandler) GetDashboardStats(c *gin.Context) {
	// 1. Metrics
	var todayRevenue float64
	var inventoryValue float64
	var totalSales float64 // Current month or total
	var totalOrders int64
	var lowStockCount int64
	var newCustomers int64

	// Today's Revenue (from Bills, as bills are final revenue)
	today := time.Now().Format("2006-01-02")
	database.DB.Model(&models.Bill{}).Where("DATE(bill_date) = ?", today).Select("COALESCE(SUM(net_payable), 0)").Scan(&todayRevenue)

	// Inventory Value
	// database.DB.Model(&models.Product{}).Select("COALESCE(SUM(unit_price * current_stock), 0)").Scan(&inventoryValue)
	// SQLite/MySQL compatible query
	var products []models.Product
	database.DB.Find(&products)
	for _, p := range products {
		inventoryValue += p.UnitPrice * float64(p.CurrentStock)
	}

	// Total Sales (All time for now, or this month)
	database.DB.Model(&models.Bill{}).Select("COALESCE(SUM(net_payable), 0)").Scan(&totalSales)

	// Total Orders (Count of bills)
	database.DB.Model(&models.Bill{}).Count(&totalOrders)

	// Low Stock
	database.DB.Model(&models.Product{}).Where("current_stock < 20").Count(&lowStockCount) // Threshold 20 mostly used

	// New Customers (Today)
	database.DB.Model(&models.Customer{}).Where("DATE(created_at) = ?", today).Count(&newCustomers)

	// 2. Charts Data

	// Monthly Chart (Last 4 weeks/months) - Simplified to last 4 days for demo or weeks if data allows
	// Let's do last 7 days sales
	type ChartData struct {
		Labels []string  `json:"labels"`
		Data   []float64 `json:"data"`
	}
	monthlyChart := ChartData{Labels: []string{}, Data: []float64{}}
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		label := date.Format("Jan 02")
		var dailySum float64
		database.DB.Model(&models.Bill{}).Where("DATE(bill_date) = ?", dateStr).Select("COALESCE(SUM(net_payable), 0)").Scan(&dailySum)
		monthlyChart.Labels = append(monthlyChart.Labels, label)
		monthlyChart.Data = append(monthlyChart.Data, dailySum)
	}

	// Category Distribution
	// Group by Brand for now as Category isn't strictly defined in Products, or simulate mock if needed.
	// We can link Product -> Brand.
	type PieData struct {
		Labels []string  `json:"labels"`
		Data   []float64 `json:"data"`
	}
	categoryChart := PieData{Labels: []string{}, Data: []float64{}}
	rows, _ := database.DB.Table("order_items").
		Joins("JOIN products ON products.id = order_items.product_id").
		Joins("JOIN brands ON brands.id = products.brand_id").
		Select("brands.name, count(order_items.id)").
		Group("brands.name").
		Rows()
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var brandName string
			var count float64
			rows.Scan(&brandName, &count)
			categoryChart.Labels = append(categoryChart.Labels, brandName)
			categoryChart.Data = append(categoryChart.Data, count)
		}
	}

	// Biller-wise Sales Chart
	type BarData struct {
		Labels []string  `json:"labels"`
		Data   []float64 `json:"data"`
	}
	billerChart := BarData{Labels: []string{}, Data: []float64{}}

	// Query to sum net_payable by user_id and join with users table to get username
	billerRows, _ := database.DB.Table("bills").
		Joins("JOIN users ON users.id = bills.user_id").
		Select("users.username, COALESCE(SUM(bills.net_payable), 0)").
		Group("users.username").
		Rows()

	if billerRows != nil {
		defer billerRows.Close()
		for billerRows.Next() {
			var username string
			var totalSales float64
			billerRows.Scan(&username, &totalSales)
			billerChart.Labels = append(billerChart.Labels, username)
			billerChart.Data = append(billerChart.Data, totalSales)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": gin.H{
			"todayRevenue":   todayRevenue,
			"inventoryValue": inventoryValue,
			"totalSales":     totalSales,
			"totalOrders":    totalOrders,
			"lowStock":       lowStockCount,
			"newCustomers":   newCustomers,
		},
		"charts": gin.H{
			"monthly":     monthlyChart,
			"category":    categoryChart,
			"billerSales": billerChart, // Added billerChart to response
			// Traffic and Weekly can be mocked or calculated similarly if needed
			"traffic": []int{10, 20, 15, 30, 25, 40, 35}, // Mock for now
			"weekly":  []int{65, 59, 80, 81, 56, 55, 40}, // Mock for now
		},
	})
}
