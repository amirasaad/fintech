package commands

import (
	"github.com/google/uuid"
)

type Transfer struct {
	UserID        uuid.UUID
	AccountID     uuid.UUID
	Amount        float64
	Currency      string
	FromAccountID uuid.UUID
	ToAccountID   uuid.UUID
}
