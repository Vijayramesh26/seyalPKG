package database

import (
	"log"

	"billing-app/config"
	"billing-app/internal/models"
	"billing-app/internal/utils"

	"gorm.io/gorm"
)

func SeedRolesAndAdmin() {
	// Seed Roles
	roles := []string{"admin", "manager", "inventory", "biller"}
	for _, r := range roles {
		var role models.Role
		if err := DB.FirstOrCreate(&role, models.Role{Name: r}).Error; err != nil {
			log.Printf("Failed to seed role %s: %v", r, err)
		}
	}

	// Seed Admin User
	var adminRole models.Role
	DB.Where("name = ?", "admin").First(&adminRole)

	var adminUser models.User
	if err := DB.Where("employee_id = ?", config.AppConfig.Defaults.AdminEmployeeID).First(&adminUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			hashedPassword, _ := utils.HashPassword(config.AppConfig.Defaults.AdminPassword)
			admin := models.User{
				EmployeeID:   config.AppConfig.Defaults.AdminEmployeeID,
				Username:     "Attributes Admin",
				PasswordHash: hashedPassword,
				RoleID:       adminRole.ID,
				IsActive:     true,
			}
			if err := DB.Create(&admin).Error; err != nil {
				log.Printf("Failed to seed admin user: %v", err)
			} else {
				log.Println("Admin user seeded successfully.")
			}
		}
	}
}
