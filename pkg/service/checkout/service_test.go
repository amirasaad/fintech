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
	mockEntity := &registry.BaseEntity{
		BEId:     "valid-session",
		BEName:   "checkout_session_123",
		BEActive: true,
		BEMetadata: map[string]string{
			"transaction_id": uuid.New().String(),
			"user_id":        uuid.New().String(),
			"account_id":     uuid.New().String(),
			"amount":         "1000",
			"currency":       "USD",
			"created_at":     time.Now().Format(time.RFC3339),
			"expires_at":     time.Now().Add(time.Hour).Format(time.RFC3339),
		},
	}

	tests := []struct {
		name      string
		sessionID string
		entity    registry.Entity
		err       error
		wantErr   bool
	}{
		{"valid session", "valid-session", mockEntity, nil, false},
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
				require.NotNil(t, session)
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
