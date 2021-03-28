package store_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/electrofelix/gin-demo/entity"
	"github.com/electrofelix/gin-demo/mocks"
	"github.com/electrofelix/gin-demo/store"
)

const (
	tableName = "test-table"
)

// Would be more useful tests for success paths to be able to execute the
// tests against an automatically spun up DB instance and confirm the general
// behaviour matches.

func userToUserAttributeValue(user entity.User) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"Id":         &types.AttributeValueMemberS{Value: user.Id},
		"Email":      &types.AttributeValueMemberS{Value: user.Email},
		"Name":       &types.AttributeValueMemberS{Value: user.Name},
		"Password":   &types.AttributeValueMemberS{Value: user.Password},
		"objectType": &types.AttributeValueMemberS{Value: "UserInfo"},
	}
}

func userToEmailAttributeValue(user entity.User) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"Id":         &types.AttributeValueMemberS{Value: user.Email},
		"UserId":     &types.AttributeValueMemberS{Value: user.Id},
		"objectType": &types.AttributeValueMemberS{Value: "UserInfo#email"},
	}
}

func TestUserStore_Create(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		mockDBClient.EXPECT().TransactWriteItems(gomock.Any(), gomock.Any()).Return(nil, nil)

		err := dataStore.Create(context.Background(), &entity.User{})
		assert.NoError(t, err)
	})

	t.Run("duplicate email", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		mockDBClient.EXPECT().TransactWriteItems(gomock.Any(), gomock.Any()).Return(
			nil, &types.ConditionalCheckFailedException{
				Message: aws.String("simulated conditional check failed"),
			},
		)

		err := dataStore.Create(context.Background(), &entity.User{})
		assert.Equal(t, err, entity.ErrIDCollision)
	})
}

func TestUserStore_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		user := entity.User{
			Email: "user1@example.com",
			Name:  "test-user",
		}

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{Item: userToUserAttributeValue(user)}, nil,
		)
		mockDBClient.EXPECT().TransactWriteItems(gomock.Any(), gomock.Any()).Return(&dynamodb.TransactWriteItemsOutput{}, nil)

		err := dataStore.Delete(context.Background(), user.Email)
		require.NoError(t, err)
	})

	t.Run("not-found", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		_ = store.NewUserStore(mockDBClient, tableName)
	})
}

func TestUserStore_GetByEmail(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		user := entity.User{
			Id:    xid.New().String(),
			Email: "user1@example.com",
			Name:  "test-user",
		}

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{Item: userToEmailAttributeValue(user)}, nil,
		)
		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{Item: userToUserAttributeValue(user)}, nil,
		)

		got, err := dataStore.GetByEmail(context.Background(), user.Email)
		require.NoError(t, err)

		assert.Equal(t, user, *got)
	})
}

func TestUserStore_GetById(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		user := entity.User{
			Email: "user1@example.com",
			Name:  "test-user",
		}

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{Item: userToUserAttributeValue(user)}, nil,
		)

		got, err := dataStore.GetById(context.Background(), user.Email)
		require.NoError(t, err)

		assert.Equal(t, user, *got)
	})

	t.Run("bad-id", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		_, err := dataStore.GetById(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("not-found", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{}, nil,
		)

		user, err := dataStore.GetById(context.Background(), xid.New().String())
		require.Error(t, err)

		assert.Equal(t, entity.ErrNotFound, err)
		assert.Empty(t, user)
	})
}

func TestUserStore_List(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		users := []entity.User{
			{
				Email: "user1@example.com",
				Name:  "test-user1",
			},
			{
				Email: "user2@example.com",
				Name:  "test-user2",
			},
		}

		mockDBClient.EXPECT().Scan(gomock.Any(), gomock.Any()).Return(
			&dynamodb.ScanOutput{
				Items: []map[string]types.AttributeValue{
					userToUserAttributeValue(users[0]),
					userToUserAttributeValue(users[1]),
				},
				Count: int32(len(users)),
			},
			nil,
		)

		got, err := dataStore.List(context.Background())
		require.NoError(t, err)

		assert.ElementsMatch(t, users, got)
	})
}

func TestUserStore_Put(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		user := entity.User{
			Id:    xid.New().String(),
			Email: "user1@exmaple.com",
			Name:  "test-user1",
		}

		mockDBClient.EXPECT().PutItem(gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, input *dynamodb.PutItemInput) {
				id := input.Item["Id"].(*types.AttributeValueMemberS)
				email := input.Item["Email"].(*types.AttributeValueMemberS)

				assert.Equal(t, user.Id, id.Value)
				assert.Equal(t, user.Email, email.Value)
			},
		).Return(nil, nil)

		err := dataStore.Put(context.Background(), &user)
		require.NoError(t, err)
	})

	t.Run("fallback-update", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		user := entity.User{
			Id:    xid.New().String(),
			Email: "user1@exmaple.com",
			Name:  "test-user1",
		}
		updateUser := user
		updateUser.Email = "user2@example.com"

		mockDBClient.EXPECT().PutItem(gomock.Any(), gomock.Any()).Return(
			nil, &types.ConditionalCheckFailedException{
				Message: aws.String("simulated conditional check failed"),
			},
		)
		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{Item: userToUserAttributeValue(user)}, nil,
		)
		mockDBClient.EXPECT().TransactWriteItems(gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, input *dynamodb.TransactWriteItemsInput) {
				assert.Len(t, input.TransactItems, 3)
			},
		).Return(nil, nil)

		err := dataStore.Put(context.Background(), &updateUser)
		require.NoError(t, err)
	})
}

func TestUserStore_Update(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success-same-email", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		user := entity.User{
			Id:    xid.New().String(),
			Email: "user1@exmaple.com",
			Name:  "test-user1",
		}

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{Item: userToUserAttributeValue(user)}, nil,
		)
		mockDBClient.EXPECT().TransactWriteItems(gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, input *dynamodb.TransactWriteItemsInput) {
				assert.Len(t, input.TransactItems, 1)
			},
		).Return(nil, nil)

		err := dataStore.Update(context.Background(), &user)
		require.NoError(t, err)
	})

	t.Run("success-modified-email", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		user := entity.User{
			Id:    xid.New().String(),
			Email: "user1@exmaple.com",
			Name:  "test-user1",
		}

		updateUser := user
		updateUser.Email = "user2@example.com"

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{Item: userToUserAttributeValue(user)}, nil,
		)
		mockDBClient.EXPECT().TransactWriteItems(gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, input *dynamodb.TransactWriteItemsInput) {
				assert.Len(t, input.TransactItems, 3)
			},
		).Return(nil, nil)

		err := dataStore.Update(context.Background(), &updateUser)
		require.NoError(t, err)
	})
}
