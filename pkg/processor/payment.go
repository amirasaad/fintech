package processor

import (
	"context"
	"fmt"

	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
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
	uow            repository.UnitOfWork
}

func NewDefaultPaymentEventProcessor(
	svc *account.Service,
	uow repository.UnitOfWork,
) *DefaultPaymentEventProcessor {
	return &DefaultPaymentEventProcessor{
		AccountService: svc,
		uow:            uow,
	}
}

func (p *DefaultPaymentEventProcessor) ProcessEvent(event Event) error {
	// First, we need to get the transaction ID using the payment ID
	// since the event only contains the payment ID
	ctx := context.Background()

	// Get the transaction by payment ID to find the transaction ID
	tx, err := p.getTransactionByPaymentID(ctx, event.PaymentID)
	if err != nil {
		return fmt.Errorf("failed to get transaction by payment ID %s: %w", event.PaymentID, err)
	}

	// Update the transaction status using the transaction ID
	update := dto.TransactionUpdate{
		Status: &event.Status,
	}

	return p.AccountService.UpdateTransaction(ctx, tx.ID, update)
}

// getTransactionByPaymentID retrieves a transaction by its payment ID
func (p *DefaultPaymentEventProcessor) getTransactionByPaymentID(
	ctx context.Context,
	paymentID string,
) (*accountdomain.Transaction, error) {
	// Use the provided unit of work to get the transaction by payment ID
	var tx *accountdomain.Transaction
	err := p.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			return err
		}
		tx, err = txRepo.GetByPaymentID(paymentID)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction by payment ID: %w", err)
	}

	if tx == nil {
		return nil, fmt.Errorf("transaction not found for payment ID: %s", paymentID)
	}

	return tx, nil
}
