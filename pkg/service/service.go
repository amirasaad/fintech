// Package service provides business logic for the fintech application.
// It is organized into sub-packages for different domains:
// - account: Account and transaction management
// - auth: Authentication and authorization
// - user: User management
// - currency: Currency and exchange rate management
//
// To use services, import the specific sub-package:
//
//	import "github.com/amirasaad/fintech/pkg/service/account"
//	import "github.com/amirasaad/fintech/pkg/service/auth"
//	import "github.com/amirasaad/fintech/pkg/service/user"
//	import "github.com/amirasaad/fintech/pkg/service/currency"
//
// For backward compatibility, you can also import this package directly:
//
//	import "github.com/amirasaad/fintech/pkg/service"
package service

import (
	_ "github.com/amirasaad/fintech/pkg/service/account"
	_ "github.com/amirasaad/fintech/pkg/service/auth"
	_ "github.com/amirasaad/fintech/pkg/service/currency"
	_ "github.com/amirasaad/fintech/pkg/service/user"
)
