package controller_test

import (
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
