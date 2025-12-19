package checkout

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/google/uuid"
)

// Session represents a checkout session with its metadata
type Session struct {
	ID            string    `json:"id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	UserID        uuid.UUID `json:"user_id"`
	AccountID     uuid.UUID `json:"account_id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CheckoutURL   string    `json:"checkout_url"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// Service provides high-level operations for managing checkout sessions
type Service struct {
	registry registry.Provider
	logger   *slog.Logger
}

// New creates a new checkout service with the given registry and logger
func New(reg registry.Provider, logger *slog.Logger) *Service {
	return &Service{
		registry: reg,
		logger:   logger,
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

	// Convert Entity to Session
	return s.entityToSession(entity)
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

	// Convert the first matching entity to Session
	return s.entityToSession(entities[0])
}

// GetSessionsByUserID retrieves all checkout sessions for a given user ID
func (s *Service) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	entities, err := s.registry.ListByMetadata(ctx, "user_id", userID.String())
	if err != nil {
		return nil, fmt.Errorf("error getting sessions by user ID: %w", err)
	}

	var sessions []*Session
	for _, entity := range entities {
		session, err := s.entityToSession(entity)
		if err != nil {
			return nil, fmt.Errorf("error converting entity to session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
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

	if !money.Code(s.Currency).IsValid() {
		return fmt.Errorf("invalid currency code: %s", s.Currency)
	}
	return nil
}

// FormatAmount formats the amount according to the currency's decimal places
func (s *Session) FormatAmount() (string, error) {
	// Create a Money object from the amount and currency
	m, err := money.NewFromSmallestUnit(s.Amount, money.Code(s.Currency))
	if err != nil {
		return "", fmt.Errorf("failed to create money object: %w", err)
	}
	// Use Money's String() method which handles the formatting
	return m.String(), nil
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
	entity.SetMetadata("transaction_id", session.TransactionID.String())
	entity.SetMetadata("user_id", session.UserID.String())
	entity.SetMetadata("account_id", session.AccountID.String())
	entity.SetMetadata("amount", fmt.Sprintf("%d", session.Amount))
	entity.SetMetadata("currency", session.Currency)
	entity.SetMetadata("status", session.Status)
	entity.SetMetadata("checkout_url", session.CheckoutURL)
	entity.SetMetadata("created_at", session.CreatedAt.Format(time.RFC3339))
	entity.SetMetadata("expires_at", session.ExpiresAt.Format(time.RFC3339))

	// Store in registry
	ctx := context.Background()
	err := s.registry.Register(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to register session: %w", err)
	}

	return nil
}

// entityToSession converts a registry.Entity to a Session
func (s *Service) entityToSession(entity registry.Entity) (*Session, error) {
	if entity == nil {
		return nil, fmt.Errorf("entity cannot be nil")
	}

	metadata := entity.Metadata()
	// Debug: Log all metadata keys and values
	s.logger.Debug("Entity metadata", "metadata", metadata)

	session := &Session{
		ID:            entity.ID(),
		TransactionID: uuid.Nil,
		UserID:        uuid.Nil,
		AccountID:     uuid.Nil,
		Status:        "",
		CheckoutURL:   "",
		CreatedAt:     time.Time{},
		ExpiresAt:     time.Time{},
	}

	// Parse transaction ID
	if txID, ok := metadata["transaction_id"]; ok && txID != "" {
		id, err := uuid.Parse(txID)
		if err != nil {
			return nil, fmt.Errorf("invalid transaction ID in metadata: %w", err)
		}
		session.TransactionID = id
	}

	// Parse user ID
	if userID, ok := metadata["user_id"]; ok && userID != "" {
		id, err := uuid.Parse(userID)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID in metadata: %w", err)
		}
		session.UserID = id
	}

	// Parse account ID
	if accountID, ok := metadata["account_id"]; ok && accountID != "" {
		id, err := uuid.Parse(accountID)
		if err != nil {
			return nil, fmt.Errorf("invalid account ID in metadata: %w", err)
		}
		session.AccountID = id
	}

	// Parse amount
	if amount, ok := metadata["amount"]; ok && amount != "" {
		if amt, err := strconv.ParseInt(amount, 10, 64); err == nil {
			session.Amount = amt
		}
	}

	// Set other fields
	session.Currency = metadata["currency"]
	session.Status = metadata["status"]
	session.CheckoutURL = metadata["checkout_url"]

	// Parse timestamps
	if createdAt, ok := metadata["created_at"]; ok && createdAt != "" {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			session.CreatedAt = t
		}
	}

	if expiresAt, ok := metadata["expires_at"]; ok && expiresAt != "" {
		if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
			session.ExpiresAt = t
		}
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
