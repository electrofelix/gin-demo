package service_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/electrofelix/gin-demo/entity"
	"github.com/electrofelix/gin-demo/mocks"
	"github.com/electrofelix/gin-demo/service"
)

func TestUserService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		user := entity.User{
			Email: "user1@test.com",
			Name:  "test-user",
		}

		mockStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&user, nil)
		mockStore.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

		got, err := svc.Delete(context.Background(), user.Email)
		require.NoError(t, err)

		assert.Equal(t, user, got)
	})

	t.Run("not-found", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		mockStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, entity.ErrNotFound)

		got, err := svc.Delete(context.Background(), "any-email@test.com")
		if assert.ErrorIs(t, err, entity.ErrNotFound) {
			assert.Equal(t, entity.User{}, got)
		}
	})
}

func TestUserService_Get(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		user := entity.User{
			Email: "user1@example.com",
			Name:  "test-user",
		}

		mockStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&user, nil)

		got, err := svc.Get(context.Background(), user.Email)
		require.NoError(t, err)

		assert.Equal(t, user, got)
	})

	t.Run("bad-id", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		_, err := svc.Get(context.Background(), "")
		assert.ErrorIs(t, err, entity.ErrIDMissing)
	})

	t.Run("not-found", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		mockStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, entity.ErrNotFound)

		user, err := svc.Get(context.Background(), xid.New().String())
		require.Error(t, err)

		assert.ErrorIs(t, err, entity.ErrNotFound)
		assert.Equal(t, entity.User{}, user)
	})
}

func TestUserService_List(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		users := []entity.User{
			{
				Email: "user1@test.com",
				Name:  "test-user1",
			},
			{
				Email: "user2@test.com",
				Name:  "test-user2",
			},
		}

		mockStore.EXPECT().List(gomock.Any()).Return(users, nil)

		got, err := svc.List(context.Background())
		require.NoError(t, err)

		assert.ElementsMatch(t, users, got)
	})
}

func TestUserService_Put(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		user := entity.User{
			Email: "user1@test.com",
			Name:  "test-user1",
		}

		mockStore.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)

		got, err := svc.Put(context.Background(), user)
		require.NoError(t, err)

		assert.Equal(t, user, got)
	})

	t.Run("bad-id", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		_, err := svc.Put(context.Background(), entity.User{})
		
		assert.ErrorIs(t, err, entity.ErrIDMissing)
	})
}
