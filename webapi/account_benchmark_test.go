package webapi

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
)

// NOTE: This file assumes AccountTestSuite and its helpers are defined in account_test.go

func (s *AccountTestSuite) BenchmarkAccountDeposit(b *testing.B) {
	s.BeforeTest("", "BenchmarkAccountDeposit")
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo, nil).Maybe()
	s.mockUow.EXPECT().TransactionRepository().Return(s.transRepo, nil).Maybe()
	testAccount := account.NewAccount(s.testUser.ID)
	s.accountRepo.EXPECT().Get(mock.Anything).Return(testAccount, nil).Maybe()
	s.transRepo.EXPECT().Create(mock.Anything).Return(nil).Maybe()
	s.accountRepo.EXPECT().Update(mock.Anything).Return(nil).Maybe()
	s.mockUow.EXPECT().Begin().Return(nil).Maybe()
	s.mockUow.EXPECT().Commit().Return(nil).Maybe()

	depositBody := []byte(`{"amount": 100.0}`)
	url := fmt.Sprintf("/account/%s/deposit", testAccount.ID)

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest("POST", url, bytes.NewBuffer(depositBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.testToken)
		resp, err := s.app.Test(req, 10000)
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
		resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != fiber.StatusOK {
			b.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	}
}

func (s *AccountTestSuite) BenchmarkAccountWithdraw(b *testing.B) {
	s.BeforeTest("BenchmarkAccountWithdraw", "BenchmarkAccountWithdraw")
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo, nil).Maybe()
	s.mockUow.EXPECT().TransactionRepository().Return(s.transRepo, nil).Maybe()
	testAccount := account.NewAccount(s.testUser.ID)
	money, _ := money.NewMoney(1000.0, currency.Code("USD"))
	_, _ = testAccount.Deposit(s.testUser.ID, money)
	s.accountRepo.EXPECT().Get(mock.Anything).Return(testAccount, nil).Maybe()
	s.transRepo.EXPECT().Create(mock.Anything).Return(nil).Maybe()
	s.accountRepo.EXPECT().Update(mock.Anything).Return(nil).Maybe()
	s.mockUow.EXPECT().Begin().Return(nil).Maybe()
	s.mockUow.EXPECT().Commit().Return(nil).Maybe()

	withdrawBody := []byte(`{"amount": 100.0}`)
	url := fmt.Sprintf("/account/%s/withdraw", testAccount.ID)

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest("POST", url, bytes.NewBuffer(withdrawBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.testToken)
		resp, err := s.app.Test(req, 10000)
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
		resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != fiber.StatusOK {
			b.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	}
}
