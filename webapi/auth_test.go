package webapi

import (
	"bytes"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginRoute_BadRequest(t *testing.T) {
	app, _, _, _, _, _ := SetupTestApp(t)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":123}`))) // Invalid JSON
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestLoginRoute_Unauthorized(t *testing.T) {
	app, userRepo, _, _, mockUow, _ := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername(mock.Anything).Return(nil, nil).Once() // User not found
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":"nonexistent","password":"password"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestLoginRoute_InvalidPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	testUser.Password = string(hash)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()

	body := bytes.NewBuffer([]byte(`{"identity":"testuser","password":"wrongpassword"}`)) // Invalid password
	req := httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestLoginRoute_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	testUser.Password = string(hash)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":"testuser","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestLoginRoute_InternalServerError(t *testing.T) {
	app, userRepo, _, _, mockUow, _ := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername(mock.Anything).Return(nil, errors.New("db error")).Once() // Simulate DB error
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":"testuser","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestLoginRoute_ServiceError(t *testing.T) {
	app, _, _, _, mockUow, _ := SetupTestApp(t)
	mockUow.On("UserRepository").Return(nil).Once() // Simulate UoW error
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"identity":"testuser","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
