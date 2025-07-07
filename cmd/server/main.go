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
// @host fintech-beryl-beta.vercel.app
// @BasePath /
func main() {
	db, err := infra.NewDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	// Create UOW factory
	uowFactory := func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW(db)
	}

	// Create services
	accountSvc := service.NewAccountService(uowFactory, service.NewStubCurrencyConverter())
	userSvc := service.NewUserService(uowFactory)
	authSvc := service.NewAuthService(uowFactory, service.NewJWTAuthStrategy(uowFactory))

	log.Fatal(webapi.NewApp(accountSvc, userSvc, authSvc).Listen(":3000"))

}
