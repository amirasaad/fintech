package checkout

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/google/uuid"
)

// Session represents a checkout session with its metadata
type Session struct {
	ID            string         `json:"id"`
	TransactionID uuid.UUID      `json:"transaction_id"`
	UserID        uuid.UUID      `json:"user_id"`
	AccountID     uuid.UUID      `json:"account_id"`
	Amount        int64          `json:"amount"`
	Currency      string         `json:"currency"`
	Status        string         `json:"status"`
	CheckoutURL   string         `json:"checkout_url"`
	CreatedAt     time.Time      `json:"created_at"`
	ExpiresAt     time.Time      `json:"expires_at"`
	currencyInfo  *currency.Meta // Cached currency info
}

// Service provides high-level operations for managing checkout sessions
type Service struct {
	registry registry.Provider
}

// NewService creates a new checkout service with the given registry
func NewService(reg registry.Provider) *Service {
	return &Service{
		registry: reg,
	}
}

// CreateSession creates a new checkout session
func (s *Service) CreateSession(
	ctx context.Context,
	sessionID string,
	id string,
	txID uuid.UUID,
	userID uuid.UUID,
	accountID uuid.UUID,
	amount int64,
	currencyCode string,
	checkoutURL string,
	expiresIn time.Duration,
) (*Session, error) {
	// Create the session
	session := &Session{
		ID:            sessionID,
		TransactionID: txID,
		UserID:        userID,
		AccountID:     accountID,
		Amount:        amount,
		Currency:      currencyCode,
		Status:        "created",
		CheckoutURL:   checkoutURL,
		CreatedAt:     time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(expiresIn),
	}

	// Validate the session
	if err := session.Validate(); err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	// Save to registry
	if err := s.saveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a checkout session by ID
func (s *Service) GetSession(
	ctx context.Context,
	id string,
) (*Session, error) {
	entity, err := s.registry.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error getting session: %w", err)
	}
	if entity == nil {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Convert registry.Meta to our internal Meta type
	meta := registry.Meta{
		ID:       entity.ID(),
		Name:     entity.Name(),
		Active:   entity.Active(),
		Metadata: entity.Metadata(),
	}

	return s.metaToSession(meta)
}

// GetSessionByTransactionID retrieves a checkout session by transaction ID
func (s *Service) GetSessionByTransactionID(
	ctx context.Context,
	txID uuid.UUID,
) (*Session, error) {
	// Search for session by transaction ID in metadata
	entities, err := s.registry.ListByMetadata(
		ctx,
		"transaction_id",
		txID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("error searching for session: %w", err)
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("session with transaction ID %s not found", txID)
	}

	// Convert the first matching entity to our internal Meta type
	entity := entities[0]
	meta := registry.Meta{
		ID:       entity.ID(),
		Name:     entity.Name(),
		Active:   entity.Active(),
		Metadata: entity.Metadata(),
	}

	return s.metaToSession(meta)
}

// UpdateStatus updates the status of a checkout session
func (s *Service) UpdateStatus(
	ctx context.Context,
	id, status string,
) error {
	// Get the existing entity
	entity, err := s.registry.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}
	if entity == nil {
		return fmt.Errorf("session not found: %s", id)
	}

	// Update the status in metadata
	metadata := entity.Metadata()
	metadata["status"] = status

	// Update the active status based on the new status
	active := status != "expired" && status != "canceled" && status != "failed"

	// Create a new entity with updated fields
	updatedEntity := &registry.BaseEntity{
		BEId:       entity.ID(),
		BEName:     entity.Name(),
		BEActive:   active,
		BEMetadata: metadata,
	}

	// Save the updated entity
	err = s.registry.Register(ctx, updatedEntity)
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	return nil
}

// Validate checks if the session is valid
func (s *Session) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if s.TransactionID == uuid.Nil {
		return fmt.Errorf("transaction ID cannot be nil")
	}

	if s.UserID == uuid.Nil {
		return fmt.Errorf("user ID cannot be nil")
	}

	if s.AccountID == uuid.Nil {
		return fmt.Errorf("account ID cannot be nil")
	}

	if s.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Validate currency
	if !currency.IsSupported(s.Currency) {
		return fmt.Errorf("unsupported currency: %s", s.Currency)
	}

	// Get and cache currency info
	currencyInfo, err := currency.Get(s.Currency)
	if err != nil {
		return fmt.Errorf("failed to get currency info: %w", err)
	}
	s.currencyInfo = &currencyInfo

	return nil
}

// FormatAmount formats the amount according to the currency's decimal places
func (s *Session) FormatAmount() (string, error) {
	if s.currencyInfo == nil {
		// Try to get currency info if not cached
		currencyInfo, err := currency.Get(s.Currency)
		if err != nil {
			return "", fmt.Errorf("failed to get currency info: %w", err)
		}
		s.currencyInfo = &currencyInfo
	}

	// Convert to float for display
	amount := float64(s.Amount) / float64(s.currencyInfo.Decimals)
	return fmt.Sprintf("%.*f", s.currencyInfo.Decimals, amount), nil
}

// GetCurrencySymbol returns the currency symbol
func (s *Session) GetCurrencySymbol() (string, error) {
	if s.currencyInfo == nil {
		// Try to get currency info if not cached
		currencyInfo, err := currency.Get(s.Currency)
		if err != nil {
			return "", fmt.Errorf("failed to get currency info: %w", err)
		}
		s.currencyInfo = &currencyInfo
	}

	return s.currencyInfo.Symbol, nil
}

// saveSession saves the session to the registry
func (s *Service) saveSession(session *Session) error {
	// Create a base entity with the session data
	entity := &registry.BaseEntity{
		BEId:   session.ID,
		BEName: fmt.Sprintf("checkout_session_%s", session.TransactionID.String()),
		BEActive: session.Status != "expired" &&
			session.Status != "canceled" && session.Status != "failed",
		BEMetadata: make(map[string]string),
	}

	// Add all fields as metadata for searchability
	entity.BEMetadata["transaction_id"] = session.TransactionID.String()
	entity.BEMetadata["user_id"] = session.UserID.String()
	entity.BEMetadata["account_id"] = session.AccountID.String()
	entity.BEMetadata["amount"] = fmt.Sprintf("%d", session.Amount)
	entity.BEMetadata["currency"] = session.Currency
	entity.BEMetadata["status"] = session.Status
	entity.BEMetadata["checkout_url"] = session.CheckoutURL
	entity.BEMetadata["created_at"] = session.CreatedAt.Format(time.RFC3339)
	entity.BEMetadata["expires_at"] = session.ExpiresAt.Format(time.RFC3339)

	// Store in registry
	ctx := context.Background()
	err := s.registry.Register(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to register session: %w", err)
	}

	return nil
}

// metaToSession converts registry.Meta to Session
func (s *Service) metaToSession(meta registry.Meta) (*Session, error) {
	txID, err := uuid.Parse(meta.Metadata["transaction_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid transaction_id: %w", err)
	}

	userID, err := uuid.Parse(meta.Metadata["user_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	accountID, err := uuid.Parse(meta.Metadata["account_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	var amount int64
	_, err = fmt.Sscanf(meta.Metadata["amount"], "%d", &amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, meta.Metadata["created_at"])
	if err != nil {
		return nil, fmt.Errorf("invalid created_at: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, meta.Metadata["expires_at"])
	if err != nil {
		return nil, fmt.Errorf("invalid expires_at: %w", err)
	}

	session := &Session{
		ID:            meta.ID,
		TransactionID: txID,
		UserID:        userID,
		AccountID:     accountID,
		Amount:        amount,
		Currency:      meta.Metadata["currency"],
		Status:        meta.Metadata["status"],
		CheckoutURL:   meta.Metadata["checkout_url"],
		CreatedAt:     createdAt,
		ExpiresAt:     expiresAt,
	}

	// Validate the session
	if err := session.Validate(); err != nil {
		return nil, fmt.Errorf("invalid session data in registry: %w", err)
	}

	return session, nil
}

// ToJSON converts a Session to its JSON representation
func (s *Session) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON creates a Session from its JSON representation
func FromJSON(data []byte) (*Session, error) {
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Validate the session after unmarshaling
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("invalid session data: %w", err)
	}

	return &s, nil
}
