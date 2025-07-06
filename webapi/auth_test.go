package webapi

import (
	"bytes"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type AuthTestSuite struct {
	E2ETestSuite
	app      *fiber.App
	userRepo *fixtures.MockUserRepository
	mockUow  *fixtures.MockUnitOfWork
	testUser *domain.User
}

func (s *AuthTestSuite) SetupTest() {
	s.app, s.userRepo, _, _, s.mockUow, s.testUser = SetupTestApp(s.T())
}

func (s *AuthTestSuite) TestLoginRoute_BadRequest() {

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":123}`))) // Invalid JSON
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AuthTestSuite) TestLoginRoute_Unauthorized() {

	s.mockUow.EXPECT().UserRepository().Return(s.userRepo).Once()
	s.userRepo.EXPECT().GetByUsername(mock.Anything).Return(nil, nil).Once() // User not found
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":"nonexistent","password":"password"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AuthTestSuite) TestLoginRoute_InvalidPassword() {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	s.testUser.Password = string(hash)
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo).Once()
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil).Once()

	body := bytes.NewBuffer([]byte(`{"identity":"testuser","password":"wrongpassword"}`)) // Invalid password
	req := httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AuthTestSuite) TestLoginRoute_Success() {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	s.testUser.Password = string(hash)
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo).Once()
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil).Once()
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":"testuser","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *AuthTestSuite) TestLoginRoute_InternalServerError() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo).Once()
	s.userRepo.EXPECT().GetByUsername(mock.Anything).Return(nil, errors.New("db error")).Once() // Simulate DB error
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":"testuser","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *AuthTestSuite) TestLoginRoute_ServiceError() {
	s.mockUow.On("UserRepository").Return(nil).Once() // Simulate UoW error
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":"testuser","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
