package main

import (
	"log"
	"time"

	"billing-app/config"
	"billing-app/internal/handler"
	"billing-app/internal/middleware"
	"billing-app/internal/models"
	"billing-app/pkg/database"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load Configuration
	config.LoadConfig()

	// 2. Connect to Database
	database.Connect()

	// 3. Auto-Migrate Models
	log.Println("Running migrations...")

	err := database.DB.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.LoginHistory{},
		&models.Brand{},
		&models.Category{}, // Added
		&models.Product{},
		&models.StockEntry{},
		&models.Customer{},
		&models.CustomerOrder{},
		&models.OrderItem{},
		&models.Discount{},
		&models.DiscountRule{}, // Added
		&models.Bill{},
		&models.BillItem{},
	)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migrations completed successfully.")

	// 3a. Seed Data
	database.SeedRolesAndAdmin()

	// 4. Initialize Router
	r := gin.Default()

	// CORS Configuration
	// CORS Configuration
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 5. Setup Routes
	authHandler := &handler.AuthHandler{}
	authRoutes := r.Group("/api/v1/auth")
	{
		authRoutes.POST("/login", authHandler.Login)
	}

	userRoutes := r.Group("/api/v1/user")
	userRoutes.Use(middleware.AuthMiddleware())
	{
		userRoutes.PUT("/password", authHandler.ChangePassword)
	}

	adminHandler := &handler.AdminHandler{}
	adminRoutes := r.Group("/api/v1/admin")
	adminRoutes.Use(middleware.AuthMiddleware("admin"))
	{
		adminRoutes.POST("/employees", adminHandler.CreateEmployee)
		adminRoutes.GET("/employees", adminHandler.ListEmployees)
		adminRoutes.PUT("/employees/:id", adminHandler.UpdateEmployee)
		adminRoutes.PUT("/employees/:id/role", adminHandler.UpdateEmployeeRole)
		adminRoutes.PUT("/employees/:id/status", adminHandler.UpdateEmployeeStatus)
		adminRoutes.PUT("/employees/:id/password", adminHandler.ResetEmployeePassword)
		adminRoutes.GET("/login-history", adminHandler.GetLoginHistory)
		adminRoutes.GET("/dashboard", adminHandler.GetDashboardStats)
	}

	inventoryHandler := &handler.InventoryHandler{}

	// Public Read (Authenticated)
	r.GET("/api/v1/inventory/products", middleware.AuthMiddleware(), inventoryHandler.ListProducts)
	r.GET("/api/v1/inventory/brands", middleware.AuthMiddleware(), inventoryHandler.ListBrands)
	r.GET("/api/v1/inventory/categories", middleware.AuthMiddleware(), inventoryHandler.ListCategories) // Added

	// Protected Inventory Ops
	invRoutes := r.Group("/api/v1/inventory")
	invRoutes.Use(middleware.AuthMiddleware("admin", "manager", "inventory"))
	{
		invRoutes.POST("/products", inventoryHandler.CreateProduct)
		invRoutes.POST("/stock", inventoryHandler.AddStock)
		invRoutes.GET("/alerts", inventoryHandler.GetLowStockAlerts)
		invRoutes.POST("/categories", inventoryHandler.CreateCategory) // Added
	}

	managerHandler := &handler.ManagerHandler{}

	billingHandler := &handler.BillingHandler{}
	billingRoutes := r.Group("/api/v1/billing")
	billingRoutes.Use(middleware.AuthMiddleware("biller", "manager", "admin"))
	{
		billingRoutes.POST("/bills", billingHandler.CreateBill)
		billingRoutes.GET("/bills", billingHandler.ListBills)
		billingRoutes.GET("/next-bill-no", billingHandler.GetNextBillNo)
		billingRoutes.POST("/customers", billingHandler.CreateCustomer)
		billingRoutes.GET("/customers", billingHandler.SearchCustomers)

		billingRoutes.GET("/my-sales", billingHandler.MyTodaySales)
		billingRoutes.GET("/discount", billingHandler.GetGlobalDiscount)
		billingRoutes.GET("/discount-rules", billingHandler.GetDiscountRules)

		// Shared Order Management for Billers
		billingRoutes.GET("/orders", managerHandler.ListCustomerOrders)
		billingRoutes.PUT("/orders/:id/status", managerHandler.UpdateOrderStatus)
	}

	managerRoutes := r.Group("/api/v1/manager")
	managerRoutes.Use(middleware.AuthMiddleware("manager", "admin"))
	{
		managerRoutes.GET("/reports/sales", managerHandler.GetSalesReport)
		managerRoutes.GET("/orders", managerHandler.ListCustomerOrders)
		managerRoutes.PUT("/orders/:id/status", managerHandler.UpdateOrderStatus)
		managerRoutes.POST("/settings/discount", managerHandler.SetGlobalDiscount)
		managerRoutes.GET("/settings/discount", managerHandler.GetGlobalDiscount)
		managerRoutes.PUT("/customers/:id/discount", managerHandler.UpdateCustomerDiscount)
		managerRoutes.GET("/customers", managerHandler.GetCustomers)
		managerRoutes.GET("/dashboard", managerHandler.GetDashboardStats) // Added
	}

	publicHandler := &handler.PublicHandler{}
	publicRoutes := r.Group("/api/v1/public")
	{
		publicRoutes.GET("/config", publicHandler.GetPublicConfig)
		publicRoutes.GET("/products", publicHandler.ListPublicProducts)
		publicRoutes.POST("/orders", publicHandler.SubmitOrder)
		publicRoutes.GET("/site-info", publicHandler.GetSiteInfo)
	}

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// 6. Start Server
	port := config.AppConfig.Server.Port
	log.Printf("Server starting on port %s (UPDATED_VERSION_CHECK)", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
