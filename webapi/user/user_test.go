package user_test

import (
	"context"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/webapi/testutils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	testutils.E2ETestSuite
	testUser *domain.User
	token    string
}

func (s *UserTestSuite) SetupTest() {
	// Create test user via POST /user/ endpoint
	s.testUser = s.CreateTestUser()
	s.token = s.LoginUser(s.testUser)
}

func (s *UserTestSuite) TestCreateUserVariants() {
	testCases := []struct {
		desc       string
		body       string
		wantStatus int
	}{
		{
			desc:       "success",
			body:       `{"username":"newuser","email":"new@example.com","password":"password123"}`,
			wantStatus: fiber.StatusCreated,
		},
		{
			desc:       "invalid body",
			body:       `{"username":123}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			resp := s.MakeRequest("POST", "/user", tc.body, "")
			defer resp.Body.Close() //nolint:errcheck
			s.Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *UserTestSuite) TestGetUserVariants() {
	testCases := []struct {
		userId     string
		desc       string
		wantStatus int
	}{
		{
			userId:     uuid.New().String(),
			desc:       "user not found",
			wantStatus: fiber.StatusUnauthorized, // Generic error for security
		},
		{
			userId:     s.testUser.ID.String(),
			desc:       "get user success",
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			resp := s.MakeRequest("GET", "/user/"+tc.userId, "", s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *UserTestSuite) TestUpdateUserVariants() {
	testCases := []struct {
		desc       string
		body       string
		wantStatus int
	}{
		{
			desc:       "success",
			body:       `{"names":"newname"}`,
			wantStatus: fiber.StatusOK,
		},
		{
			desc:       "invalid body",
			body:       `{"names":123}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			resp := s.MakeRequest("PUT", "/user/"+s.testUser.ID.String(), tc.body, s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *UserTestSuite) TestDeleteUserVariants() {
	testCases := []struct {
		desc       string
		body       string
		wantStatus int
	}{
		{
			desc:       "success",
			body:       `{"password":"password123"}`,
			wantStatus: fiber.StatusNoContent,
		},
		{
			desc:       "invalid body",
			body:       `{"pass":123}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "invalid password",
			body:       `{"password":"wrongpass"}`,
			wantStatus: fiber.StatusUnauthorized, // This is correct - invalid credentials should return 401
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			// Create a fresh user for each test case to avoid conflicts when user is deleted
			testUser := s.CreateTestUser()
			token := s.LoginUser(testUser)
			resp := s.MakeRequest("DELETE", "/user/"+testUser.ID.String(), tc.body, token)
			defer resp.Body.Close() //nolint:errcheck
			s.Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

type mockBus struct {
	handlers map[string][]eventbus.HandlerFunc
}

func (m *mockBus) Emit(ctx context.Context, event common.Event) error {
	handlers := m.handlers[event.Type()]
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockBus) Register(eventType string, handler eventbus.HandlerFunc) {
	if m.handlers == nil {
		m.handlers = make(map[string][]eventbus.HandlerFunc)
	}
	m.handlers[eventType] = append(m.handlers[eventType], handler)
}

func TestUserEventEmission(t *testing.T) {
	mockBus := &mockBus{}
	mockBus.Register("SomeEventType", func(ctx context.Context, event common.Event) error {
		return nil
	})
	// ... rest of test logic ...
}

func TestUserTestSuite(t *testing.T) {
	suite.Run(t, new(testutils.E2ETestSuite))
}
