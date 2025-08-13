package domain

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
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
type ConversionInfo = currency.Info

// CurrencyConverter and related
type CurrencyConverter = currency.Converter

// ExchangeRate represents an exchange rate between two currencies.
type ExchangeRate = currency.ExchangeRate

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

	ErrExchangeRateUnavailable = currency.ErrExchangeRateUnavailable
	ErrUnsupportedCurrencyPair = currency.ErrUnsupportedCurrencyPair
	ErrExchangeRateExpired     = currency.ErrExchangeRateExpired
	ErrExchangeRateInvalid     = currency.ErrExchangeRateInvalid
)
