package handler

import (
	"net/http"

	"billing-app/internal/models"
	"billing-app/internal/utils"
	"billing-app/pkg/database"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	EmployeeID string `json:"employee_id" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type AuthHandler struct{}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Preload("Role").Where("employee_id = ?", req.EmployeeID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is inactive"})
		return
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Role.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Update Login History (Async in production, but sync here for simplicity)
	go func(uid uint, ip string) {
		database.DB.Create(&models.LoginHistory{
			UserID:    uid,
			IPAddress: ip,
		})
	}(user.ID, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role.Name,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
