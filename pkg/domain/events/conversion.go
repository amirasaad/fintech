package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
)

// CurrencyConversionRequested is an agnostic event
// for requesting currency conversion in any business flow.
type CurrencyConversionRequested struct {
	FlowEvent
	OriginalRequest Event `json:"-"` // Handle this field manually in MarshalJSON/UnmarshalJSON
	Amount          *money.Money
	To              money.Code
	TransactionID   uuid.UUID

	// Used for JSON serialization
	RequestType    string          `json:"requestType,omitempty"`
	RequestPayload json.RawMessage `json:"requestPayload,omitempty"`
}

func (e CurrencyConversionRequested) Type() string {
	return EventTypeCurrencyConversionRequested.String()
}

// MarshalJSON implements custom JSON marshaling for CurrencyConversionRequested
func (e CurrencyConversionRequested) MarshalJSON() ([]byte, error) {
	// Create an auxiliary type to avoid recursion
	type Alias CurrencyConversionRequested

	// Create a copy to avoid modifying the original
	eCopy := e

	// Marshal the original request if it exists
	if eCopy.OriginalRequest != nil {
		// Store the type name for proper unmarshaling
		eCopy.RequestType = fmt.Sprintf("%T", eCopy.OriginalRequest)

		// Marshal the original request
		var err error
		eCopy.RequestPayload, err = json.Marshal(eCopy.OriginalRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal original request: %w", err)
		}
	}

	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(&eCopy),
	}

	return json.Marshal(aux)
}

// UnmarshalJSON implements custom JSON unmarshaling for CurrencyConversionRequested
func (e *CurrencyConversionRequested) UnmarshalJSON(data []byte) error {
	// Create an auxiliary type to avoid recursion
	type Alias CurrencyConversionRequested
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	// Unmarshal the main fields
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal CurrencyConversionRequested: %w", err)
	}

	// If we have a request type and payload, try to unmarshal it
	if e.RequestType != "" && len(e.RequestPayload) > 0 {
		// Create a new instance of the appropriate type based on RequestType
		var request Event
		switch e.RequestType {
		case "*events.DepositRequested", "events.DepositRequested":
			req := &DepositRequested{}
			if err := json.Unmarshal(e.RequestPayload, req); err != nil {
				return fmt.Errorf("failed to unmarshal DepositRequested: %w", err)
			}
			request = req
		case "*events.WithdrawRequested", "events.WithdrawRequested":
			req := &WithdrawRequested{}
			if err := json.Unmarshal(e.RequestPayload, req); err != nil {
				return fmt.Errorf("failed to unmarshal WithdrawRequested: %w", err)
			}
			request = req
		case "*events.TransferRequested", "events.TransferRequested":
			req := &TransferRequested{}
			if err := json.Unmarshal(e.RequestPayload, req); err != nil {
				return fmt.Errorf("failed to unmarshal TransferRequested: %w", err)
			}
			request = req
		default:
			return fmt.Errorf("unsupported request type: %s", e.RequestType)
		}
		e.OriginalRequest = request
	}

	return nil
}

// CurrencyConverted is an agnostic event for reporting
// the successful result of a currency conversion.
type CurrencyConverted struct {
	CurrencyConversionRequested
	TransactionID   uuid.UUID
	ConvertedAmount *money.Money
	ConversionInfo  *provider.ExchangeInfo `json:"conversionInfo"`
}

func (e CurrencyConverted) Type() string { return EventTypeCurrencyConverted.String() }

// MarshalJSON implements custom JSON marshaling for CurrencyConverted
func (e CurrencyConverted) MarshalJSON() ([]byte, error) {
	// Create an auxiliary structure to explicitly handle all fields
	aux := struct {
		// Embedded CurrencyConversionRequested fields
		ID                   uuid.UUID       `json:"id"`
		FlowType             string          `json:"flowType"`
		UserID               uuid.UUID       `json:"userId"`
		AccountID            uuid.UUID       `json:"accountId"`
		CorrelationID        uuid.UUID       `json:"correlationId"`
		Timestamp            time.Time       `json:"timestamp"`
		Amount               *money.Money    `json:"amount"`
		To                   money.Code      `json:"to"`
		RequestTransactionID uuid.UUID       `json:"requestTransactionId"` // From embedded CCR
		RequestType          string          `json:"requestType,omitempty"`
		RequestPayload       json.RawMessage `json:"requestPayload,omitempty"`

		// CurrencyConverted specific fields
		TransactionID   uuid.UUID              `json:"transactionId"`
		ConvertedAmount *money.Money           `json:"convertedAmount"`
		ConversionInfo  *provider.ExchangeInfo `json:"conversionInfo"`
	}{
		// Copy from embedded CurrencyConversionRequested
		ID:                   e.ID,
		FlowType:             e.FlowType,
		UserID:               e.UserID,
		AccountID:            e.AccountID,
		CorrelationID:        e.CorrelationID,
		Timestamp:            e.Timestamp,
		Amount:               e.Amount,
		To:                   e.To,
		RequestTransactionID: e.TransactionID,
		RequestType:          e.RequestType,
		RequestPayload:       e.RequestPayload,

		// Copy CurrencyConverted specific fields
		TransactionID:   e.TransactionID,
		ConvertedAmount: e.ConvertedAmount, // Keep as pointer
		ConversionInfo:  e.ConversionInfo,
	}

	// Handle OriginalRequest marshaling
	if e.OriginalRequest != nil {
		aux.RequestType = fmt.Sprintf("%T", e.OriginalRequest)
		var err error
		aux.RequestPayload, err = json.Marshal(e.OriginalRequest)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to marshal original request in CurrencyConverted: %w",
				err,
			)
		}
	}

	return json.Marshal(aux)
}

// UnmarshalJSON implements custom JSON unmarshaling for CurrencyConverted
func (e *CurrencyConverted) UnmarshalJSON(data []byte) error {
	// Create an auxiliary structure to match the marshaling format
	aux := struct {
		// Embedded CurrencyConversionRequested fields
		ID                   uuid.UUID       `json:"id"`
		FlowType             string          `json:"flowType"`
		UserID               uuid.UUID       `json:"userId"`
		AccountID            uuid.UUID       `json:"accountId"`
		CorrelationID        uuid.UUID       `json:"correlationId"`
		Timestamp            time.Time       `json:"timestamp"`
		Amount               money.Money     `json:"amount"`
		To                   money.Code      `json:"to"`
		RequestTransactionID uuid.UUID       `json:"requestTransactionId"` // From embedded CCR
		RequestType          string          `json:"requestType,omitempty"`
		RequestPayload       json.RawMessage `json:"requestPayload,omitempty"`

		// CurrencyConverted specific fields
		TransactionID   uuid.UUID       `json:"transactionId"`
		ConvertedAmount *money.Money    `json:"convertedAmount"`
		ConversionInfo  json.RawMessage `json:"conversionInfo"`
	}{}

	// Unmarshal the main fields
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal CurrencyConverted: %w", err)
	}

	// Copy fields to the embedded CurrencyConversionRequested
	e.ID = aux.ID
	e.FlowType = aux.FlowType
	e.UserID = aux.UserID
	e.AccountID = aux.AccountID
	e.CorrelationID = aux.CorrelationID
	e.Timestamp = aux.Timestamp
	// Create a new money.Money pointer and copy the value
	if aux.Amount != (money.Money{}) {
		amount := aux.Amount // Create a copy
		e.Amount = &amount
	} else {
		e.Amount = nil
	}
	e.To = aux.To
	e.TransactionID = aux.RequestTransactionID
	e.RequestType = aux.RequestType
	e.RequestPayload = aux.RequestPayload

	// Copy CurrencyConverted specific fields
	e.TransactionID = aux.TransactionID
	e.ConvertedAmount = aux.ConvertedAmount

	// Parse ConversionInfo if present
	if len(aux.ConversionInfo) > 0 {
		var info provider.ExchangeInfo
		if err := json.Unmarshal(aux.ConversionInfo, &info); err != nil {
			return fmt.Errorf("failed to unmarshal ConversionInfo: %w", err)
		}
		e.ConversionInfo = &info
	}

	// Handle the OriginalRequest reconstruction
	if aux.RequestType != "" && len(aux.RequestPayload) > 0 {
		// Create a new instance of the appropriate type based on RequestType
		var request Event
		switch aux.RequestType {
		case "*events.DepositRequested", "events.DepositRequested":
			req := &DepositRequested{}
			if err := json.Unmarshal(aux.RequestPayload, req); err != nil {
				return fmt.Errorf(
					"failed to unmarshal DepositRequested in CurrencyConverted: %w",
					err,
				)
			}
			request = req
		case "*events.WithdrawRequested", "events.WithdrawRequested":
			req := &WithdrawRequested{}
			if err := json.Unmarshal(aux.RequestPayload, req); err != nil {
				return fmt.Errorf(
					"failed to unmarshal WithdrawRequested in CurrencyConverted: %w",
					err,
				)
			}
			request = req
		case "*events.TransferRequested", "events.TransferRequested":
			req := &TransferRequested{}
			if err := json.Unmarshal(aux.RequestPayload, req); err != nil {
				return fmt.Errorf(
					"failed to unmarshal TransferRequested in CurrencyConverted: %w",
					err,
				)
			}
			request = req
		default:
			return fmt.Errorf(
				"unsupported request type in CurrencyConverted: %s",
				aux.RequestType,
			)
		}
		e.OriginalRequest = request
	}

	return nil
}

// CurrencyConversionFailed is an event for reporting a failed currency conversion.
type CurrencyConversionFailed struct {
	FlowEvent
	TransactionID uuid.UUID
	Amount        money.Money
	To            money.Code
	Reason        string
}

func (e CurrencyConversionFailed) Type() string {
	return EventTypeCurrencyConversionFailed.String()
}
