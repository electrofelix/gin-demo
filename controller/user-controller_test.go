package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/electrofelix/gin-demo/controller"
	"github.com/electrofelix/gin-demo/entity"
	"github.com/electrofelix/gin-demo/mocks"
)

func setupMocks(t *testing.T) (*controller.UserController, *gin.Engine, *mocks.MockUserService, *test.Hook) {
	t.Helper()
	ctrl := gomock.NewController(t)

	testLogger := logrus.New()
	logHook := test.NewLocal(testLogger)
	mockService := mocks.NewMockUserService(ctrl)
	engine := gin.Default()
	c := controller.New(mockService, engine, controller.WithLogger(testLogger))

	return c, engine, mockService, logHook
}

func TestNew(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockService := mocks.NewMockUserService(ctrl)
		c := controller.New(mockService, gin.Default())

		assert.IsType(t, &controller.UserController{}, c)
	})

	t.Run("with-logger", func(t *testing.T) {
		c, _, _, _ := setupMocks(t)

		assert.IsType(t, &controller.UserController{}, c)
	})
}

func TestUserController_list(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, engine, mockService, _ := setupMocks(t)
		recorder := httptest.NewRecorder()

		mockService.EXPECT().List(gomock.Any()).Return([]entity.User{}, nil)

		req, err := http.NewRequest("GET", "/users", nil)
		require.NoError(t, err)

		engine.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.Equal(t, "[]", recorder.Body.String())
	})

	t.Run("multiple-users", func(t *testing.T) {
		_, engine, mockService, _ := setupMocks(t)
		recorder := httptest.NewRecorder()

		users := []entity.User{
			{
				Email:     "user1@test.com",
				Name:      "Test user 1",
				LastLogin: time.Now(),
			},
			{
				Email:     "user2@test.com",
				Name:      "Test user 2",
				LastLogin: time.Now(),
			},
		}

		jsonBody, err := json.Marshal(users)
		require.NoError(t, err)

		mockService.EXPECT().List(gomock.Any()).Return(users, nil)

		req, err := http.NewRequest("GET", "/users", nil)
		require.NoError(t, err)

		engine.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)

		assert.Equal(t, jsonBody, recorder.Body.Bytes())
	})

	t.Run("error", func(t *testing.T) {
		_, engine, mockService, _ := setupMocks(t)
		recorder := httptest.NewRecorder()

		mockService.EXPECT().List(gomock.Any()).Return([]entity.User{}, errors.New("failed lookup"))

		req, err := http.NewRequest("GET", "/users", nil)
		require.NoError(t, err)

		engine.ServeHTTP(recorder, req)

		assert.Equal(t, 500, recorder.Code)
		assert.Equal(t, "{\"error\":\"Internal Error\"}", recorder.Body.String())
	})
}

// setup a user entity and equvalent json for the request
func setupTestUser(t *testing.T) (entity.User, *bytes.Buffer) {
	newUser := entity.User{
		Name:     "test user",
		Email:    "test@example.com",
		Password: "simple-password",
	}

	jsonBody, err := json.Marshal(newUser)
	require.NoError(t, err)

	return newUser, bytes.NewBuffer(jsonBody)
}

func TestUserController_create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, engine, mockService, _ := setupMocks(t)
		recorder := httptest.NewRecorder()

		newUser, jsonBody := setupTestUser(t)

		var returnedUser entity.User

		mockService.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, user entity.User) (entity.User, error) {
				returnedUser = newUser

				returnedUser.Password = ""

				return returnedUser, nil
			},
		)

		req, err := http.NewRequest("POST", "/users", jsonBody)
		require.NoError(t, err)

		engine.ServeHTTP(recorder, req)

		jsonUser, err := json.Marshal(returnedUser)
		require.NoError(t, err)

		assert.Equal(t, 201, recorder.Code)
		assert.Equal(t, string(jsonUser), recorder.Body.String())
	})

	t.Run("duplicate", func(t *testing.T) {
		_, engine, mockService, _ := setupMocks(t)
		recorder := httptest.NewRecorder()

		mockService.EXPECT().Create(gomock.Any(), gomock.Any()).Return(
			entity.User{}, entity.ErrEmailDuplicate,
		)

		_, jsonBody := setupTestUser(t)
		req, err := http.NewRequest("POST", "/users", jsonBody)
		require.NoError(t, err)

		engine.ServeHTTP(recorder, req)

		assert.Equal(t, 409, recorder.Code)
		assert.Equal(t, fmt.Sprintf("{\"error\":\"%s\"}", entity.ErrEmailDuplicate.Error()), recorder.Body.String())
	})
}
