package account

import (
	"github.com/amirasaad/fintech/pkg/domain"
)

// CreateAccountRequest represents the request body for creating a new account.
type CreateAccountRequest struct {
	Currency string `json:"currency" validate:"omitempty,len=3,uppercase,alpha"`
}

// DepositRequest represents the request body for depositing funds into an account.
type DepositRequest struct {
	Amount   float64 `json:"amount" xml:"amount" form:"amount" validate:"required,gt=0"`
	Currency string  `json:"currency" validate:"omitempty,len=3,uppercase"`
}

// WithdrawRequest represents the request body for withdrawing funds from an account.
type WithdrawRequest struct {
	Amount   float64 `json:"amount" xml:"amount" form:"amount" validate:"required,gt=0"`
	Currency string  `json:"currency" validate:"omitempty,len=3,uppercase"`
}

// ConversionResponse wraps a transaction and conversion details if a currency conversion occurred.
type ConversionResponse struct {
	Transaction       *domain.Transaction `json:"transaction"`
	OriginalAmount    float64             `json:"original_amount,omitempty"`
	OriginalCurrency  string              `json:"original_currency,omitempty"`
	ConvertedAmount   float64             `json:"converted_amount,omitempty"`
	ConvertedCurrency string              `json:"converted_currency,omitempty"`
	ConversionRate    float64             `json:"conversion_rate,omitempty"`
}

// TransactionDTO is the API response representation of a transaction.
type TransactionDTO struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	AccountID string  `json:"account_id"`
	Amount    float64 `json:"amount"`
	Balance   float64 `json:"balance"`
	CreatedAt string  `json:"created_at"`
	Currency  string  `json:"currency"`

	// Conversion fields (only present if conversion occurred)
	OriginalAmount   *float64 `json:"original_amount,omitempty"`
	OriginalCurrency *string  `json:"original_currency,omitempty"`
	ConversionRate   *float64 `json:"conversion_rate,omitempty"`
}

// ConversionResponseDTO wraps a transaction and conversion details for API responses.
type ConversionResponseDTO struct {
	Transaction       *TransactionDTO `json:"transaction"`
	OriginalAmount    float64         `json:"original_amount,omitempty"`
	OriginalCurrency  string          `json:"original_currency,omitempty"`
	ConvertedAmount   float64         `json:"converted_amount,omitempty"`
	ConvertedCurrency string          `json:"converted_currency,omitempty"`
	ConversionRate    float64         `json:"conversion_rate,omitempty"`
}

// ToTransactionDTO maps a domain.Transaction to a TransactionDTO.
func ToTransactionDTO(tx *domain.Transaction) *TransactionDTO {
	if tx == nil {
		return nil
	}
	dto := &TransactionDTO{
		ID:        tx.ID.String(),
		UserID:    tx.UserID.String(),
		AccountID: tx.AccountID.String(),
		Amount:    float64(tx.Amount) / 100.0, // assuming cents
		Balance:   float64(tx.Balance) / 100.0,
		CreatedAt: tx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Currency:  string(tx.Currency),
	}

	// Include conversion fields if they exist
	if tx.OriginalAmount != nil {
		dto.OriginalAmount = tx.OriginalAmount
	}
	if tx.OriginalCurrency != nil {
		dto.OriginalCurrency = tx.OriginalCurrency
	}
	if tx.ConversionRate != nil {
		dto.ConversionRate = tx.ConversionRate
	}

	return dto
}

// ToConversionResponseDTO maps a transaction and conversion info to a ConversionResponseDTO.
func ToConversionResponseDTO(tx *domain.Transaction, convInfo *domain.ConversionInfo) *ConversionResponseDTO {
	// If conversion info is provided (from service layer), use it
	if convInfo != nil {
		return &ConversionResponseDTO{
			Transaction:       ToTransactionDTO(tx),
			OriginalAmount:    convInfo.OriginalAmount,
			OriginalCurrency:  convInfo.OriginalCurrency,
			ConvertedAmount:   convInfo.ConvertedAmount,
			ConvertedCurrency: convInfo.ConvertedCurrency,
			ConversionRate:    convInfo.ConversionRate,
		}
	}

	// If no conversion info provided but transaction has stored conversion data, use that
	if tx.OriginalAmount != nil && tx.OriginalCurrency != nil && tx.ConversionRate != nil {
		return &ConversionResponseDTO{
			Transaction:       ToTransactionDTO(tx),
			OriginalAmount:    *tx.OriginalAmount,
			OriginalCurrency:  *tx.OriginalCurrency,
			ConvertedAmount:   float64(tx.Amount) / 100.0, // Convert from cents
			ConvertedCurrency: string(tx.Currency),
			ConversionRate:    *tx.ConversionRate,
		}
	}

	// No conversion occurred
	return nil
}
