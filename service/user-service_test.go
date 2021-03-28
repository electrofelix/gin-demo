package service_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/electrofelix/gin-demo/entity"
	"github.com/electrofelix/gin-demo/mocks"
	"github.com/electrofelix/gin-demo/service"
)

func TestUserService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		user := entity.User{
			Id:    xid.New().String(),
			Email: "user1@test.com",
			Name:  "test-user",
		}

		mockStore.EXPECT().Create(gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, newUser *entity.User) {
				assert.NotEqual(t, user.Password, newUser.Password)
			},
		).Return(nil)

		_, err := svc.Create(context.Background(), user)
		assert.NoError(t, err)
	})
}

func TestUserService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		user := entity.User{
			Id:    xid.New().String(),
			Email: "user1@test.com",
			Name:  "test-user",
		}

		mockStore.EXPECT().GetById(gomock.Any(), gomock.Any()).Return(&user, nil)
		mockStore.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

		got, err := svc.Delete(context.Background(), user.Id)
		require.NoError(t, err)

		assert.Equal(t, user, got)
	})

	t.Run("not-found", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		mockStore.EXPECT().GetById(gomock.Any(), gomock.Any()).Return(nil, entity.ErrNotFound)

		got, err := svc.Delete(context.Background(), xid.New().String())
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
			Id:    xid.New().String(),
			Email: "user1@example.com",
			Name:  "test-user",
		}

		mockStore.EXPECT().GetById(gomock.Any(), gomock.Any()).Return(&user, nil)

		got, err := svc.Get(context.Background(), user.Id)
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

		mockStore.EXPECT().GetById(gomock.Any(), gomock.Any()).Return(nil, entity.ErrNotFound)

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
				Id:    xid.New().String(),
				Email: "user1@test.com",
				Name:  "test-user1",
			},
			{
				Id:    xid.New().String(),
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
			Id:    xid.New().String(),
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

func TestUserService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		user := entity.User{
			Id:       xid.New().String(),
			Email:    "user1@test.com",
			Name:     "test-user1",
			Password: "some-password",
		}

		userUpdate := entity.User{
			Id: xid.New().String(),
		}

		mockStore.EXPECT().GetById(gomock.Any(), gomock.Any()).Return(&user, nil)
		mockStore.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)

		got, err := svc.Update(context.Background(), user.Id, userUpdate)
		require.NoError(t, err)

		assert.Equal(t, user.Email, got.Email)
		assert.Equal(t, user.Name, got.Name)
		assert.Equal(t, "", got.Password)
		assert.NotEqual(t, user.Password, "", "should only blank returned user password")
	})

	t.Run("update-name", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		user := entity.User{
			Id:    xid.New().String(),
			Email: "user1@test.com",
			Name:  "test-user1",
		}

		userUpdate := entity.User{
			Email: "user1@test.com",
			Name:  "test-user2",
		}

		mockStore.EXPECT().GetById(gomock.Any(), gomock.Any()).Return(&user, nil)
		mockStore.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)

		got, err := svc.Update(context.Background(), user.Id, userUpdate)
		require.NoError(t, err)

		assert.Equal(t, user.Email, got.Email)
		assert.NotEqual(t, user.Name, got.Name)
		assert.Equal(t, userUpdate.Name, got.Name)
	})

	t.Run("update-password", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		user := entity.User{
			Id:       xid.New().String(),
			Email:    "user1@test.com",
			Name:     "test-user1",
			Password: "some-password",
		}

		userUpdate := entity.User{
			Password: "some-password",
		}

		mockStore.EXPECT().GetById(gomock.Any(), gomock.Any()).Return(&user, nil)
		mockStore.EXPECT().Put(gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, user *entity.User) {
				assert.NotEqual(t, "some-password", user.Password)

				err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("some-password"))
				assert.NoError(t, err)
			},
		).Return(nil)

		got, err := svc.Update(context.Background(), user.Id, userUpdate)
		require.NoError(t, err)

		assert.Equal(t, user.Email, got.Email)
		assert.Equal(t, user.Name, got.Name)
	})

	t.Run("no-id", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		_, err := svc.Update(context.Background(), "", entity.User{})

		assert.ErrorIs(t, err, entity.ErrIDMissing)
	})

	t.Run("bad-id", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)

		userUpdate := entity.User{
			Email: "user2@test.com",
		}

		_, err := svc.Update(context.Background(), "a-bad-id", userUpdate)

		assert.ErrorIs(t, err, entity.ErrIDInvalid)
	})
}

func setupUserLoginResponses(
	t *testing.T, mock *mocks.MockUserStore, svc *service.UserService,
) (entity.User, entity.UserLogin) {
	t.Helper()

	user := entity.User{
		Email:    "user1@test.com",
		Name:     "test-user1",
		Password: "a-test-password",
	}

	userLogin := entity.UserLogin{
		Email:    user.Email,
		Password: "a-test-password",
	}

	// capture the encrypted password using create to help
	mock.EXPECT().Create(gomock.Any(), gomock.Any()).Do(
		func(ctx context.Context, newUser *entity.User) {
			user.Password = newUser.Password
		},
	).Return(nil)

	_, err := svc.Create(context.Background(), user)
	require.NoError(t, err)

	return user, userLogin
}

func TestUserService_ValidateCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("valid-credentials", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)
		user, userLogin := setupUserLoginResponses(t, mockStore, svc)

		mockStore.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).Return(&user, nil)
		mockStore.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)

		err := svc.ValidateCredentials(context.Background(), userLogin)
		assert.NoError(t, err)
	})

	t.Run("password-mismatch", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)
		user, userLogin := setupUserLoginResponses(t, mockStore, svc)

		mockStore.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).Return(&user, nil)

		userLogin.Password = "the-wrong-password"

		err := svc.ValidateCredentials(context.Background(), userLogin)
		assert.ErrorIs(t, err, entity.ErrBadCredentials)
	})

	t.Run("not-found", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)
		_, userLogin := setupUserLoginResponses(t, mockStore, svc)

		mockStore.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).Return(nil, entity.ErrNotFound)

		err := svc.ValidateCredentials(context.Background(), userLogin)
		assert.ErrorIs(t, err, entity.ErrBadCredentials)
	})

	t.Run("missing-id", func(t *testing.T) {
		mockStore := mocks.NewMockUserStore(ctrl)
		svc := service.New(mockStore)
		_, userLogin := setupUserLoginResponses(t, mockStore, svc)

		mockStore.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).Return(nil, entity.ErrIDMissing)

		err := svc.ValidateCredentials(context.Background(), userLogin)
		assert.ErrorIs(t, err, entity.ErrInternalError)
	})
}
