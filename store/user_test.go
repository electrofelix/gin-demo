package store_test

import (
	"context"
	"testing"

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

func TestUserStore_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		user := entity.User{
			Email: "user1@example.com",
			Name:  "test-user",
		}

		mockDBClient.EXPECT().DeleteItem(gomock.Any(), gomock.Any()).Return(&dynamodb.DeleteItemOutput{}, nil)

		err := dataStore.Delete(context.Background(), user.Email)
		require.NoError(t, err)
	})

	t.Run("not-found", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		_ = store.NewUserStore(mockDBClient, tableName)
	})
}

func TestUserStore_Get(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		user := entity.User{
			Email: "user1@example.com",
			Name:  "test-user",
		}

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Email":      &types.AttributeValueMemberS{Value: user.Email},
					"Name":       &types.AttributeValueMemberS{Value: user.Name},
					"objectType": &types.AttributeValueMemberS{Value: "User"},
				},
			},
			nil,
		)

		got, err := dataStore.Get(context.Background(), user.Email)
		require.NoError(t, err)

		assert.Equal(t, user, *got)
	})

	t.Run("bad-id", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		_, err := dataStore.Get(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("not-found", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		dataStore := store.NewUserStore(mockDBClient, tableName)

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{}, nil,
		)

		user, err := dataStore.Get(context.Background(), xid.New().String())
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
					{
						"Email":      &types.AttributeValueMemberS{Value: users[0].Email},
						"Name":       &types.AttributeValueMemberS{Value: users[0].Name},
						"objectType": &types.AttributeValueMemberS{Value: "User"},
					},
					{
						"Email":      &types.AttributeValueMemberS{Value: users[1].Email},
						"Name":       &types.AttributeValueMemberS{Value: users[1].Name},
						"objectType": &types.AttributeValueMemberS{Value: "User"},
					},
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
			Email: "user1@exmaple.com",
			Name:  "test-user1",
		}

		mockDBClient.EXPECT().PutItem(gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, input *dynamodb.PutItemInput) {
				email := input.Item["Email"].(*types.AttributeValueMemberS)

				assert.Equal(t, user.Email, email.Value)
			},
		).Return(nil, nil)

		err := dataStore.Put(context.Background(), &user)
		require.NoError(t, err)
	})
}
