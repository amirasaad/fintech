package infra

import (
	"errors"
	"time"

	"github.com/amirasaad/fintech/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Add appEnv as a parameter for dependency-injected environment
func NewDBConnection(
	cnf config.DBConfig,
	appEnv string,
) (*gorm.DB, error) {
	databaseUrl := cnf.Url
	if databaseUrl == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	var logMode logger.LogLevel
	if appEnv == "development" {
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

	sqlDB, err := connection.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	return connection, nil
}
