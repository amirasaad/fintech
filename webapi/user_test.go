package webapi

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestUpdateUserSuccess(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Times(2)
	userRepo.EXPECT().Get(testUser.ID).Return(testUser, nil).Once()
	userRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	mockUow.EXPECT().Begin().Return(nil).Once()
	mockUow.EXPECT().Commit().Return(nil).Once()

	body := bytes.NewBuffer([]byte(`{"names":"newname"}`))
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUpdateUserInvalidBody(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	body := bytes.NewBuffer([]byte(`{"names":123}`)) // Invalid body
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateUserNotFound(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Times(2)
	userRepo.EXPECT().Get(testUser.ID).Return(nil, errors.New("not found")).Once()

	body := bytes.NewBuffer([]byte(`{"names":"newname"}`))
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUpdateUserInternalError(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Times(2)
	userRepo.EXPECT().Get(testUser.ID).Return(testUser, nil).Once()
	userRepo.EXPECT().Update(mock.Anything).Return(errors.New("internal error")).Once()
	mockUow.EXPECT().Begin().Return(nil).Once()
	mockUow.EXPECT().Rollback().Return(nil).Once()

	body := bytes.NewBuffer([]byte(`{"names":"newname"}`))
	req := httptest.NewRequest("PUT", fmt.Sprintf("/user/%s", testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
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

func TestDeleteUserSuccess(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Times(2)
	userRepo.EXPECT().Valid(testUser.ID, "password123").Return(true).Once()
	userRepo.EXPECT().Delete(testUser.ID).Return(nil).Once()
	mockUow.EXPECT().Begin().Return(nil).Once()
	mockUow.EXPECT().Commit().Return(nil).Once()

	body := bytes.NewBuffer([]byte(`{"password":"password123"}`))
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestDeleteUserInvalidBody(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Times(2)
	userRepo.EXPECT().Valid(mock.Anything, "").Return(false).Once()

	body := bytes.NewBuffer([]byte(`{"pass":123}`)) // Invalid body
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestDeleteUserInvalidPassword(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Times(2)
	userRepo.EXPECT().Valid(testUser.ID, "wrongpass").Return(false).Once()

	body := bytes.NewBuffer([]byte(`{"password":"wrongpass"}`))
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestDeleteUserInternalError(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Times(2)
	userRepo.EXPECT().Valid(testUser.ID, "password123").Return(true).Once()
	userRepo.EXPECT().Delete(testUser.ID).Return(errors.New("internal error")).Once()
	mockUow.EXPECT().Begin().Return(nil).Once()
	mockUow.EXPECT().Rollback().Return(nil).Once()

	body := bytes.NewBuffer([]byte(`{"password":"password123"}`))
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%s", testUser.ID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
