package account

import (
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
)

//revive:disable

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

// ExternalTarget represents the destination for an external withdrawal, such as a bank account or wallet.
type ExternalTarget struct {
	BankAccountNumber     string `json:"bank_account_number,omitempty" validate:"omitempty,min=6,max=34"`
	RoutingNumber         string `json:"routing_number,omitempty" validate:"omitempty,min=6,max=12"`
	ExternalWalletAddress string `json:"external_wallet_address,omitempty" validate:"omitempty,min=6,max=128"`
}

// WithdrawRequest represents the request body for withdrawing funds from an account.
type WithdrawRequest struct {
	Amount         float64        `json:"amount" xml:"amount" form:"amount" validate:"required,gt=0"`
	Currency       string         `json:"currency" validate:"omitempty,len=3,uppercase"`
	ExternalTarget ExternalTarget `json:"external_target" validate:"required"`
}

// TransferRequest represents the request body for transferring funds between accounts.
type TransferRequest struct {
	Amount               float64 `json:"amount" validate:"required,gt=0"`
	Currency             string  `json:"currency" validate:"omitempty,len=3,uppercase,alpha"`
	DestinationAccountID string  `json:"destination_account_id" validate:"required,uuid4"`
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
}

// ConversionInfoDTO holds conversion details for API responses.
type ConversionInfoDTO struct {
	OriginalAmount    float64 `json:"original_amount"`
	OriginalCurrency  string  `json:"original_currency"`
	ConvertedAmount   float64 `json:"converted_amount"`
	ConvertedCurrency string  `json:"converted_currency"`
	ConversionRate    float64 `json:"conversion_rate"`
}

// TransferResponseDTO is the API response for a transfer operation, containing both transactions and a single conversion_info field (like deposit/withdraw).
type TransferResponseDTO struct {
	Outgoing       *TransactionDTO    `json:"outgoing_transaction"`
	Incoming       *TransactionDTO    `json:"incoming_transaction"`
	ConversionInfo *ConversionInfoDTO `json:"conversion_info"`
}

// ToTransactionDTO maps a dto.TransactionRead to a TransactionDTO.
func ToTransactionDTO(tx *dto.TransactionRead) *TransactionDTO {
	if tx == nil {
		return nil
	}
	dto := &TransactionDTO{
		ID:        tx.ID.String(),
		UserID:    tx.UserID.String(),
		AccountID: tx.AccountID.String(),
		Amount:    tx.Amount,
		Currency:  tx.Currency,
		Balance:   tx.Balance,
		CreatedAt: tx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return dto
}

// ToConversionInfoDTO maps provider.ExchangeRate to ConversionInfoDTO.
func ToConversionInfoDTO(convInfo *exchange.RateInfo) *ConversionInfoDTO {
	if convInfo == nil {
		return nil
	}
	return &ConversionInfoDTO{
		OriginalAmount:    0, // Not directly available from RateInfo
		OriginalCurrency:  convInfo.FromCurrency,
		ConvertedAmount:   0, // Not directly available from RateInfo
		ConvertedCurrency: convInfo.ToCurrency,
		ConversionRate:    convInfo.Rate,
	}
}

// ToTransferResponseDTO maps domain transactions and conversion info to a TransferResponseDTO with a single conversion_info field.
func ToTransferResponseDTO(txOut, txIn *dto.TransactionRead, convInfo *exchange.RateInfo) *TransferResponseDTO {
	return &TransferResponseDTO{
		Outgoing:       ToTransactionDTO(txOut),
		Incoming:       ToTransactionDTO(txIn),
		ConversionInfo: ToConversionInfoDTO(convInfo),
	}
}

//revive:enable
