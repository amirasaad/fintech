package main

import (
	"fmt"
	"github.com/amirasaad/fintech/infra"
	"os"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	argsLen := len(os.Args)
	if argsLen < 2 {
		println("cmd is required")
		return
	}
	cmd := os.Args[1]
	println(fmt.Sprintf("argsLen: %d", argsLen))
	println(fmt.Sprintf("cmd: %s", cmd))
	dialector := sqlite.Open("fixtures.db")

	db, err := gorm.Open(dialector, &gorm.Config{})
	db.AutoMigrate(&domain.Account{}, &domain.Transaction{})
	scv := service.NewAccountService(func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW(db)
	})
	var account *domain.Account
	switch cmd {
	case "create":
		account, err = scv.CreateAccount(uuid.New())
	}
	if err != nil {
		println("An error occurred", err)
		return
	}
	println(account)
}
