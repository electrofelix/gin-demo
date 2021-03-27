package store

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/database_mocks.go -package=mocks github.com/electrofelix/gin-demo/store DynamoDBAPI

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/electrofelix/gin-demo/entity"
	"github.com/sirupsen/logrus"
)

const (
	key = "UserInfo"
)

type DynamoDBOptions = func(*dynamodb.Options)

type DynamoDBAPI interface {
	CreateTable(context.Context, *dynamodb.CreateTableInput, ...DynamoDBOptions) (*dynamodb.CreateTableOutput, error)
	GetItem(context.Context, *dynamodb.GetItemInput, ...DynamoDBOptions) (*dynamodb.GetItemOutput, error)
	DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...DynamoDBOptions) (*dynamodb.DeleteItemOutput, error)
	ListTables(context.Context, *dynamodb.ListTablesInput, ...DynamoDBOptions) (*dynamodb.ListTablesOutput, error)
	PutItem(context.Context, *dynamodb.PutItemInput, ...DynamoDBOptions) (*dynamodb.PutItemOutput, error)
	Scan(context.Context, *dynamodb.ScanInput, ...DynamoDBOptions) (*dynamodb.ScanOutput, error)
}

type UserStore struct {
	dbClient  DynamoDBAPI
	tableName string
	logger    *logrus.Logger
}

type Option func(*UserStore)

func NewUserStore(dbClient DynamoDBAPI, dbTable string, options ...Option) *UserStore {
	us := &UserStore{
		dbClient:  dbClient,
		tableName: dbTable,
		logger:    logrus.StandardLogger(),
	}

	for _, opt := range options {
		opt(us)
	}

	return us
}

func (us *UserStore) InitializeTable(ctx context.Context) error {
	us.logger.Infoln("Table initializing")
	// useful for dev environment, probably better to avoid granting
	// permissions in a production environment if the data is critical
	result, err := us.dbClient.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		return err
	}

	us.logger.Infoln("Found tables:", result.TableNames)

	for _, name := range result.TableNames {
		if name == us.tableName {
			us.logger.Infof("table '%s' already exists, skipping initialization", us.tableName)

			return nil
		}
	}

	us.logger.Infof("table '%s' not found, attemting bootstrap", us.tableName)

	_, err = us.dbClient.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: &us.tableName,
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("Email"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("objectType"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("Email"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("objectType"),
				KeyType:       types.KeyTypeRange,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	})
	if err != nil {
		return err
	}

	us.logger.Infof("table %s created successfully", us.tableName)

	return nil
}

func (us *UserStore) Create(ctx context.Context, user *entity.User) error {
	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		us.logger.Errorf("Marshal failed for user (%s): %v", user.Email, err)

		return err
	}

	item["objectType"] = &types.AttributeValueMemberS{Value: key}

	putItem := dynamodb.PutItemInput{
		Item:                item,
		TableName:           aws.String(us.tableName),
		ConditionExpression: aws.String("attribute_not_exists(Email)"),
	}

	_, err = us.dbClient.PutItem(ctx, &putItem)
	if err != nil {
		if errors.Is(err, &types.ConditionalCheckFailedException{}) {
			return entity.ErrIDCollision
		}

		// some other error, such as capacity provisioned exceeded or failure
		// to talk to the endpoint
		us.logger.Errorf("error putting item %s: %v", key, item["Email"])

		return err
	}

	return nil
}

func (us *UserStore) Delete(ctx context.Context, id string) error {

	deleteItem := dynamodb.DeleteItemInput{
		Key: map[string]types.AttributeValue{
			"Email":      &types.AttributeValueMemberS{Value: id},
			"objectType": &types.AttributeValueMemberS{Value: key},
		},
		TableName: aws.String(us.tableName),
	}

	// output, ignored, includes metrics such as capacity consumed, which would be
	// use to emit via prometheus
	_, err := us.dbClient.DeleteItem(ctx, &deleteItem)
	if err != nil {
		us.logger.Errorf("error during delete: %v", err)

		return err
	}

	return nil
}

func (us *UserStore) Get(ctx context.Context, id string) (*entity.User, error) {
	if id == "" {
		return nil, entity.ErrIDMissing
	}

	getItem := dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"Email": &types.AttributeValueMemberS{Value: id},
			// using a sort key makes it easier to split the object into
			// multiple pieces for storing if needed in the future as the object grows
			"objectType": &types.AttributeValueMemberS{Value: key},
		},
		TableName: aws.String(us.tableName),
	}

	result, err := us.dbClient.GetItem(ctx, &getItem)
	if err != nil {
		us.logger.Errorf("error during get: %v", err)

		return nil, err
	}

	user := entity.User{}

	err = attributevalue.UnmarshalMap(result.Item, &user)
	if err != nil {
		us.logger.Errorf("error unmarshaling %s %s: %v", key, id, err)

		return nil, err
	}

	if user == (entity.User{}) {
		return nil, entity.ErrNotFound
	}

	return &user, nil
}

func (us *UserStore) List(ctx context.Context) ([]entity.User, error) {
	scanInput := dynamodb.ScanInput{
		TableName:        aws.String(us.tableName),
		FilterExpression: aws.String(fmt.Sprintf("objectType <> %s", key)),
	}

	// ideally switch to using a secondary index based on the objectType
	// and use query instead of scan.
	result, err := us.dbClient.Scan(ctx, &scanInput)
	if err != nil {
		us.logger.Errorf("error during scan: %v", err)

		return nil, err
	}

	users := make([]entity.User, result.Count)

	err = attributevalue.UnmarshalListOfMaps(result.Items, &users)
	if err != nil {
		us.logger.Errorf("error unmarshaling %s: %v", key, err)

		return nil, err
	}

	return users, nil
}

func (us *UserStore) Put(ctx context.Context, user *entity.User) error {
	if user.Email == "" {
		return entity.ErrIDMissing
	}

	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		us.logger.Errorf("Marshal failed for user (%s): %v", user.Email, err)

		return err
	}

	item["objectType"] = &types.AttributeValueMemberS{Value: key}

	putItem := dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(us.tableName),
	}

	_, err = us.dbClient.PutItem(ctx, &putItem)
	if err != nil {
		us.logger.Errorf("error putting item %s: %v", key, item["Email"])

		return err
	}

	return nil
}
