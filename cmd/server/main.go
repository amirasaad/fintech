package main

import (
	"log"

	"github.com/amirasaad/fintech/infra"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/amirasaad/fintech/webapi"
)

func main() {
	db, err := infra.NewDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(webapi.NewApp(func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW(db)
	}, service.NewJWTAuthStrategy(func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW(db)
	})).Listen(":3000"))

}
