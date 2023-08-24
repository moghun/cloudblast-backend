package repositories

import (
	"goodBlast-backend/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type DynamoDBRepository struct {
	client *dynamodb.DynamoDB
}

//NewDynamoDBRepository creates a new DynamoDB repository
func NewDynamoDBRepository() (*DynamoDBRepository, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-central-1"),
	})
	if err != nil {
		return nil, err
	}

	client := dynamodb.New(sess)
	return &DynamoDBRepository{client: client}, nil
}

//Create a new user given a user struct
func (repo *DynamoDBRepository) CreateUser(user *models.User) error {
    av, err := dynamodbattribute.MarshalMap(user)
    if err != nil {
        return err
    }

    input := &dynamodb.PutItemInput{
        TableName: aws.String("User"),
        Item:      av,
    }

    _, err = repo.client.PutItem(input)
    if err != nil {
        return err
    }

    return nil
}

//Update a given user field with a given value
//Casts the value to the correct type for the DynamoDB table
func (repo *DynamoDBRepository) UpdateUserField(username, fieldName string, value interface{}) error {
    updateExpression := "SET #fieldName = :val"  // Use an expression attribute name for the field name

    attrValue, err := dynamodbattribute.Marshal(value)
    if err != nil {
        return err
    }

    expressionAttributeNames := map[string]*string{
        "#fieldName": aws.String(fieldName),  // Use the actual field name as the alias
    }

    input := &dynamodb.UpdateItemInput{
        TableName: aws.String("User"),
        Key: map[string]*dynamodb.AttributeValue{
            "username": {S: aws.String(username)},
        },
        ExpressionAttributeNames:  expressionAttributeNames,
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":val": attrValue,
        },
        UpdateExpression: aws.String(updateExpression),
    }

    _, err = repo.client.UpdateItem(input)
    if err != nil {
        return err
    }

    return nil
}



func (repo *DynamoDBRepository) GetAllUsers() ([]models.User, error) {
    input := &dynamodb.ScanInput{
        TableName: aws.String("User"),
    }

    result, err := repo.client.Scan(input)
    if err != nil {
        return nil, err
    }

    users := make([]models.User, 0)
    for _, item := range result.Items {
        var user models.User
        err := dynamodbattribute.UnmarshalMap(item, &user)
        if err != nil {
            return nil, err
        }
        users = append(users, user)
    }

    return users, nil
}

//Search for a user by username
//Returns all user data if found and nil if not found
func (repo *DynamoDBRepository) GetUserByUsername(username string) (*models.User, error) {
    keyCondition := expression.Key("username").Equal(expression.Value(username))

    expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
    if err != nil {
        return nil, err
    }

    input := &dynamodb.QueryInput{
        TableName:                 aws.String("User"),
        KeyConditionExpression:    expr.KeyCondition(),
        ExpressionAttributeNames:  expr.Names(),
        ExpressionAttributeValues: expr.Values(),
    }

    result, err := repo.client.Query(input)
    if err != nil {
        return nil, err
    }

    if len(result.Items) == 0 {
        return nil, nil // User not found
    }

    var user models.User
    err = dynamodbattribute.UnmarshalMap(result.Items[0], &user)
    if err != nil {
        return nil, err
    }

    return &user, nil
}