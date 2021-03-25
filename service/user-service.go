package service

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/user-service_mocks.go -package=mocks github.com/electrofelix/gin-demo/service DynamoDBAPI

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"

	"github.com/electrofelix/gin-demo/entity"
)

const (
	key = "User"
)

type DynamoDBOptions = func(*dynamodb.Options)

type DynamoDBAPI interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...DynamoDBOptions) (*dynamodb.GetItemOutput, error)
	DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...DynamoDBOptions) (*dynamodb.DeleteItemOutput, error)
	PutItem(context.Context, *dynamodb.PutItemInput, ...DynamoDBOptions) (*dynamodb.PutItemOutput, error)
	Scan(context.Context, *dynamodb.ScanInput, ...DynamoDBOptions) (*dynamodb.ScanOutput, error)
}

type UserService struct {
	tableName      string
	dynamodbClient DynamoDBAPI
	logger         *logrus.Logger
}

type Option func(*UserService)

func New(dynamodbClient DynamoDBAPI, dbTable string, options ...Option) *UserService {
	us := &UserService{
		dynamodbClient: dynamodbClient,
		tableName:      dbTable,
		logger:         logrus.StandardLogger(),
	}

	for _, opt := range options {
		opt(us)
	}

	return us
}

func (us *UserService) Delete(ctx context.Context, id string) (entity.User, error) {

	item, err := us.Get(ctx, id)
	if err != nil {
		us.logger.Errorf("error retrieving item before delete: %v", err)

		return entity.User{}, err
	}

	deleteItem := dynamodb.DeleteItemInput{
		Key: map[string]types.AttributeValue{
			"Id":         &types.AttributeValueMemberS{Value: id},
			"objectType": &types.AttributeValueMemberS{Value: key},
		},
		TableName: aws.String(us.tableName),
	}

	// output, ignored, includes metrics such as capacity consumed, which would be
	// use to emit via prometheus
	_, err = us.dynamodbClient.DeleteItem(ctx, &deleteItem)
	if err != nil {
		us.logger.Errorf("error during delete: %v", err)

		return entity.User{}, nil
	}

	return item, nil
}

func (us *UserService) Get(ctx context.Context, id string) (entity.User, error) {
	if _, err := xid.FromString(id); err != nil {
		return entity.User{}, err
	}

	getItem := dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"Id": &types.AttributeValueMemberS{Value: id},
			// using a sort key makes it easier to split the object into
			// multiple pieces for storing if needed in the future as the object grows
			"objectType": &types.AttributeValueMemberS{Value: key},
		},
		TableName: aws.String(us.tableName),
	}

	result, err := us.dynamodbClient.GetItem(ctx, &getItem)
	if err != nil {
		us.logger.Errorf("error during delete: %v", err)

		return entity.User{}, nil
	}

	user := entity.User{}

	err = attributevalue.UnmarshalMap(result.Item, &user)
	if err != nil {
		return user, fmt.Errorf("failed to unmarshal Items, %w", err)
	}

	return user, nil
}

func (us *UserService) List(ctx context.Context) ([]entity.User, error) {
	scanInput := dynamodb.ScanInput{
		TableName:        aws.String(us.tableName),
		FilterExpression: aws.String(fmt.Sprintf("objectType = %s", key)),
	}

	// ideally switch to using a secondary index based on the objectType
	// and use query instead of scan.
	result, err := us.dynamodbClient.Scan(ctx, &scanInput)
	if err != nil {
		us.logger.Errorf("error during scan: %v", err)
	}

	users := make([]entity.User, result.Count)

	err = attributevalue.UnmarshalListOfMaps(result.Items, &users)
	if err != nil {
		us.logger.Errorf("error unmarshaling %s: %v", key, err)

		return []entity.User{}, err
	}

	return users, nil
}

func (us *UserService) Put(ctx context.Context, user entity.User) (entity.User, error) {
	if _, err := xid.FromString(user.Id); err != nil {
		return entity.User{}, err
	}

	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		us.logger.Errorf("Marshal failed for user (%s): %v", user.Id, err)
	}

	item["objectType"] = &types.AttributeValueMemberS{Value: key}

	putItem := dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(us.tableName),
	}

	_, err = us.dynamodbClient.PutItem(ctx, &putItem)
	if err != nil {
		us.logger.Errorf("error putting item %s: %v", key, item["Id"])

		return entity.User{}, err
	}

	return user, nil
}
