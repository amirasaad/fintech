package account

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/google/uuid"
)

// Deposit adds funds to the specified account and creates a transaction record.
// The method supports multi-currency deposits with automatic currency conversion
// when the deposit currency differs from the account currency.
//
// The operation is wrapped with automatic transaction management and includes
// comprehensive validation, error handling, and logging.
//
// Key Features:
// - Multi-currency support with real-time conversion
// - Automatic transaction record creation
// - Comprehensive validation (positive amounts, valid currencies)
// - User authorization checks
// - Detailed logging for observability
//
// Parameters:
//   - userID: The UUID of the user making the deposit (must own the account)
//   - accountID: The UUID of the account to deposit into
//   - amount: The amount to deposit (must be positive)
//   - currencyCode: The ISO 4217 currency code of the deposit amount
//
// Returns:
//   - A pointer to the created transaction record
//   - A pointer to conversion information (if currency conversion occurred)
//   - An error if the operation fails
//
// Currency Conversion:
// If the deposit currency differs from the account currency, the system will:
// 1. Fetch real-time exchange rates from the configured provider
// 2. Convert the amount to the account's currency
// 3. Store conversion details for audit purposes
// 4. Update the account balance with the converted amount
//
// Error Scenarios:
// - Account not found: Returns domain.ErrAccountNotFound
// - User not authorized: Returns domain.ErrUserUnauthorized
// - Invalid amount: Returns domain.ErrTransactionAmountMustBePositive
// - Invalid currency: Returns domain.ErrInvalidCurrencyCode
// - Insufficient funds: Returns domain.ErrInsufficientFunds
// - Conversion failure: Returns conversion service error
//
// Example:
//
//	tx, convInfo, err := service.Deposit(userID, accountID, 100.0, currency.Code("EUR"))
//	if err != nil {
//	    log.Error("Deposit failed", "error", err)
//	    return
//	}
//	if convInfo != nil {
//	    log.Info("Currency conversion applied",
//	        "originalAmount", convInfo.OriginalAmount,
//	        "convertedAmount", convInfo.ConvertedAmount,
//	        "rate", convInfo.ConversionRate)
//	}
func (s *AccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	req := operationRequest{
		userID:       userID,
		accountID:    accountID,
		amount:       amount,
		currencyCode: currencyCode,
		operation:    OperationDeposit,
	}
	
	result, err := s.executeOperation(req, &depositHandler{})
	if err != nil {
		return nil, nil, err
	}
	
	return result.transaction, result.convInfo, nil
}