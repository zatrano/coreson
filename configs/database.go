package configs

import (
	"os"
	"strconv"
	"time"

	"zatrano/pkg/env"
	"zatrano/pkg/logs"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

func InitDB() {
	err := godotenv.Load()
	if err != nil {
		logs.SLog.Warnw(".env dosyası yüklenemedi, sistem ortam değişkenleri kullanılacak (eğer varsa)", "error", err)
	} else {
		logs.SLog.Info(".env dosyası başarıyla yüklendi")
	}

	portStr := env.GetEnvWithDefault("DB_PORT", "5432")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		logs.SLog.Fatalw("Invalid DB_PORT environment variable",
			"value", portStr,
			"error", err,
		)
	}

	dbConfig := DatabaseConfig{
		Host:     env.GetEnvWithDefault("DB_HOST", "localhost"),
		Port:     port,
		User:     env.GetEnvWithDefault("DB_USERNAME", "postgres"),
		Password: env.GetEnvWithDefault("DB_PASSWORD", ""),
		Name:     env.GetEnvWithDefault("DB_DATABASE", "myapp"),
		SSLMode:  env.GetEnvWithDefault("DB_SSL_MODE", "disable"),
		TimeZone: env.GetEnvWithDefault("DB_TIMEZONE", "UTC"),
	}

	logs.Log.Info("Database configuration loaded",
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port),
		zap.String("user", dbConfig.User),
		zap.String("database", dbConfig.Name),
		zap.String("sslmode", dbConfig.SSLMode),
		zap.String("timezone", dbConfig.TimeZone),
	)

	dsn := "host=" + dbConfig.Host +
		" user=" + dbConfig.User +
		" password=" + dbConfig.Password +
		" dbname=" + dbConfig.Name +
		" port=" + strconv.Itoa(dbConfig.Port) +
		" sslmode=" + dbConfig.SSLMode +
		" TimeZone=" + dbConfig.TimeZone

	var gormerr error
	DB, gormerr = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(getGormLogLevel()),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if gormerr != nil {
		logs.Log.Fatal("Failed to connect to database",
			zap.String("host", dbConfig.Host),
			zap.Int("port", dbConfig.Port),
			zap.String("user", dbConfig.User),
			zap.String("database", dbConfig.Name),
			zap.Error(gormerr),
		)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		logs.Log.Fatal("Failed to get underlying sql.DB instance", zap.Error(err))
	}

	maxIdleConns := env.GetEnvAsInt("DB_MAX_IDLE_CONNS", 10)
	maxOpenConns := env.GetEnvAsInt("DB_MAX_OPEN_CONNS", 100)
	connMaxLifetimeMinutes := env.GetEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 60)

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetimeMinutes) * time.Minute)

	logs.Log.Info("Database connection established successfully",
		zap.Int("max_idle_conns", maxIdleConns),
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("conn_max_lifetime_minutes", connMaxLifetimeMinutes),
	)
}

func getGormLogLevel() logger.LogLevel {
	switch os.Getenv("DB_LOG_LEVEL") {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	default:
		return logger.Info
	}
}

func GetDB() *gorm.DB {
	if DB == nil {
		logs.Log.Fatal("Database connection not initialized. Call InitDB() first.")
	}
	return DB
}

func CloseDB() error {
	if DB == nil {
		logs.SLog.Info("Database connection already closed or not initialized.")
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		logs.Log.Error("Failed to get database instance for closing", zap.Error(err))
		return err
	}

	err = sqlDB.Close()
	if err != nil {
		logs.Log.Error("Error closing database connection", zap.Error(err))
		return err
	}

	logs.SLog.Info("Database connection closed successfully.")
	DB = nil
	return nil
}
