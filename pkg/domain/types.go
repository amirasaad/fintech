package domain

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
)

// Type aliases for backward compatibility

// Account and related
// Account is an alias for account.Account
// Deprecated: Use account.Account directly.
type Account = account.Account

// Transaction represents a financial transaction in the system.
// Deprecated: Use account.Transaction directly.
type Transaction = account.Transaction

// ConversionInfo contains information about currency conversion for a transaction.
type ConversionInfo = common.ConversionInfo

// CurrencyConverter and related
type CurrencyConverter = money.CurrencyConverter

// ExchangeRate represents an exchange rate between two currencies.
type ExchangeRate = money.ExchangeRate

// User and related
type User = user.User

// Error aliases for backward compatibility
var (
	ErrDepositAmountExceedsMaxSafeInt  = account.ErrDepositAmountExceedsMaxSafeInt
	ErrTransactionAmountMustBePositive = account.ErrTransactionAmountMustBePositive
	ErrInsufficientFunds               = account.ErrInsufficientFunds
	ErrAccountNotFound                 = account.ErrAccountNotFound
	ErrInvalidCurrencyCode             = common.ErrInvalidCurrencyCode
	ErrUserUnauthorized                = user.ErrUserUnauthorized

	ErrExchangeRateUnavailable = money.ErrExchangeRateUnavailable
	ErrUnsupportedCurrencyPair = money.ErrUnsupportedCurrencyPair
	ErrExchangeRateExpired     = money.ErrExchangeRateExpired
	ErrExchangeRateInvalid     = money.ErrExchangeRateInvalid
)

// Event represents a domain event in the system.
type Event interface {
	Type() string
}
