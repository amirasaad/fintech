package main

import (
	infra2 "github.com/amirasaad/fintech/infra"
	"log"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/webapi"
)

func main() {
	db, err := infra2.NewDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(webapi.NewApp(func() (repository.UnitOfWork, error) {
		return infra2.NewGormUoW(db)
	}).Listen(":3000"))

}
