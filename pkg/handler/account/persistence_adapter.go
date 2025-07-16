package account

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
)

type PersistenceAdapter struct {
	uow    repository.UnitOfWork
	logger *slog.Logger
}

func NewPersistenceAdapter(uow repository.UnitOfWork, logger *slog.Logger) *PersistenceAdapter {
	return &PersistenceAdapter{uow: uow, logger: logger}
}

// Implement persistence methods as needed for event-driven deposit chain.
