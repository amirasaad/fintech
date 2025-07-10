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

type Transaction = account.Transaction

type ConversionInfo = common.ConversionInfo

// CurrencyConverter and related
type CurrencyConverter = money.CurrencyConverter
type ExchangeRate = money.ExchangeRate

// User and related
type User = user.User

// Error aliases for backward compatibility
var (
	ErrDepositAmountExceedsMaxSafeInt  = account.ErrDepositAmountExceedsMaxSafeInt
	ErrTransactionAmountMustBePositive = account.ErrTransactionAmountMustBePositive
	ErrWithdrawalAmountMustBePositive  = account.ErrWithdrawalAmountMustBePositive
	ErrInsufficientFunds               = account.ErrInsufficientFunds
	ErrAccountNotFound                 = account.ErrAccountNotFound
	ErrInvalidCurrencyCode             = common.ErrInvalidCurrencyCode
	ErrUserUnauthorized                = user.ErrUserUnauthorized

	ErrExchangeRateUnavailable = money.ErrExchangeRateUnavailable
	ErrUnsupportedCurrencyPair = money.ErrUnsupportedCurrencyPair
	ErrExchangeRateExpired     = money.ErrExchangeRateExpired
	ErrExchangeRateInvalid     = money.ErrExchangeRateInvalid
)
