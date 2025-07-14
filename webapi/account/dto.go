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
	Amount      float64 `json:"amount" xml:"amount" form:"amount" validate:"required,gt=0"`
	Currency    string  `json:"currency" validate:"omitempty,len=3,uppercase"`
	MoneySource string  `json:"money_source" validate:"required,min=2,max=64"`
}

// WithdrawRequest represents the request body for withdrawing funds from an account.
type WithdrawRequest struct {
	Amount   float64 `json:"amount" xml:"amount" form:"amount" validate:"required,gt=0"`
	Currency string  `json:"currency" validate:"omitempty,len=3,uppercase"`
}

// TransferRequest represents the request body for transferring funds between accounts.
type TransferRequest struct {
	Amount               float64 `json:"amount" validate:"required,gt=0"`
	Currency             string  `json:"currency" validate:"omitempty,len=3,uppercase,alpha"`
	DestinationAccountID string  `json:"destination_account_id" validate:"required,uuid4"`
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
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	AccountID   string  `json:"account_id"`
	Amount      float64 `json:"amount"`
	Balance     float64 `json:"balance"`
	CreatedAt   string  `json:"created_at"`
	Currency    string  `json:"currency"`
	MoneySource string  `json:"money_source"`

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

// TransferResponseDTO is the API response for a transfer operation, containing both outgoing and incoming transactions.
type TransferResponseDTO struct {
	Outgoing *TransactionDTO `json:"outgoing_transaction"`
	Incoming *TransactionDTO `json:"incoming_transaction"`
}

// ToTransactionDTO maps a domain.Transaction to a TransactionDTO.
func ToTransactionDTO(tx *domain.Transaction) *TransactionDTO {
	if tx == nil {
		return nil
	}
	dto := &TransactionDTO{
		ID:          tx.ID.String(),
		UserID:      tx.UserID.String(),
		AccountID:   tx.AccountID.String(),
		Amount:      tx.Amount.AmountFloat(),
		Currency:    string(tx.Amount.Currency()),
		Balance:     tx.Balance.AmountFloat(),
		CreatedAt:   tx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		MoneySource: string(tx.MoneySource),
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

	// No conversion occurred
	return nil
}

// ToTransferResponseDTO maps domain transactions to a TransferResponseDTO.
func ToTransferResponseDTO(txOut, txIn *domain.Transaction) *TransferResponseDTO {
	return &TransferResponseDTO{
		Outgoing: ToTransactionDTO(txOut),
		Incoming: ToTransactionDTO(txIn),
	}
}
