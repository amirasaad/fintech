package account

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/google/uuid"
)

// Withdraw removes funds from the specified account and creates a transaction record.
// The method supports multi-currency withdrawals with automatic currency conversion
// when the withdrawal currency differs from the account currency.
//
// The operation is wrapped with automatic transaction management and includes
// comprehensive validation, error handling, and logging.
//
// Key Features:
// - Multi-currency support with real-time conversion
// - Automatic transaction record creation
// - Comprehensive validation (positive amounts, valid currencies)
// - User authorization checks
// - Insufficient funds validation
// - Detailed logging for observability
//
// Parameters:
//   - userID: The UUID of the user making the withdrawal (must own the account)
//   - accountID: The UUID of the account to withdraw from
//   - amount: The amount to withdraw (must be positive)
//   - currencyCode: The ISO 4217 currency code of the withdrawal amount
//
// Returns:
//   - A pointer to the created transaction record
//   - A pointer to conversion information (if currency conversion occurred)
//   - An error if the operation fails
//
// Currency Conversion:
// If the withdrawal currency differs from the account currency, the system will:
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
//	tx, convInfo, err := service.Withdraw(userID, accountID, 50.0, currency.Code("USD"))
//	if err != nil {
//	    log.Error("Withdraw failed", "error", err)
//	    return
//	}
//	if convInfo != nil {
//	    log.Info("Currency conversion applied",
//	        "originalAmount", convInfo.OriginalAmount,
//	        "convertedAmount", convInfo.ConvertedAmount,
//	        "rate", convInfo.ConversionRate)
//	}
func (s *AccountService) Withdraw(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (
	tx *account.Transaction,
	convInfo *common.ConversionInfo,
	err error,
) {
	req := operationRequest{
		userID:       userID,
		accountID:    accountID,
		amount:       amount,
		currencyCode: currencyCode,
		operation:    OperationWithdraw,
	}
	
	result, err := s.executeOperation(req, &withdrawHandler{})
	if err != nil {
		return nil, nil, err
	}
	
	return result.transaction, result.convInfo, nil
}