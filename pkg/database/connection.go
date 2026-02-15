package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"billing-app/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	var dialector gorm.Dialector
	dbConf := config.AppConfig.Database

	if dbConf.URL != "" {
		log.Println("Using DATABASE_URL for connection")
		if dbConf.Type == "postgres" || strings.HasPrefix(dbConf.URL, "postgres") {
			dialector = postgres.Open(dbConf.URL)
		} else {
			dsn := dbConf.URL
			// Convert mysql:// or mariadb:// URL to DSN if needed
			if strings.HasPrefix(dsn, "mysql://") || strings.HasPrefix(dsn, "mariadb://") {
				log.Println("Converting URL to DSN format")
				rawDSN := dsn
				if strings.HasPrefix(dsn, "mysql://") {
					rawDSN = strings.TrimPrefix(dsn, "mysql://")
				} else {
					rawDSN = strings.TrimPrefix(dsn, "mariadb://")
				}

				parts := strings.SplitN(rawDSN, "@", 2)
				if len(parts) == 2 {
					creds := parts[0]
					rest := parts[1]
					hostParts := strings.SplitN(rest, "/", 2)
					if len(hostParts) == 2 {
						hostPort := hostParts[0]
						dbName := hostParts[1]
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
			dialector = mysql.Open(dsn)
		}
	} else {
		log.Printf("Connecting to DB: Type=%s, Host=%s, User=%s, Name=%s, Port=%s", dbConf.Type, dbConf.Host, dbConf.User, dbConf.Name, dbConf.Port)

		if dbConf.Type == "postgres" {
			sslMode := "disable"
			if dbConf.SSL {
				sslMode = "require"
			}
			dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
				dbConf.Host, dbConf.User, dbConf.Password, dbConf.Name, dbConf.Port, sslMode)
			dialector = postgres.Open(dsn)
		} else {
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port, dbConf.Name)
			if dbConf.SSL {
				dsn += "&tls=true"
			}
			dialector = mysql.Open(dsn)
		}
	}

	var err error
	DB, err = gorm.Open(dialector, &gorm.Config{
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
