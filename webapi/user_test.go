package webapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	E2ETestSuite
	app         *fiber.App
	userRepo    *fixtures.MockUserRepository
	mockUow     *fixtures.MockUnitOfWork
	testUser    *domain.User
	testToken   string
	authService *service.AuthService
}

func (s *UserTestSuite) SetupTest() {
	s.app, s.userRepo, _, _, s.mockUow, s.testUser, s.authService, _ = SetupTestApp(s.T())
	// Setup mock for login request
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo).Maybe()
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil).Maybe()
	s.testToken = getTestToken(s.T(), s.app, s.testUser)
}

func (s *UserTestSuite) TestCreateUser() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Create(mock.Anything).Return(nil)
	s.mockUow.EXPECT().Begin().Return(nil)
	s.mockUow.EXPECT().Commit().Return(nil)

	body := bytes.NewBuffer([]byte(`{"username":"testuser","email":"fixtures@example.com","password":"password123"}`))
	req := httptest.NewRequest("POST", "/user", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusCreated, resp.StatusCode)
}

func (s *UserTestSuite) TestCreateUserInvalidBody() {
	body := bytes.NewBuffer([]byte(`{"username":"","email":"not-an-email","password":"123"}`))
	req := httptest.NewRequest("POST", "/user", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *UserTestSuite) TestGetUserNotFound() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Get(s.testUser.ID).Return(s.testUser, nil)

	id := uuid.New()
	s.userRepo.EXPECT().Get(id).Return(&domain.User{}, domain.ErrUserNotFound)

	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%s", id), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusNotFound, resp.StatusCode)
}

func (s *UserTestSuite) TestGetUserSuccess() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Get(s.testUser.ID).Return(s.testUser, nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%s", s.testUser.ID), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)

	var response Response
	bodyBytes, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	err = json.Unmarshal(bodyBytes, &response)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().NotNil(response.Data)
}

func (s *UserTestSuite) TestUpdateUserUnauthorized() {
	id := uuid.New()
	body := bytes.NewBuffer([]byte(`{"names":"newname"}`))
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", id), body)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserTestSuite) TestUpdateUserSuccess() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Get(s.testUser.ID).Return(s.testUser, nil)
	s.userRepo.EXPECT().Update(mock.Anything).Return(nil)
	s.mockUow.EXPECT().Begin().Return(nil)
	s.mockUow.EXPECT().Commit().Return(nil)

	body := bytes.NewBuffer([]byte(`{"names":"newname"}`))
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", s.testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *UserTestSuite) TestUpdateUserInvalidBody() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Get(s.testUser.ID).Return(s.testUser, nil)

	body := bytes.NewBuffer([]byte(`{"names":123}`)) // Invalid body
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", s.testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *UserTestSuite) TestUpdateUserNotFound() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Get(s.testUser.ID).Return(nil, errors.New("not found"))

	body := bytes.NewBuffer([]byte(`{"names":"newname"}`))
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", s.testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusNotFound, resp.StatusCode)
}

func (s *UserTestSuite) TestUpdateUserInternalError() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Get(s.testUser.ID).Return(s.testUser, nil)
	s.userRepo.EXPECT().Update(mock.Anything).Return(errors.New("internal error"))
	s.mockUow.EXPECT().Begin().Return(nil)
	s.mockUow.EXPECT().Rollback().Return(nil)

	body := bytes.NewBuffer([]byte(`{"names":"newname"}`))
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", s.testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *UserTestSuite) TestDeleteUserUnauthorized() {
	id := uuid.New()
	body := bytes.NewBuffer([]byte(`{"password":"wrongpass"}`))
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", id), body)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserTestSuite) TestDeleteUserSuccess() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Valid(s.testUser.ID, "password123").Return(true)
	s.userRepo.EXPECT().Delete(s.testUser.ID).Return(nil)
	s.mockUow.EXPECT().Begin().Return(nil)
	s.mockUow.EXPECT().Commit().Return(nil)

	body := bytes.NewBuffer([]byte(`{"password":"password123"}`))
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", s.testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusNoContent, resp.StatusCode)
}

func (s *UserTestSuite) TestDeleteUserInvalidBody() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Valid(s.testUser.ID, "").Return(false)

	body := bytes.NewBuffer([]byte(`{"pass":123}`)) // Invalid body
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", s.testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserTestSuite) TestDeleteUserInvalidPassword() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Valid(s.testUser.ID, "wrongpass").Return(false)

	body := bytes.NewBuffer([]byte(`{"password":"wrongpass"}`))
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", s.testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserTestSuite) TestDeleteUserInternalError() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().Valid(s.testUser.ID, "password123").Return(true)
	s.userRepo.EXPECT().Delete(s.testUser.ID).Return(errors.New("internal error"))
	s.mockUow.EXPECT().Begin().Return(nil)
	s.mockUow.EXPECT().Rollback().Return(nil)

	body := bytes.NewBuffer([]byte(`{"password":"password123"}`))
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", s.testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func TestUserTestSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}
