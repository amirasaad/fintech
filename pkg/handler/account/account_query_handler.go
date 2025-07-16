package account

import (
	"context"
	"errors"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

type GetAccountResult struct {
	Account *account.Account
}

type getAccountQueryHandler struct {
	uow      repository.UnitOfWork
	eventBus eventbus.EventBus
}

func (h *getAccountQueryHandler) HandleQuery(ctx context.Context, query any) (any, error) {
	q, ok := query.(account.GetAccountQuery)
	if !ok {
		return nil, errors.New("invalid query type")
	}
	accID, err := uuid.Parse(q.AccountID)
	if err != nil {
		// Emit query failed event
		_ = h.eventBus.Publish(ctx, account.AccountQueryFailedEvent{
			Query: q,
			Error: "invalid account ID format",
		})
		return nil, err
	}
	userID, err := uuid.Parse(q.UserID)
	if err != nil {
		// Emit query failed event
		_ = h.eventBus.Publish(ctx, account.AccountQueryFailedEvent{
			Query: q,
			Error: "invalid user ID format",
		})
		return nil, err
	}

	// For read-only queries, we don't need transactional guarantees
	repo, err := h.uow.AccountRepository()
	if err != nil {
		// Emit query failed event
		_ = h.eventBus.Publish(ctx, account.AccountQueryFailedEvent{
			Query: q,
			Error: "failed to get account repository",
		})
		return nil, err
	}

	acc, err := repo.Get(accID)
	if err != nil {
		// Emit query failed event
		_ = h.eventBus.Publish(ctx, account.AccountQueryFailedEvent{
			Query: q,
			Error: err.Error(),
		})
		return nil, err
	}
	if acc == nil || acc.UserID != userID {
		// Emit query failed event
		_ = h.eventBus.Publish(ctx, account.AccountQueryFailedEvent{
			Query: q,
			Error: "account not found or unauthorized",
		})
		return nil, errors.New("account not found or unauthorized")
	}

	// Query succeeded, emit success event
	_ = h.eventBus.Publish(ctx, account.AccountQuerySucceededEvent{
		Query:   q,
		Account: acc,
	})

	return GetAccountResult{Account: acc}, nil
}

func GetAccountQueryHandler(uow repository.UnitOfWork, eventBus eventbus.EventBus) QueryHandler {
	return &getAccountQueryHandler{uow: uow, eventBus: eventBus}
}
