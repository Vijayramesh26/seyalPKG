package config

import (
	"billing-app/internal/models"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Defaults DefaultsConfig
	Site     models.SiteInfo
}

type ServerConfig struct {
	Port               string
	Env                string
	JWTSecret          string `mapstructure:"jwt_secret"`
	JWTExpirationHours int    `mapstructure:"jwt_expiration_hours"`
}

type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	URL      string
}

type DefaultsConfig struct {
	AdminPassword   string `mapstructure:"admin_password"`
	AdminEmployeeID string `mapstructure:"admin_employee_id"`
	BillerPrefix    string `mapstructure:"biller_prefix"`
	InventoryPrefix string `mapstructure:"inventory_prefix"`
	ManagerPrefix   string `mapstructure:"manager_prefix"`
	CompanyName     string `mapstructure:"company_name"`
	CompanyLogo     string `mapstructure:"company_logo"`
	CompanyAddress  string `mapstructure:"company_address"`
	CompanyPhone    string `mapstructure:"company_phone"`
}

var AppConfig *Config

func LoadConfig() {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// Read .env file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: .env file not found, checking environment variables: %v", err)
	}

	// Enable reading from OS environment variables as fallback/override
	viper.AutomaticEnv()

	// Explicitly bind environment variables for robustness
	viper.BindEnv("SERVER_PORT", "PORT") // Fallback to PORT if SERVER_PORT is missing
	viper.BindEnv("DATABASE_URL")

	// Manually map configuration to struct
	AppConfig = &Config{
		Server: ServerConfig{
			Port:               viper.GetString("SERVER_PORT"),
			Env:                viper.GetString("SERVER_ENV"),
			JWTSecret:          viper.GetString("JWT_SECRET"),
			JWTExpirationHours: viper.GetInt("JWT_EXPIRATION_HOURS"),
		},
		Database: DatabaseConfig{
			Driver:   viper.GetString("DB_DRIVER"),
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
			URL:      viper.GetString("DATABASE_URL"),
		},
		Defaults: DefaultsConfig{
			AdminPassword:   viper.GetString("ADMIN_PASSWORD"),
			AdminEmployeeID: viper.GetString("ADMIN_EMPLOYEE_ID"),
			BillerPrefix:    viper.GetString("BILLER_PREFIX"),
			InventoryPrefix: viper.GetString("INVENTORY_PREFIX"),
			ManagerPrefix:   viper.GetString("MANAGER_PREFIX"),
			CompanyName:     viper.GetString("COMPANY_NAME"),
			CompanyLogo:     viper.GetString("COMPANY_LOGO"),
			CompanyAddress:  viper.GetString("COMPANY_ADDRESS"),
			CompanyPhone:    viper.GetString("COMPANY_PHONE"),
		},
	}

	// Load TOML Config for Site Info
	siteViper := viper.New()
	siteViper.SetConfigFile("config/config.toml")
	siteViper.SetConfigType("toml")
	if err := siteViper.ReadInConfig(); err != nil {
		log.Printf("Warning: config/config.toml not found, using empty site info: %v", err)
	} else {
		if err := siteViper.UnmarshalKey("site", &AppConfig.Site); err != nil {
			log.Printf("Error: Failed to unmarshal site info from TOML: %v", err)
		}
	}

	log.Printf("Configuration loaded successfully:")
	log.Printf("- Server Port: %s", AppConfig.Server.Port)
	log.Printf("- Server Env: %s", AppConfig.Server.Env)
	log.Printf("- JWT Secret Path: %s", func() string {
		if AppConfig.Server.JWTSecret != "" {
			return "SET"
		}
		return "NOT SET"
	}())
	log.Printf("- Database Driver: %s", AppConfig.Database.Driver)
	log.Printf("- Database Host: %s", AppConfig.Database.Host)
	log.Printf("- Database Port: %s", AppConfig.Database.Port)
	log.Printf("- Database Name: %s", AppConfig.Database.Name)
	log.Printf("- Database URL: %s", func() string {
		if AppConfig.Database.URL != "" {
			return "SET"
		}
		return "NOT SET"
	}())
	log.Printf("- Company Name: %s", AppConfig.Defaults.CompanyName)
}
