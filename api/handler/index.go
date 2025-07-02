package handler

import (
	"net/http"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/webapi"
	"github.com/go-openapi/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/google/uuid"
)

// Handler is the main entry point of the application. Think of it like the main() method
func Handler(w http.ResponseWriter, r *http.Request) {
	// This is needed to set the proper request path in `*fiber.Ctx`
	r.RequestURI = r.URL.String()

	handler().ServeHTTP(w, r)
}

// building the fiber application
func handler() http.HandlerFunc {
	app := fiber.New()

	app.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{
			"uri":  ctx.Request().URI().String(),
			"path": ctx.Path(),
		})
	})

	webapi.AccountRoutes(app, func() (repository.UnitOfWork, error) {
		return NewMemoryUoW()
	})
	return adaptor.FiberApp(app)
}

type accountRepo struct {
	accounts map[uuid.UUID]*domain.Account
}
type transactionRepo struct {
	transactions map[uuid.UUID]*domain.Transaction
}

type userRepo struct {
	users map[uuid.UUID]*domain.User
}

// Valid implements repository.UserRepository.
func (u *userRepo) Valid(id uuid.UUID, password string) bool {
	panic("unimplemented")
}

// Create implements repository.UserRepository.
func (u *userRepo) Create(user *domain.User) error {
	panic("unimplemented")
}

// Delete implements repository.UserRepository.
func (u *userRepo) Delete(id uuid.UUID) error {
	panic("unimplemented")
}

// Get implements repository.UserRepository.
func (u *userRepo) Get(id uuid.UUID) (*domain.User, error) {
	panic("unimplemented")
}

// GetByEmail implements repository.UserRepository.
func (u *userRepo) GetByEmail(email string) (*domain.User, error) {
	panic("unimplemented")
}

// GetByUsername implements repository.UserRepository.
func (u *userRepo) GetByUsername(username string) (*domain.User, error) {
	panic("unimplemented")
}

// Update implements repository.UserRepository.
func (u *userRepo) Update(user *domain.User) error {
	panic("unimplemented")
}

// Create implements repository.TransactionRepository.
func (t *transactionRepo) Create(transaction *domain.Transaction) error {
	t.transactions[transaction.ID] = transaction
	return nil
}

// Get implements repository.TransactionRepository.
func (t *transactionRepo) Get(id uuid.UUID) (*domain.Transaction, error) {
	transaction, ok := t.transactions[id]
	if !ok {
		return nil, errors.NotFound("transaction not found")
	}
	return transaction, nil
}

// List implements repository.TransactionRepository.
func (t *transactionRepo) List(accountID uuid.UUID) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction
	for _, tx := range t.transactions {
		if tx.AccountID == accountID {
			transactions = append(transactions, tx)
		}
	}
	return transactions, nil
}

// Create implements repository.AccountRepository.
func (a *accountRepo) Create(account *domain.Account) error {
	a.accounts[account.ID] = account
	return nil
}

// Delete implements repository.AccountRepository.
func (a *accountRepo) Delete(id uuid.UUID) error {
	delete(a.accounts, id)
	return nil
}

// Get implements repository.AccountRepository.
func (a *accountRepo) Get(id uuid.UUID) (*domain.Account, error) {
	account, ok := a.accounts[id]
	if !ok {
		return nil, errors.NotFound("account not found")
	}
	return account, nil
}

// Update implements repository.AccountRepository.
func (a *accountRepo) Update(account *domain.Account) error {
	if _, ok := a.accounts[account.ID]; !ok {
		return errors.NotFound("account not found")
	}
	a.accounts[account.ID] = account
	return nil
}

type memoryUoW struct {
}

// Begin implements repository.UnitOfWork.
func (m *memoryUoW) Begin() error {
	return nil
}

// Commit implements repository.UnitOfWork.
func (m *memoryUoW) Commit() error {
	return nil
}

// Rollback implements repository.UnitOfWork.
func (m *memoryUoW) Rollback() error {
	return nil
}

// UserRepository implements repository.UnitOfWork.
func (m *memoryUoW) UserRepository() repository.UserRepository {
	return &userRepo{
		users: make(map[uuid.UUID]*domain.User),
	}
}

// AccountRepository implements repository.UnitOfWork.
func (m *memoryUoW) AccountRepository() repository.AccountRepository {
	return &accountRepo{
		accounts: make(map[uuid.UUID]*domain.Account),
	}
}

// TransactionRepository implements repository.UnitOfWork.
func (m *memoryUoW) TransactionRepository() repository.TransactionRepository {
	return &transactionRepo{
		transactions: make(map[uuid.UUID]*domain.Transaction),
	}
}

// NewMemoryUoW creates a new in-memory unit of work for testing purposes.
func NewMemoryUoW() (repository.UnitOfWork, error) {
	return &memoryUoW{}, nil
}
