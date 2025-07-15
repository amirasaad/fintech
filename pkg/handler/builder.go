package handler

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/repository"
)

// ChainBuilder builds operation-specific chains
type ChainBuilder struct {
	uow       repository.UnitOfWork
	converter money.CurrencyConverter
	logger    *slog.Logger
	provider  provider.PaymentProvider // Add provider field
}

// NewChainBuilder creates a new chain builder
func NewChainBuilder(uow repository.UnitOfWork, converter money.CurrencyConverter, provider provider.PaymentProvider, logger *slog.Logger) *ChainBuilder {
	return &ChainBuilder{
		uow:       uow,
		converter: converter,
		provider:  provider,
		logger:    logger,
	}
}

// BuildDepositChain builds a chain for deposit operations
func (b *ChainBuilder) BuildDepositChain() OperationHandler {
	validation := &ValidationHandler{
		uow:    b.uow,
		logger: b.logger,
	}
	moneyCreation := &MoneyCreationHandler{
		logger: b.logger,
	}
	currencyConversion := &CurrencyConversionHandler{
		converter: b.converter,
		logger:    b.logger,
	}
	paymentProvider := &PaymentProviderHandler{
		logger:   b.logger,
		provider: b.provider, // Use injected provider
	}
	domainOperation := &DepositOperationHandler{
		logger: b.logger,
	}
	persistence := &DepositPersistenceHandler{
		uow:    b.uow,
		logger: b.logger,
	}

	// Chain them together
	validation.SetNext(moneyCreation)
	moneyCreation.SetNext(currencyConversion)
	currencyConversion.SetNext(paymentProvider)
	paymentProvider.SetNext(domainOperation)
	domainOperation.SetNext(persistence)

	return validation
}

// BuildWithdrawChain builds a chain for withdraw operations
func (b *ChainBuilder) BuildWithdrawChain() OperationHandler {
	validation := &ValidationHandler{
		uow:    b.uow,
		logger: b.logger,
	}
	moneyCreation := &MoneyCreationHandler{
		logger: b.logger,
	}
	currencyConversion := &CurrencyConversionHandler{
		converter: b.converter,
		logger:    b.logger,
	}
	domainOperation := &WithdrawOperationHandler{
		logger: b.logger,
	}
	persistence := &WithdrawPersistenceHandler{
		uow:    b.uow,
		logger: b.logger,
	}

	// Chain them together
	validation.SetNext(moneyCreation)
	moneyCreation.SetNext(currencyConversion)
	currencyConversion.SetNext(domainOperation)
	domainOperation.SetNext(persistence)

	return validation
}

// BuildTransferChain builds a chain for transfer operations
func (b *ChainBuilder) BuildTransferChain() OperationHandler {
	validation := &TransferValidationHandler{
		uow:    b.uow,
		logger: b.logger,
	}
	moneyCreation := &MoneyCreationHandler{
		logger: b.logger,
	}
	currencyConversion := &CurrencyConversionHandler{
		converter: b.converter,
		logger:    b.logger,
	}
	domainOperation := &TransferOperationHandler{
		logger: b.logger,
	}
	persistence := &TransferPersistenceHandler{
		uow:    b.uow,
		logger: b.logger,
	}

	// Chain them together
	validation.SetNext(moneyCreation)
	moneyCreation.SetNext(currencyConversion)
	currencyConversion.SetNext(domainOperation)
	domainOperation.SetNext(persistence)

	return validation
}
