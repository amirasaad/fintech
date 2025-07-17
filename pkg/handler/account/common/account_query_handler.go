package common

import (
	"context"
	"errors"

	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/queries"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

type getAccountQueryHandler struct {
	uow      repository.UnitOfWork
	eventBus eventbus.EventBus
}

func (h *getAccountQueryHandler) HandleQuery(ctx context.Context, query any) (any, error) {
	q, ok := query.(queries.GetAccountQuery)
	if !ok {
		return nil, errors.New("invalid query type")
	}
	accID, err := uuid.Parse(q.AccountID)
	if err != nil {
		return nil, err
	}
	repo, err := h.uow.AccountRepository()
	if err != nil {
		return nil, err
	}
	acc, err := repo.Get(accID)
	if err != nil || acc == nil {
		evt := events.AccountQueryFailedEvent{
			Query:  q,
			Reason: "account not found",
		}
		_ = h.eventBus.Publish(ctx, evt)
		return nil, errors.New("account not found")
	}
	result := queries.GetAccountResult{
		AccountID: acc.ID.String(),
		UserID:    acc.UserID.String(),
		Balance:   acc.Balance.AmountFloat(),
		Currency:  string(acc.Balance.Currency()),
	}
	evt := events.AccountQuerySucceededEvent{
		Result: result,
	}
	_ = h.eventBus.Publish(ctx, evt)
	return result, nil
}

func GetAccountQueryHandler(uow repository.UnitOfWork, eventBus eventbus.EventBus) QueryHandler {
	return &getAccountQueryHandler{uow: uow, eventBus: eventBus}
}
