package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"billing-app/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	var dsn string

	// Prioritize DATABASE_URL if provided (common on Render/SkySQL)
	if config.AppConfig.Database.URL != "" {
		log.Println("Using DATABASE_URL for connection")
		dsn = config.AppConfig.Database.URL

		// Convert mysql:// or mariadb:// URL to DSN if needed
		if strings.HasPrefix(dsn, "mysql://") || strings.HasPrefix(dsn, "mariadb://") {
			log.Println("Converting URL to DSN format")
			// Standard URL: mysql://user:pass@host:port/dbname
			// DSN: user:pass@tcp(host:port)/dbname?params

			rawDSN := dsn
			if strings.HasPrefix(dsn, "mysql://") {
				rawDSN = strings.TrimPrefix(dsn, "mysql://")
			} else {
				rawDSN = strings.TrimPrefix(dsn, "mariadb://")
			}

			// Split at @ to get credentials and host/db
			parts := strings.SplitN(rawDSN, "@", 2)
			if len(parts) == 2 {
				creds := parts[0]
				rest := parts[1]

				// Split rest at / to get host:port and dbname
				hostParts := strings.SplitN(rest, "/", 2)
				if len(hostParts) == 2 {
					hostPort := hostParts[0]
					dbName := hostParts[1]

					// Handle query params if any
					params := ""
					if strings.Contains(dbName, "?") {
						dbParts := strings.SplitN(dbName, "?", 2)
						dbName = dbParts[0]
						params = "?" + dbParts[1]
					} else {
						params = "?charset=utf8mb4&parseTime=True&loc=Local"
					}

					dsn = fmt.Sprintf("%s@tcp(%s)/%s%s", creds, hostPort, dbName, params)
				}
			}
		}
	} else {
		log.Println("Constructing DSN from individual components")
		// Use a more robust way to construct DSN to handle special characters
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.AppConfig.Database.User,
			config.AppConfig.Database.Password,
			config.AppConfig.Database.Host,
			config.AppConfig.Database.Port,
			config.AppConfig.Database.Name,
		)
	}

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		// Log the error but be careful not to reveal sensitive info if possible
		// However, for debugging connection issues, the full error is needed
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	// Connection pooling configuration
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connection established successfully")
}
