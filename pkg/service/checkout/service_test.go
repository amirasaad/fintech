package checkout

import (
	"context"
	"testing"
	"time"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CreateSession(t *testing.T) {
	tests := []struct {
		name        string
		currency    string
		amount      int64
		expiresIn   time.Duration
		registryErr error
		wantErr     bool
	}{
		{"valid session", "USD", 1000,
			time.Hour, nil, false},
		{"invalid currency", "XXX", 1000,
			time.Hour, nil, true},
		{"negative amount", "USD", -100,
			time.Hour, nil, true},
		{"registry error", "USD", 1000,
			time.Hour, assert.AnError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := mocks.NewRegistryProvider(t)
			if !tt.wantErr || tt.registryErr != nil {
				mr.On(
					"Register",
					mock.Anything,
					mock.Anything,
				).Return(tt.registryErr)
			}

			svc := New(mr)
			_, err := svc.CreateSession(
				context.Background(),
				"test-session",
				"test-id",
				uuid.New(),
				uuid.New(),
				uuid.New(),
				tt.amount,
				tt.currency,
				"https://checkout.example.com",
				tt.expiresIn,
			)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mr.AssertExpectations(t)
		})
	}
}

func TestService_GetSession(t *testing.T) {
	transactionID := uuid.New()
	userID := uuid.New()
	accountID := uuid.New()

	sessionEntity := registry.NewBaseEntity(
		"valid-session",
		"checkout_session_123",
	)
	sessionEntity.SetMetadata("transaction_id", transactionID.String())
	sessionEntity.SetMetadata("user_id", userID.String())
	sessionEntity.SetMetadata("account_id", accountID.String())
	sessionEntity.SetMetadata("amount", "1000")
	sessionEntity.SetMetadata("currency", "USD")
	sessionEntity.SetMetadata("status", "pending")
	sessionEntity.SetMetadata("checkout_url", "https://example.com/checkout/123")
	sessionEntity.SetMetadata("created_at", time.Now().Format(time.RFC3339))
	sessionEntity.SetMetadata("expires_at", time.Now().Add(time.Hour).Format(time.RFC3339))

	tests := []struct {
		name      string
		sessionID string
		entity    registry.Entity
		err       error
		wantErr   bool
	}{
		{"valid session", "valid-session", sessionEntity, nil, false},
		{"not found", "missing-session", nil, assert.AnError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := mocks.NewRegistryProvider(t)
			mr.On("Get", mock.Anything, tt.sessionID).Return(tt.entity, tt.err)

			svc := New(mr)
			session, err := svc.GetSession(context.Background(), tt.sessionID)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, session)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, session)
				// Additional assertions for valid session
				assert.NotEqual(t, uuid.Nil, session.TransactionID)
				assert.NotEqual(t, uuid.Nil, session.UserID)
				assert.NotEqual(t, uuid.Nil, session.AccountID)
				assert.Positive(t, session.Amount)
				assert.NotEmpty(t, session.Currency)
				assert.NotEmpty(t, session.Status)
				assert.NotEmpty(t, session.CheckoutURL)
				assert.False(t, session.CreatedAt.IsZero())
				assert.False(t, session.ExpiresAt.IsZero())
			}
		})
	}
}

func TestSession_Validate(t *testing.T) {
	validUUID := uuid.New()
	tests := []struct {
		name    string
		session Session
		wantErr string
	}{
		{
			"valid",
			Session{
				ID: "valid", TransactionID: validUUID,
				UserID: validUUID, AccountID: validUUID,
				Amount: 1000, Currency: "USD",
			},
			"",
		},
		{"empty ID", Session{ID: ""},
			"session ID cannot be empty"},
		{"nil transaction ID", Session{ID: "test",
			UserID: validUUID, AccountID: validUUID},
			"transaction ID cannot be nil"},
		{"invalid currency", Session{ID: "test",
			TransactionID: validUUID, UserID: validUUID,
			Amount:    100,
			AccountID: validUUID, Currency: "XXX"},
			"unsupported currency"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSession_FormatAmount(t *testing.T) {
	// Setup test cases with proper expectations
	tests := []struct {
		name     string
		currency string
		amount   int64
		expected string
	}{
		{
			name:     "USD with 2 decimal places",
			currency: "USD",
			amount:   1000, // $10.00 (1000 / 100)
			expected: "10.00 USD",
		},
		{
			name:     "JPY with 0 decimal places",
			currency: "JPY",
			amount:   1000, // 1000 JPY
			expected: "1000 JPY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new registry for each test
			reg, err := currency.New(context.Background())
			require.NoError(t, err)

			// Register test currency
			err = reg.Register(currency.Meta{
				Code:     tt.currency,
				Name:     tt.currency + " Currency",
				Symbol:   tt.currency,
				Decimals: map[string]int{"USD": 2, "JPY": 0}[tt.currency],
			})
			require.NoError(t, err)

			s := Session{
				ID:       "test-session",
				Currency: tt.currency,
				Amount:   tt.amount,
			}

			// Test FormatAmount
			result, err := s.FormatAmount()
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestService_UpdateStatus(t *testing.T) {
	mr := mocks.NewRegistryProvider(t)
	mr.On("Get", mock.Anything, "test-session").Return(&registry.BaseEntity{
		BEId:     "test-session",
		BEName:   "checkout_session_123",
		BEActive: true,
		BEMetadata: map[string]string{
			"status": "created",
		},
	}, nil)
	mr.On("Register", mock.Anything, mock.Anything).Return(nil)

	svc := New(mr)
	err := svc.UpdateStatus(context.Background(), "test-session", "completed")

	require.NoError(t, err)
	mr.AssertExpectations(t)
}
