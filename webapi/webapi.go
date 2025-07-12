// Package webapi provides HTTP handlers and API endpoints for the fintech application.
// It is organized into sub-packages for different domains:
// - account: Account and transaction endpoints
// - auth: Authentication endpoints
// - user: User management endpoints
// - currency: Currency and exchange rate endpoints
// - common: Shared utilities, middleware, and test helpers
//
// To use handlers, import the specific sub-package:
//
//	import "github.com/amirasaad/fintech/webapi/account"
//	import "github.com/amirasaad/fintech/webapi/auth"
//	import "github.com/amirasaad/fintech/webapi/user"
//	import "github.com/amirasaad/fintech/webapi/currency"
//	import "github.com/amirasaad/fintech/webapi/common"
//
// For backward compatibility, you can also import this package directly:
//
//	import "github.com/amirasaad/fintech/webapi"
package webapi

import (
	_ "github.com/amirasaad/fintech/webapi/account"
	_ "github.com/amirasaad/fintech/webapi/auth"
	_ "github.com/amirasaad/fintech/webapi/common"
	_ "github.com/amirasaad/fintech/webapi/currency"
	_ "github.com/amirasaad/fintech/webapi/user"
)
