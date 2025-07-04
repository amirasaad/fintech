package webapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	app, userRepo, _, _, mockUow, _ := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo)
	userRepo.On("Create", mock.Anything).Return(nil)
	mockUow.On("Begin").Return(nil)
	mockUow.On("Commit").Return(nil)
	body := bytes.NewBuffer([]byte(`{"username":"testuser","email":"fixtures@example.com","password":"password123"}`))
	req := httptest.NewRequest("POST", "/user", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestCreateUserInvalidBody(t *testing.T) {
	app, _, _, _, _, _ := SetupTestApp(t)
	body := bytes.NewBuffer([]byte(`{"username":"","email":"not-an-email","password":"123"}`))
	req := httptest.NewRequest("POST", "/user", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetUserNotFound(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo)

	id := uuid.New()
	userRepo.EXPECT().Get(id).Return(&domain.User{}, domain.ErrUserNotFound)
	// Add missing expectation for GetByUsername, which may be called during token validation
	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%s", id), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetUserSuccess(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo)
	userRepo.EXPECT().Get(testUser.ID).Return(testUser, nil)
	token := getTestToken(t, app, userRepo, mockUow, testUser)
	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%s", testUser.ID), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	var response Response
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	fmt.Printf("Response body: %s\n", string(bodyBytes))
	err = json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)
	assert.NotNil(t, response.Data)
}

func TestUpdateUserUnauthorized(t *testing.T) {
	app, _, _, _, _, _ := SetupTestApp(t)
	id := uuid.New()
	body := bytes.NewBuffer([]byte(`{"names":"newname"}`))
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", id), body)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestDeleteUserUnauthorized(t *testing.T) {
	app, _, _, _, _, _ := SetupTestApp(t)
	id := uuid.New()
	body := bytes.NewBuffer([]byte(`{"password":"wrongpass"}`))
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", id), body)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}
