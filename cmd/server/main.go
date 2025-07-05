package main

import (
	"log"

	"github.com/amirasaad/fintech/infra"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/amirasaad/fintech/webapi"
)

// @title Fintech API
// @version 1.0.0
// @description Fintech API documentation
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email fiber@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/MIT
// @host localhost:3000
// @BasePath /
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
