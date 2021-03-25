package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/electrofelix/gin-demo/mocks"
	"github.com/electrofelix/gin-demo/server"
)

func TestServer_Start(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("starts", func(t *testing.T) {
		logger := logrus.New()
		hook := test.NewLocal(logger)
		controller := mocks.NewMockController(ctrl)

		// looking to capture logs, and start on any free port
		s := server.New(controller, server.WithLogger(logger), server.WithAddress(":0"))

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		t.Cleanup(cancel)

		err := s.Start(ctx)
		require.NoError(t, err)

		assert.Equal(t, "Shutdown complete", hook.LastEntry().Message)
	})
}
