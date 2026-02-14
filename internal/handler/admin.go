package handler

import (
	"fmt"
	"net/http"

	"billing-app/config"
	"billing-app/internal/models"
	"billing-app/internal/utils"
	"billing-app/pkg/database"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct{}

type CreateEmployeeRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	RoleID   uint   `json:"role_id" binding:"required"`
	Mobile   string `json:"mobile"`
}

func (h *AdminHandler) CreateEmployee(c *gin.Context) {
	var req CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Generate Employee ID
	empID := generateEmployeeID(req.RoleID)

	user := models.User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		RoleID:       req.RoleID,
		EmployeeID:   empID,
		Mobile:       req.Mobile,
		IsActive:     true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "user_id": user.ID})
}

func (h *AdminHandler) ListEmployees(c *gin.Context) {
	var users []models.User
	if err := database.DB.Preload("Role").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *AdminHandler) UpdateEmployeeRole(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		RoleID uint `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", id).Update("role_id", req.RoleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
}

func (h *AdminHandler) UpdateEmployeeStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		IsActive       bool   `json:"is_active"`
		InactiveReason string `json:"inactive_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", id).Updates(models.User{IsActive: req.IsActive, InactiveReason: req.InactiveReason}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

func (h *AdminHandler) UpdateEmployee(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Username string `json:"username" binding:"required"`
		Mobile   string `json:"mobile"`
		RoleID   uint   `json:"role_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"username": req.Username,
		"mobile":   req.Mobile,
		"role_id":  req.RoleID,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Employee updated successfully"})
}

func (h *AdminHandler) ResetEmployeePassword(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, _ := utils.HashPassword(req.Password)
	if err := database.DB.Model(&models.User{}).Where("id = ?", id).Update("password_hash", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

func (h *AdminHandler) GetLoginHistory(c *gin.Context) {
	var history []models.LoginHistory
	if err := database.DB.Preload("User").Preload("User.Role").Order("login_time desc").Limit(100).Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch login history"})
		return
	}
	c.JSON(http.StatusOK, history)
}

func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	var totalEmployees int64
	var activeUsers int64

	database.DB.Model(&models.User{}).Count(&totalEmployees)
	database.DB.Model(&models.User{}).Where("is_active = ?", true).Count(&activeUsers)

	// Role Distribution
	type RoleCount struct {
		Name  string
		Count int
	}
	var roleCounts []RoleCount
	database.DB.Model(&models.User{}).Select("roles.name, count(users.id) as count").Joins("left join roles on roles.id = users.role_id").Group("roles.name").Scan(&roleCounts)

	c.JSON(http.StatusOK, gin.H{
		"total_employees":   totalEmployees,
		"active_users":      activeUsers,
		"role_distribution": roleCounts,
	})
}

func generateEmployeeID(roleID uint) string {
	var prefix string
	switch roleID {
	case 1:
		prefix = config.AppConfig.Defaults.AdminEmployeeID // Fallback/Config logic needs review if this is a full ID
		if len(prefix) > 3 {
			prefix = prefix[:3]
		} else {
			prefix = "ADM"
		}
	case 2:
		prefix = config.AppConfig.Defaults.ManagerPrefix
	case 3:
		prefix = config.AppConfig.Defaults.InventoryPrefix
	case 4:
		prefix = config.AppConfig.Defaults.BillerPrefix
	default:
		prefix = "EMP"
	}

	var lastUser models.User
	// Find last user with this prefix to increment.
	// Note: This is a simple implementation. In high concurrency, use a sequence or localized lock.
	if err := database.DB.Where("employee_id LIKE ?", prefix+"%").Order("id desc").First(&lastUser).Error; err != nil {
		return fmt.Sprintf("%s001", prefix)
	}

	// Extract number
	var lastID int
	fmt.Sscanf(lastUser.EmployeeID, prefix+"%d", &lastID)
	return fmt.Sprintf("%s%03d", prefix, lastID+1)
}
