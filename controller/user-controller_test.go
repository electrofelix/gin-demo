package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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

func setupMocks(t *testing.T) (*controller.UserController, *mocks.MockUserService, *test.Hook) {
	t.Helper()
	ctrl := gomock.NewController(t)

	testLogger := logrus.New()
	logHook := test.NewLocal(testLogger)
	mockService := mocks.NewMockUserService(ctrl)
	c := controller.New(mockService, controller.WithLogger(testLogger))

	return c, mockService, logHook
}

func TestNew(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockService := mocks.NewMockUserService(ctrl)
		c := controller.New(mockService)

		assert.IsType(t, &controller.UserController{}, c)
	})

	t.Run("with-logger", func(t *testing.T) {
		c, _, _ := setupMocks(t)

		assert.IsType(t, &controller.UserController{}, c)
	})
}

func setupEngine(t *testing.T, c *controller.UserController) (*gin.Engine, *httptest.ResponseRecorder) {
	t.Helper()

	engine := gin.Default()
	c.RegisterRoutes(engine)

	recorder := httptest.NewRecorder()

	return engine, recorder
}

func TestUserController_list(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, mockService, _ := setupMocks(t)
		engine, recorder := setupEngine(t, c)

		mockService.EXPECT().List(gomock.Any()).Return([]entity.User{}, nil)

		req, err := http.NewRequest("GET", "/users", nil)
		require.NoError(t, err)

		engine.ServeHTTP(recorder, req)

		assert.Equal(t, 200, recorder.Code)
		assert.Equal(t, "[]", recorder.Body.String())
	})

	t.Run("error", func(t *testing.T) {
		c, mockService, _ := setupMocks(t)
		engine, recorder := setupEngine(t, c)

		mockService.EXPECT().List(gomock.Any()).Return([]entity.User{}, errors.New("failed lookup"))

		req, err := http.NewRequest("GET", "/users", nil)
		require.NoError(t, err)

		engine.ServeHTTP(recorder, req)

		assert.Equal(t, 500, recorder.Code)
		assert.Equal(t, "{\"error\":\"Internal Error\"}", recorder.Body.String())
	})
}

func TestUserController_create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, mockService, _ := setupMocks(t)
		engine, recorder := setupEngine(t, c)

		newUser := entity.User{
			Name:     "test user",
			Email:    "test@example.com",
			Password: "simple-password",
		}

		jsonBody, err := json.Marshal(newUser)
		require.NoError(t, err)

		var modifiedUser entity.User

		mockService.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, user entity.User) (entity.User, error) {
				assert.NotEqual(t, newUser.Password, user.Password)

				// save for comparison
				modifiedUser = user

				return user, nil
			},
		)

		req, err := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)

		engine.ServeHTTP(recorder, req)

		modifiedUser.Password = ""

		jsonUser, err := json.Marshal(modifiedUser)
		require.NoError(t, err)

		assert.Equal(t, 201, recorder.Code)
		assert.Equal(t, string(jsonUser), recorder.Body.String())
	})
}
