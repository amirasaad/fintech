package main

import (
	"log"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/webapi"
	"github.com/amirasaad/fintech/webapi/infra"
)

func main() {
	db, err := infra.NewDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(webapi.NewApp(func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW(db)
	}).Listen(":3000"))

}
