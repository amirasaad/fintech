package infra

import (
	"errors"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDBConnection() (*gorm.DB, error) {
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	var logMode logger.LogLevel
	if os.Getenv("APP_ENV") == "development" {
		logMode = logger.Info
	} else {
		logMode = logger.Silent
	}

	connection, err := gorm.Open(postgres.Open(databaseUrl), &gorm.Config{
		Logger:                 logger.Default.LogMode(logMode),
		SkipDefaultTransaction: true})
	if err != nil {
		return nil, err
	}
	err = connection.AutoMigrate(&Account{}, &Transaction{}, &User{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := connection.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	return connection, nil
}
