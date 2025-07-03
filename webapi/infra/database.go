package infra

import (
	"errors"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDBConnection() (*gorm.DB, error) {
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}
	connection, err := gorm.Open(postgres.Open(databaseUrl), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		return nil, err
	}
	err = connection.AutoMigrate(&Account{}, &Transaction{}, &User{})
	if err != nil {
		return nil, err
	}
	return connection, nil
}
