package payment

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFailedHandler(t *testing.T) {
	t.Run("returns error for incorrect event type", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleFailed(h.Bus, h.UOW, h.Logger)

		// The handler should not call any repository methods for incorrect event types
		h.UOW.EXPECT().GetRepository(mock.Anything).Unset()

		err := handler(h.Ctx, &testutils.TestEvent{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected PaymentFailed event, got")
	})

	t.Run("handles error from unit of work", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		// Setup mocks

		h.UOW.EXPECT().
			GetRepository((*repotransaction.Repository)(nil)).
			Return(h.MockTxRepo, nil).
			Once()

		// Setup the mock for the Update method
		status := "failed"
		h.MockTxRepo.EXPECT().
			Update(h.Ctx, h.TransactionID, dto.TransactionUpdate{
				PaymentID: h.PaymentID,
				Status:    &status,
			}).
			Return(nil).
			Once()

		// Setup the mock for the Do method to return an error
		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			Return(assert.AnError).
			Once()

		err := HandleFailed(h.Bus, h.UOW, h.Logger)(h.Ctx, createValidPaymentFailedEvent(h))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
	})

	t.Run("successful payment failure handling", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleFailed(h.Bus, h.UOW, h.Logger)

		// Setup mocks for successful handling

		h.UOW.EXPECT().
			GetRepository((*repotransaction.Repository)(nil)).
			Return(h.MockTxRepo, nil).
			Once()

		// Setup the mock for the Update method
		status := "failed"
		h.MockTxRepo.EXPECT().
			Update(h.Ctx, h.TransactionID, dto.TransactionUpdate{
				PaymentID: h.PaymentID,
				Status:    &status,
			}).
			Return(nil).
			Once()

		// Setup the mock for the Do method to succeed
		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			Return(nil).
			Once()

		err := handler(h.Ctx, createValidPaymentFailedEvent(h))
		assert.NoError(t, err)
	})
}
