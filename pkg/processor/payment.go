package processor

import (
	"context"

	"github.com/amirasaad/fintech/pkg/service/account"
)

type Event struct {
	Provider  string
	PaymentID string
	Status    string
	RawEvent  any
}

type EventProcessor interface {
	ProcessEvent(event Event) error
}

type DefaultPaymentEventProcessor struct {
	AccountService *account.Service
}

func NewDefaultPaymentEventProcessor(svc *account.Service) *DefaultPaymentEventProcessor {
	return &DefaultPaymentEventProcessor{AccountService: svc}
}

func (p *DefaultPaymentEventProcessor) ProcessEvent(event Event) error {
	return p.AccountService.UpdateTransactionStatusByPaymentID(
		context.Background(),
		event.PaymentID,
		event.Status,
	)
}
