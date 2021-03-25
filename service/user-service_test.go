package service_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/electrofelix/gin-demo/entity"
	"github.com/electrofelix/gin-demo/mocks"
	"github.com/electrofelix/gin-demo/service"
	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tableName = "test-table"
)

func TestUserService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		svc := service.New(mockDBClient, tableName)

		user := entity.User{
			Id:   xid.New().String(),
			Name: "test-user",
		}

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Id":   &types.AttributeValueMemberS{Value: user.Id},
					"Name": &types.AttributeValueMemberS{Value: user.Name},
				},
			},
			nil,
		)

		mockDBClient.EXPECT().DeleteItem(gomock.Any(), gomock.Any()).Return(&dynamodb.DeleteItemOutput{}, nil)

		got, err := svc.Delete(context.Background(), user.Id)
		require.NoError(t, err)

		assert.Equal(t, user, got)
	})

	t.Run("not-found", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		_ = service.New(mockDBClient, tableName)
	})
}

func TestUserService_Get(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		svc := service.New(mockDBClient, tableName)

		user := entity.User{
			Id:   xid.New().String(),
			Name: "test-user",
		}

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Id":         &types.AttributeValueMemberS{Value: user.Id},
					"Name":       &types.AttributeValueMemberS{Value: user.Name},
					"objectType": &types.AttributeValueMemberS{Value: "User"},
				},
			},
			nil,
		)

		got, err := svc.Get(context.Background(), user.Id)
		require.NoError(t, err)

		assert.Equal(t, user, got)
	})

	t.Run("bad-id", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		svc := service.New(mockDBClient, tableName)

		_, err := svc.Get(context.Background(), "")
		assert.Error(t, err)

		_, err = svc.Get(context.Background(), "bad-id")
		assert.Error(t, err)
	})

	t.Run("not-found", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		svc := service.New(mockDBClient, tableName)

		mockDBClient.EXPECT().GetItem(gomock.Any(), gomock.Any()).Return(
			&dynamodb.GetItemOutput{}, nil,
		)

		user, err := svc.Get(context.Background(), xid.New().String())
		require.NoError(t, err)

		assert.Equal(t, entity.User{}, user)
	})
}

func TestUserService_List(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		svc := service.New(mockDBClient, tableName)

		users := []entity.User{
			{
				Id:   xid.New().String(),
				Name: "test-user1",
			},
			{
				Id:   xid.New().String(),
				Name: "test-user2",
			},
		}

		mockDBClient.EXPECT().Scan(gomock.Any(), gomock.Any()).Return(
			&dynamodb.ScanOutput{
				Items: []map[string]types.AttributeValue{
					{
						"Id":         &types.AttributeValueMemberS{Value: users[0].Id},
						"Name":       &types.AttributeValueMemberS{Value: users[0].Name},
						"objectType": &types.AttributeValueMemberS{Value: "User"},
					},
					{
						"Id":         &types.AttributeValueMemberS{Value: users[1].Id},
						"Name":       &types.AttributeValueMemberS{Value: users[1].Name},
						"objectType": &types.AttributeValueMemberS{Value: "User"},
					},
				},
				Count: int32(len(users)),
			},
			nil,
		)

		got, err := svc.List(context.Background())
		require.NoError(t, err)

		assert.ElementsMatch(t, users, got)
	})
}

func TestUserService_Put(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("success", func(t *testing.T) {
		mockDBClient := mocks.NewMockDynamoDBAPI(ctrl)
		svc := service.New(mockDBClient, tableName)

		user := entity.User{
			Id:   xid.New().String(),
			Name: "test-user1",
		}

		mockDBClient.EXPECT().PutItem(gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, input *dynamodb.PutItemInput) {
				id := input.Item["Id"].(*types.AttributeValueMemberS)

				assert.Equal(t, user.Id, id.Value)
			},
		).Return(nil, nil)

		got, err := svc.Put(context.Background(), user)
		require.NoError(t, err)

		assert.Equal(t, user, got)
	})
}