package payment

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
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

	return events.NewPaymentCompleted(
		&events.FlowEvent{
			ID:            h.EventID,
			CorrelationID: h.CorrelationID,
			FlowType:      "payment",
		},
		func(pc *events.PaymentCompleted) {
			paymentID := "test-payment-id"
			pc.PaymentID = &paymentID
			pc.TransactionID = h.TransactionID
			pc.Amount = amount
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
			if h.PaymentID != nil {
				pf.PaymentID = h.PaymentID
			}
			pf.TransactionID = h.TransactionID
		}).WithReason("payment processing failed")

}

// setupSuccessfulTest configures mocks for a successful payment completion
func setupSuccessfulTest(h *testutils.TestHelper) {
	// Use the amount directly from the test helper
	amount := h.Amount

	// Setup payment ID to match what createValidPaymentCompletedEvent creates
	paymentID := "test-payment-id"
	if h.PaymentID == nil {
		h.PaymentID = &paymentID
	}

	// Setup test transaction with payment ID matching the event
	tx := &dto.TransactionRead{
		ID:        h.TransactionID,
		UserID:    h.UserID,
		AccountID: h.AccountID,
		PaymentID: &paymentID,
		Status:    "pending",
		Currency:  amount.CurrencyCode().String(),
		Amount:    amount.AmountFloat(),
	}

	// Setup test account
	testAccount := &dto.AccountRead{
		ID:       h.AccountID,
		UserID:   h.UserID,
		Balance:  amount.AmountFloat(),
		Currency: amount.CurrencyCode().String(),
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
			GetByPaymentID(ctx, paymentID).
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

		// Setup mock expectations for account update
		h.MockAccRepo.EXPECT().
			Update(ctx, h.AccountID, mock.MatchedBy(func(update dto.AccountUpdate) bool {
				// Verify the account balance is being updated correctly
				return update.Balance != nil && *update.Balance > 0
			})).
			Return(nil).
			Once()

		// Setup mock expectations for transaction update
		h.MockTxRepo.EXPECT().
			Update(ctx, h.TransactionID, mock.MatchedBy(func(update dto.TransactionUpdate) bool {
				// Verify the transaction status is being updated to "completed"
				return update.Status != nil && *update.Status == "completed"
			})).
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
