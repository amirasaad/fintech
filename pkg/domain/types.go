package domain

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
)

// Type aliases for backward compatibility

// Account and related
// Account is an alias for account.Account
// Deprecated: Use account.Account directly.
type Account = account.Account

// Transaction represents a financial transaction in the system.
// Deprecated: Use account.Transaction directly.
type Transaction = account.Transaction

// User and related
type User = user.User

// Error aliases for backward compatibility
var (
	// ErrDepositAmountExceedsMaxSafeInt Account errors
	// Deprecated: Use account.ErrDepositAmountExceedsMaxSafeInt directly.
	ErrDepositAmountExceedsMaxSafeInt = account.ErrDepositAmountExceedsMaxSafeInt
	// Deprecated: Use account.ErrTransactionAmountMustBePositive directly.
	ErrTransactionAmountMustBePositive = account.ErrTransactionAmountMustBePositive
	// Deprecated: Use account.ErrInsufficientFunds directly.
	ErrInsufficientFunds = account.ErrInsufficientFunds
	// Deprecated: Use account.ErrAccountNotFound directly.
	ErrAccountNotFound = account.ErrAccountNotFound
	// Deprecated: Use account.ErrInvalidCurrencyCode directly.
	ErrInvalidCurrencyCode = common.ErrInvalidCurrencyCode
	// Deprecated: Use account.ErrUserUnauthorized directly.
	ErrUserUnauthorized = user.ErrUserUnauthorized

	// Currency-related errors
	// Deprecated: Use exchange.ErrProviderUnavailable directly.
	ErrExchangeRateUnavailable = exchange.ErrProviderUnavailable
	// Deprecated: Use exchange.ErrUnsupportedPair directly.
	ErrUnsupportedCurrencyPair = exchange.ErrUnsupportedPair
)

// ConversionInfo is an alias for exchange.RateInfo
// Deprecated: Use exchange.RateInfo directly.
type ConversionInfo = exchange.RateInfo

// ExchangeRate is an alias for exchange.RateInfo
// Deprecated: Use exchange.RateInfo directly.
type ExchangeRate = exchange.RateInfo

// Deprecated: use money.New
func NewMoney(amount float64, currencyCode money.Code) (m *money.Money, err error) {
	return money.New(amount, currencyCode)
}
