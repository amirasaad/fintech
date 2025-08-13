package account_test

import (
	"io"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	domainaccount "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain runs before any tests and applies globally for all tests in the package.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	log.SetOutput(io.Discard)

	exitVal := m.Run()
	os.Exit(exitVal)
}
func TestNewAccount(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	acc, err := domainaccount.New().WithUserID(uuid.New()).Build()
	require.NoError(err)
	assert.NotEmpty(t, acc.ID, "Account ID should not be empty")
}

func TestValidateWithdraw(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	acc, err := domainaccount.New().
		WithUserID(userID).
		WithCurrency(currency.USD).
		WithBalance(10000). // 100.00 USD
		Build()
	require.NoError(t, err)

	amount, err := money.New(50.0, "USD")
	require.NoError(t, err)

	// Test case 1: Successful withdrawal
	t.Run("successful withdrawal", func(t *testing.T) {
		err := acc.ValidateWithdraw(userID, amount)
		assert.NoError(t, err)
	})

	// Test case 2: Unauthorized withdrawal
	t.Run("unauthorized withdrawal", func(t *testing.T) {
		err := acc.ValidateWithdraw(uuid.New(), amount)
		assert.ErrorIs(t, err, domainaccount.ErrNotOwner)
	})

	// Test case 3: Insufficient funds
	t.Run("insufficient funds", func(t *testing.T) {
		amount, err := money.New(200.0, "USD")
		require.NoError(t, err)
		err = acc.ValidateWithdraw(userID, amount)
		assert.ErrorIs(t, err, domainaccount.ErrInsufficientFunds)
	})
}

func TestValidateTransfer(t *testing.T) {
	t.Parallel()
	senderID := uuid.New()
	receiverID := uuid.New()

	sourceAcc, err := domainaccount.New().
		WithUserID(senderID).
		WithCurrency(currency.USD).
		WithBalance(10000). // 100.00 USD
		Build()
	require.NoError(t, err)

	destAcc, err := domainaccount.New().
		WithUserID(receiverID).
		WithCurrency(currency.USD).
		Build()
	require.NoError(t, err)

	amount, err := money.New(50.0, "USD")
	require.NoError(t, err)

	// Test case 1: Successful transfer
	t.Run("successful transfer", func(t *testing.T) {
		err := sourceAcc.ValidateTransfer(senderID, receiverID, destAcc, amount)
		assert.NoError(t, err)
	})

	// Test case 2: Unauthorized transfer
	t.Run("unauthorized transfer", func(t *testing.T) {
		err := sourceAcc.ValidateTransfer(
			uuid.New(), receiverID, destAcc, amount)
		assert.ErrorIs(t, err, domainaccount.ErrNotOwner)
	})

	// Test case 3: Insufficient funds
	t.Run("insufficient funds", func(t *testing.T) {
		amount, err := money.New(200.0, "USD")
		require.NoError(t, err)
		err = sourceAcc.ValidateTransfer(senderID, receiverID, destAcc, amount)
		assert.ErrorIs(t, err, domainaccount.ErrInsufficientFunds)
	})

	// Test case 4: Transfer to same account
	t.Run("transfer to same account", func(t *testing.T) {
		err := sourceAcc.ValidateTransfer(senderID, senderID, sourceAcc, amount)
		assert.ErrorIs(t, err, domainaccount.ErrCannotTransferToSameAccount)
	})

	// Test case 5: Currency mismatch
	t.Run("currency mismatch", func(t *testing.T) {
		amountEUR, err := money.New(50.0, "EUR")
		require.NoError(t, err)
		err = sourceAcc.ValidateTransfer(senderID, receiverID, destAcc, amountEUR)
		assert.ErrorIs(t, err, domainaccount.ErrCurrencyMismatch)
	})
}
