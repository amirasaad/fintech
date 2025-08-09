package payment

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/stretchr/testify/mock"
)

// createValidPaymentCompletedEvent creates a valid PaymentCompletedEvent
func createValidPaymentCompletedEvent(
	h *testutils.TestHelper,
) *events.PaymentCompleted {
	// Use the amount directly from the test helper
	amount := h.Amount

	// Create a small fee amount (1% of the amount)
	feeAmount, err := money.New(amount.AmountFloat(), "USD")
	if err != nil {
		h.T.Fatalf("failed to create fee amount: %v", err)
	}

	return events.NewPaymentCompleted(
		&events.FlowEvent{
			ID:            h.EventID,
			CorrelationID: h.CorrelationID,
			FlowType:      "payment",
		},
		func(pc *events.PaymentCompleted) {
			pc.PaymentID = h.PaymentID
			pc.TransactionID = h.TransactionID
			pc.Amount = amount
			// Set provider fee if needed by the test
			pc.ProviderFee = account.Fee{
				Amount: feeAmount,
			}
			pc.Status = "completed"
		},
	)
}

// createValidPaymentFailedEvent creates a valid PaymentFailedEvent
func createValidPaymentFailedEvent(
	h *testutils.TestHelper,
) *events.PaymentFailed {
	return events.NewPaymentFailed(
		&events.FlowEvent{
			ID:            h.EventID,
			CorrelationID: h.CorrelationID,
			FlowType:      "payment",
		}, func(pf *events.PaymentFailed) {
			pf.PaymentID = h.PaymentID
			pf.TransactionID = h.TransactionID
		}).WithReason("payment processing failed")

}

// setupSuccessfulTest configures mocks for a successful payment completion
func setupSuccessfulTest(h *testutils.TestHelper) {
	// Ensure the amount is in the correct currency
	amount, err := money.New(h.Amount.AmountFloat(), "USD")
	if err != nil {
		h.T.Fatalf("failed to create money amount: %v", err)
	}

	// Setup test transaction
	tx := &dto.TransactionRead{
		ID:        h.TransactionID,
		UserID:    h.UserID,
		AccountID: h.AccountID,
		PaymentID: h.PaymentID,
		Status:    "pending",
		Currency:  "USD",
		Amount:    amount.AmountFloat(),
	}

	// Setup test account
	testAccount := &dto.AccountRead{
		ID:       h.AccountID,
		UserID:   h.UserID,
		Balance:  amount.AmountFloat(),
		Currency: "USD",
	}

	doFn := func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
		h.UOW.
			EXPECT().
			GetRepository(
				(*transaction.Repository)(nil),
			).
			Return(
				h.MockTxRepo, nil,
			)

		h.MockTxRepo.
			EXPECT().
			GetByPaymentID(ctx, h.PaymentID).
			Return(tx, nil).
			Once()

		h.UOW.
			EXPECT().
			GetRepository(
				(*repoaccount.Repository)(nil),
			).
			Return(
				h.MockAccRepo, nil,
			).Once()
		h.MockAccRepo.
			EXPECT().
			Get(ctx, h.AccountID).
			Return(testAccount, nil).
			Once()

		h.MockAccRepo.
			EXPECT().
			Update(ctx, h.AccountID, mock.AnythingOfType("dto.AccountUpdate")).
			Return(nil).
			Once()
		h.MockTxRepo.
			EXPECT().
			Update(ctx, h.TransactionID, mock.AnythingOfType("dto.TransactionUpdate")).
			Return(nil).
			Once()

		err := fn(h.UOW)
		return err
	}

	h.UOW.
		EXPECT().
		Do(
			h.Ctx,
			mock.AnythingOfType("func(repository.UnitOfWork) error")).
		RunAndReturn(doFn).
		Once()
}
