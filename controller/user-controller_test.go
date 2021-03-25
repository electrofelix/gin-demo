package controller_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/electrofelix/gin-demo/controller"
	"github.com/electrofelix/gin-demo/mocks"
)

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("simple", func(t *testing.T) {
		mockService := mocks.NewMockUserService(ctrl)
		c := controller.New(mockService)

		assert.IsType(t, &controller.UserController{}, c)
	})

	t.Run("with-logger", func(t *testing.T) {
		testLogger := logrus.Logger{}
		// use the following when need to inspect the logs in tests
		//hook := test.NewLocal(&testLogger)

		mockService := mocks.NewMockUserService(ctrl)
		c := controller.New(
			mockService,
			controller.WithLogger(&testLogger),
		)

		assert.IsType(t, &controller.UserController{}, c)
	})
}
