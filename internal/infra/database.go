package infra

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		databaseUrl = "/app/test.db" // Default path for Docker container
	}
	connection, err := gorm.Open(postgres.Open(databaseUrl), &gorm.Config{})
	if err != nil {
		panic("could not connect to the database")
	}
	err = connection.AutoMigrate(&Account{}, &Transaction{})
	if err != nil {
		panic("could not migrate the database")
	}
	DB = connection
}
