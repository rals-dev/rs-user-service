package config

import (
	"fmt"
	"net/url"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDatabase() (*gorm.DB, error) {
	cfg := Config
	encodedPassword := url.QueryEscape(cfg.Database.Password)
	uri := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.Username,
		encodedPassword,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	// Only log full SQL statements (which may include sensitive data such as
	// password hashes) outside of production; production gets warnings/errors only.
	logLevel := logger.Warn
	if cfg.AppEnv != ProductionEnv {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(uri), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConnection)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConnection)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.MaxLifetimeConnection) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.Database.MaxIdleTime) * time.Second)
	return db, nil
}
