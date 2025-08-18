package fees

import (
	"context"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
	"github.com/amirasaad/fintech/pkg/money"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	name       string
	setupMocks func(
		*testutils.TestHelper,
		*dto.TransactionRead,
		*dto.AccountRead,
		account.Fee,
	)
	expectedUpdateTx  *dto.TransactionUpdate
	expectedUpdateAcc *dto.AccountUpdate
	expectedErr       error
	transactionID     uuid.UUID
	fee               account.Fee
}

func TestFeeCalculator_ApplyFees(t *testing.T) {
	ctx := context.Background()
	tests := []testCase{
		{
			name: "successfully applies fees",
			setupMocks: func(
				h *testutils.TestHelper,
				tx *dto.TransactionRead,
				acc *dto.AccountRead,
				fee account.Fee,
			) {
				h.MockTxRepo.EXPECT().
					Get(h.Ctx, tx.ID).
					Return(tx, nil).
					Once()

				// Use the fee amount from the test data (100.00 in cents)
				feeAmount := int64(10000) // $100.00 in cents
				updateTx := dto.TransactionUpdate{
					Fee: &feeAmount,
				}
				h.MockTxRepo.EXPECT().
					Update(h.Ctx, tx.ID, updateTx).
					Return(nil).
					Once()

				// Set initial balance and calculate expected balance after fee
				initialBalance := int64(2000000)              // $20,000.00 in cents
				expectedBalance := initialBalance - feeAmount // Initial balance - fee
				updateAcc := dto.AccountUpdate{
					Balance: &expectedBalance,
					Status:  nil,
				}
				h.MockAccRepo.EXPECT().
					Get(h.Ctx, acc.ID).
					Return(acc, nil).
					Once()

				h.MockAccRepo.EXPECT().
					Update(h.Ctx, acc.ID, updateAcc).
					Return(nil).
					Once()
			},
			expectedUpdateTx: func() *dto.TransactionUpdate {
				feeAmount := int64(10000) // $100.00 in cents
				return &dto.TransactionUpdate{
					Fee: &feeAmount,
				}
			}(),
			expectedUpdateAcc: func() *dto.AccountUpdate {
				feeAmount := int64(10000)             // $100.00 in cents
				balance := int64(2000000) - feeAmount // $20,000.00 - $100.00 = $19,900.00
				return &dto.AccountUpdate{
					Balance: &balance,
				}
			}(),
			transactionID: uuid.New(),
			fee: account.Fee{
				Amount: money.Must(100, money.USD.ToCurrency()), // $100.00
				Type:   account.FeeProvider,
			},
		},
		{
			name:          "transaction not found",
			transactionID: uuid.New(),
			setupMocks: func(
				h *testutils.TestHelper,
				_ *dto.TransactionRead,
				_ *dto.AccountRead,
				_ account.Fee,
			) {
				h.MockTxRepo.EXPECT().
					Get(h.Ctx, h.TransactionID).
					Return((*dto.TransactionRead)(nil), account.ErrTransactionNotFound).
					Once()
			},
			expectedErr: account.ErrTransactionNotFound,
		},
		{
			name: "account not found",
			setupMocks: func(
				h *testutils.TestHelper,
				tx *dto.TransactionRead,
				_ *dto.AccountRead,
				_ account.Fee,
			) {
				h.MockTxRepo.EXPECT().
					Get(h.Ctx, tx.ID).
					Return(tx, nil).
					Once()

				// The implementation will try to update the transaction with the fee
				// even if the account is not found, so we need to expect this call
				h.MockTxRepo.EXPECT().
					Update(h.Ctx, tx.ID, mock.AnythingOfType("dto.TransactionUpdate")).
					Return(nil).
					Once()

				h.MockAccRepo.EXPECT().
					Get(h.Ctx, tx.AccountID).
					Return(nil, account.ErrAccountNotFound).
					Once()
			},
			expectedErr: account.ErrAccountNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test data
			accountID := uuid.New()
			userID := uuid.New()
			txID := tt.transactionID
			if txID == uuid.Nil {
				txID = uuid.New()
			}

			tx := &dto.TransactionRead{
				ID:        txID,
				AccountID: accountID,
				UserID:    userID,
				Status:    "completed",
				Amount:    10000, // $100.00
				Fee:       0,
				Currency:  "USD",
			}

			acc := &dto.AccountRead{
				ID:       accountID,
				UserID:   userID,
				Balance:  20000, // $200.00
				Currency: "USD",
			}

			// Create fee amount in smallest currency unit (10000 = 100.00)
			feeAmount := money.Must(100, money.USD.ToCurrency()) // $100.00
			fee := account.Fee{
				Amount: feeAmount,
				Type:   account.FeeProvider,
			}

			// Setup test helper and mocks
			h := testutils.New(t).
				WithAccountID(accountID).
				WithUserID(userID).
				WithTransactionID(tx.ID)

			// Setup test-specific mocks
			if tt.setupMocks != nil {
				tt.setupMocks(h, tx, acc, fee)
			}

			// Create calculator and apply fees
			calculator := NewFeeCalculator(h.MockTxRepo, h.MockAccRepo, h.Logger)
			err := calculator.ApplyFees(ctx, tx.ID, fee)

			// Verify results
			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewFeeCalculator(t *testing.T) {
	txRepo := mocks.NewTransactionRepository(t)
	accRepo := mocks.NewAccountRepository(t)
	logger := slog.Default()

	tests := []struct {
		name    string
		txRepo  repotransaction.Repository
		accRepo repoaccount.Repository
		logger  *slog.Logger
		wantErr bool
	}{
		{
			name:    "valid parameters",
			txRepo:  txRepo,
			accRepo: accRepo,
			logger:  logger,
			wantErr: false,
		},
		{
			name:    "nil transaction repository",
			txRepo:  nil,
			accRepo: accRepo,
			logger:  logger,
			wantErr: true,
		},
		{
			name:    "nil account repository",
			txRepo:  txRepo,
			accRepo: nil,
			logger:  logger,
			wantErr: true,
		},
		{
			name:    "nil logger",
			txRepo:  txRepo,
			accRepo: accRepo,
			logger:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculator := NewFeeCalculator(tt.txRepo, tt.accRepo, tt.logger)
			if tt.wantErr {
				assert.Nil(t, calculator)
			} else {
				assert.NotNil(t, calculator)
			}
		})
	}
}
