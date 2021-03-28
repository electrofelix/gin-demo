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
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...DynamoDBOptions) (*dynamodb.TransactWriteItemsOutput, error)
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
				AttributeName: aws.String("Id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("objectType"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("Id"),
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

	transaction := dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					Item:                item,
					TableName:           aws.String(us.tableName),
					ConditionExpression: aws.String("attribute_not_exists(Id)"),
				},
			},
			{
				// save a second object at the same time where the email is the Id
				Put: &types.Put{
					Item: map[string]types.AttributeValue{
						"Id": &types.AttributeValueMemberS{
							Value: user.Email,
						},
						"UserId": &types.AttributeValueMemberS{
							Value: user.Id,
						},
						"objectType": &types.AttributeValueMemberS{
							Value: fmt.Sprintf("%s#email", key),
						},
					},
					TableName:           aws.String(us.tableName),
					ConditionExpression: aws.String("attribute_not_exists(Id)"),
				},
			},
		},
	}

	_, err = us.dbClient.TransactWriteItems(ctx, &transaction)
	if err != nil {
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) {
			return entity.ErrIDCollision
		}

		// some other error, such as capacity provisioned exceeded or failure
		// to talk to the endpoint
		us.logger.Errorf("error putting item %s %s for %s: %v", key, user.Id, user.Email, err)

		return err
	}

	return nil
}

func (us *UserStore) Delete(ctx context.Context, id string) error {
	// should update Delete to require the object not just the id, for
	// now retrieve first to have access to the email for the delete
	user, err := us.GetById(ctx, id)
	if err != nil {
		return err
	}

	transaction := dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				// main object
				Delete: &types.Delete{
					Key: map[string]types.AttributeValue{
						"Id":         &types.AttributeValueMemberS{Value: id},
						"objectType": &types.AttributeValueMemberS{Value: key},
					},
					TableName: aws.String(us.tableName),
				},
			},
			{
				// secondary object for unique email
				Delete: &types.Delete{
					Key: map[string]types.AttributeValue{
						"Id": &types.AttributeValueMemberS{
							Value: user.Email,
						},
						"objectType": &types.AttributeValueMemberS{
							Value: fmt.Sprintf("%s#email", key),
						},
					},
					TableName: aws.String(us.tableName),
				},
			},
		},
	}

	// output, ignored, includes metrics such as capacity consumed, which would be
	// use to emit via prometheus
	_, err = us.dbClient.TransactWriteItems(ctx, &transaction)
	if err != nil {
		us.logger.Errorf("error during delete: %v", err)

		return err
	}

	return nil
}

func (us *UserStore) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	if email == "" {
		return nil, entity.ErrIDMissing
	}

	getItem := dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"Id": &types.AttributeValueMemberS{
				Value: email,
			},
			"objectType": &types.AttributeValueMemberS{
				Value: fmt.Sprintf("%s#email", key),
			},
		},
		TableName: aws.String(us.tableName),
	}

	result, err := us.dbClient.GetItem(ctx, &getItem)
	if err != nil {
		us.logger.Errorf("error during get: %v", err)

		return nil, err
	}

	id := result.Item["UserId"]
	if id == nil {
		return nil, entity.ErrNotFound
	}

	var userId string

	err = attributevalue.Unmarshal(id, &userId)
	if err != nil {
		us.logger.Errorf("error unmarshaling %s %s: %v", key, id, err)

		return nil, err
	}

	return us.GetById(ctx, userId)
}

func (us *UserStore) GetById(ctx context.Context, id string) (*entity.User, error) {
	if id == "" {
		return nil, entity.ErrIDMissing
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
		FilterExpression: aws.String("objectType = :type"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":type": &types.AttributeValueMemberS{
				Value: key,
			},
		},
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
	if user.Id == "" {
		return entity.ErrIDMissing
	}

	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		us.logger.Errorf("Marshal failed for user (%s): %v", user.Id, err)

		return err
	}

	item["objectType"] = &types.AttributeValueMemberS{Value: key}

	// attempt a put item first with the optimistic view that it'll be
	// rare to update the email address field.
	putItem := dynamodb.PutItemInput{
		Item:                item,
		TableName:           aws.String(us.tableName),
		ConditionExpression: aws.String("Email = :email"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{
				Value: user.Email,
			},
		},
	}

	_, err = us.dbClient.PutItem(ctx, &putItem)
	if err != nil {
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) {
			// email is being modified, fall back to attempting an Update
			// can invert this later once switched to using update which
			// can determine whether it can simply call put or needs to
			// perform a full transact multiple entry update
			return us.Update(ctx, user)
		}

		us.logger.Errorf("error putting item %s %s: %v", key, item, err)

		return err
	}

	return nil
}

// Update performs a get first in order to determine if additional operations
// must be performed in case the field requires special handling.
// Emails must be unique in addition to the Id, therefore for dynamodb
// that means being a primary key and requires two PutItems to be
// performed as well as a DeleteItem to remove the old email
func (us *UserStore) Update(ctx context.Context, user *entity.User) error {
	if user.Id == "" {
		return entity.ErrIDMissing
	}

	currentUser, err := us.GetById(ctx, user.Id)
	if err != nil {
		return err
	}

	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		us.logger.Errorf("Marshal failed for user (%s): %v", user.Id, err)

		return err
	}

	item["objectType"] = &types.AttributeValueMemberS{Value: key}

	transaction := dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					Item:      item,
					TableName: aws.String(us.tableName),
				},
			},
		},
	}

	if user.Email != currentUser.Email {
		// need to append insertion of the new entry for an email address
		// and removal of the old as a single transaction
		transaction.TransactItems = append(
			transaction.TransactItems,
			types.TransactWriteItem{
				Put: &types.Put{
					Item: map[string]types.AttributeValue{
						"Id": &types.AttributeValueMemberS{
							Value: user.Email,
						},
						"UserId": &types.AttributeValueMemberS{
							Value: user.Id,
						},
						"objectType": &types.AttributeValueMemberS{
							Value: fmt.Sprintf("%s#email", key),
						},
					},
					TableName:           aws.String(us.tableName),
					ConditionExpression: aws.String("attribute_not_exists(Id)"),
				},
			},
			types.TransactWriteItem{
				// secondary object for unique email
				Delete: &types.Delete{
					Key: map[string]types.AttributeValue{
						"Id": &types.AttributeValueMemberS{
							Value: currentUser.Email,
						},
						"objectType": &types.AttributeValueMemberS{
							Value: fmt.Sprintf("%s#email", key),
						},
					},
					TableName: aws.String(us.tableName),
				},
			},
		)
	}

	_, err = us.dbClient.TransactWriteItems(ctx, &transaction)
	if err != nil {
		us.logger.Errorf("error putting item %s: %v", key, user.Id)

		return err
	}

	return nil
}
