package repositories

import (
	"cloudblast-backend/internal/models"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type DynamoDBRepository struct {
	client *dynamodb.DynamoDB
    mu     sync.Mutex // Mutex for synchronization

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

//USER
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

// Get all users from the database
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

//Get a user by username
func (repo *DynamoDBRepository) GetUserByUsername(username string) (*models.User, error) {
    input := &dynamodb.GetItemInput{
        TableName: aws.String("User"),
        Key: map[string]*dynamodb.AttributeValue{
            "username": {S: aws.String(username)},
        },
    }
    result, err := repo.client.GetItem(input)
    if err != nil {
        return nil, err
    }

    if result == nil || result.Item == nil {
        return nil, nil // User not found
    }

    var user models.User
    if err := dynamodbattribute.UnmarshalMap(result.Item, &user); err != nil {
        return nil, err
    }

    return &user, nil
}

// Get all users in a tournament
func (repo *DynamoDBRepository) GetUsersInTournament(tournamentID string) ([]models.UserInTournament, error) {
    var users []models.UserInTournament

    input := &dynamodb.QueryInput{
        TableName: aws.String("UserInTournament"),
        KeyConditionExpression: aws.String("tournament_id = :tournament_id"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":tournament_id": {S: aws.String(tournamentID)},
        },
    }

    result, err := repo.client.Query(input)
    if err != nil {
        return nil, err
    }

    for _, item := range result.Items {
        var user models.UserInTournament
        err := dynamodbattribute.UnmarshalMap(item, &user)
        if err != nil {
            return nil, err
        }
        users = append(users, user)
    }

    return users, nil
}


//TOURNAMENT
//Create a new tournament given a tournament struct
func (repo *DynamoDBRepository) CreateTournament(tournament *models.Tournament) error {
	av, err := dynamodbattribute.MarshalMap(tournament)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("Tournament"),
		Item:      av,
	}

	_, err = repo.client.PutItem(input)
	if err != nil {
		return err
	}

	return nil
}

//Update a given tournament field with a given value
//Casts the value to the correct type for the DynamoDB table
func (repo *DynamoDBRepository) UpdateTournamentField(tournamentID, fieldName string, value interface{}) error {
	updateExpression := "SET #fieldName = :val"  // Use an expression attribute name for the field name

	attrValue, err := dynamodbattribute.Marshal(value)
	if err != nil {
		return err
	}

	expressionAttributeNames := map[string]*string{
		"#fieldName": aws.String(fieldName),  // Use the actual field name as the alias
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("Tournament"),
		Key: map[string]*dynamodb.AttributeValue{
			"tournament_id": {S: aws.String(tournamentID)},
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

//Search for a tournament by tournament ID
//Returns all tournament data if found and nil if not found
func (repo *DynamoDBRepository) GetTournamentByID(tournamentID string) (*models.Tournament, error) {
	keyCondition := expression.Key("tournament_id").Equal(expression.Value(tournamentID))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String("Tournament"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := repo.client.Query(input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil // Tournament not found
	}

	var tournament models.Tournament
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &tournament)
	if err != nil {
		return nil, err
	}

	return &tournament, nil
}

//Get all tournaments from the database
func (repo *DynamoDBRepository) GetAllTournaments() ([]models.Tournament, error) {
    scanInput := &dynamodb.ScanInput{
        TableName: aws.String("Tournament"),
    }

    result, err := repo.client.Scan(scanInput)
    if err != nil {
        return nil, err
    }

    var tournaments []models.Tournament
    for _, item := range result.Items {
        var tournament models.Tournament
        err := dynamodbattribute.UnmarshalMap(item, &tournament)
        if err != nil {
            return nil, err
        }
        tournaments = append(tournaments, tournament)
    }

    return tournaments, nil
}

//Get the latest started tournament
func (repo *DynamoDBRepository) GetLatestTournament() (string, error) {
    tournaments, err := repo.GetAllTournaments()
    if err != nil {
        
        return "", err
    }

    var latestTournamentID string
    var latestStartTime time.Time

    for _, tournament := range tournaments {
        if tournament.StartTime.After(latestStartTime) {
            latestStartTime = tournament.StartTime
            latestTournamentID = tournament.TournamentID
        }
    }
    return latestTournamentID, nil
}

// Get a user's UserInTournament record for a specific tournament
func (repo *DynamoDBRepository) GetUserInTournamentByUsernameAndTournamentID(username, tournamentID string) (*models.UserInTournament, error) {
	keyCondition := expression.Key("username").Equal(expression.Value(username)).
		And(expression.Key("tournament_id").Equal(expression.Value(tournamentID)))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String("UserInTournament"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := repo.client.Query(input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil // User not found in tournament
	}

	var userInTournament models.UserInTournament
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &userInTournament)
	if err != nil {
		return nil, err
	}

	return &userInTournament, nil
}

// Register a user to the latest tournament
// Returns the group ID if successful, -1 if the user is already registered, -2 if the user does not have enough coins,
// -3 if the user is not at least level 10, and -4 if the user does not exist
// This function is thread-safe to ensure atomicity for the number of registered users
func (repo *DynamoDBRepository) RegisterToTournament(username string) (int, error) {
    // Lock the mutex to ensure atomicity
    repo.mu.Lock()

    // Check if the user has at least 500 coins
    user, err := repo.GetUserByUsername(username)
    if user == nil {
        defer repo.mu.Unlock()
        return -1, nil
    }
    if err != nil {
        defer repo.mu.Unlock()
        return -1, err
    }
     // Check if the user is at least level 10
    if user.Progress_Level < 10 {
        defer repo.mu.Unlock()
        return -4, nil
    }

    if user.Coins < 500 {
        defer repo.mu.Unlock()
        return -3, nil
    }

    tournamentID, err := repo.GetLatestTournament()
    if err != nil {
        defer repo.mu.Unlock()
        return -1, err
    }

    // Check if the user is already registered in the tournament
    existingUserInTournament, err := repo.GetUserInTournamentByUsernameAndTournamentID(username, tournamentID)
    if err != nil {
        defer repo.mu.Unlock()
        return -1, err
    }
    if existingUserInTournament != nil {
        defer repo.mu.Unlock()
        return -2, nil // User already registered
    }

    // Retrieve the current tournament's information
    tournament, err := repo.GetTournamentByID(tournamentID)
    if err != nil {
        defer repo.mu.Unlock()
        return -1, err
    }

    // Calculate group ID, set score and rank as 0, and claimed as false
    currentRegisteredUsers := tournament.NumRegisteredUsers
    groupID := (currentRegisteredUsers / 35) + 1

    newUserInTournament := models.UserInTournament{
        Username:     username,
        TournamentID: tournamentID,
        GroupID:      groupID,
        Score:        0,
        Rank:         0,
        Claimed:      false,
    }

    // Marshal and put the new user in the tournament into the database
    av, err := dynamodbattribute.MarshalMap(newUserInTournament)
    if err != nil {
        defer repo.mu.Unlock()
        return -1, err
    }

    input := &dynamodb.PutItemInput{
        TableName: aws.String("UserInTournament"),
        Item:      av,
    }
    _, err = repo.client.PutItem(input)
    if err != nil {
        defer repo.mu.Unlock()
        return -1, err
    }

    // Update the tournament's number of registered users atomically
    _ = repo.UpdateTournamentField(tournamentID, "num_registered_users", currentRegisteredUsers+1)
    _ = repo.UpdateTournamentField(tournamentID, "latest_group_id", groupID)
    defer repo.mu.Unlock()

    // Update the user's Latest_Tournament_ID field using UpdateUserField
    err = repo.UpdateUserField(username, "latest_tournament_id", tournamentID)
    if err != nil {
        return -1, err
    }

    newCoinBalance := user.Coins - 500
    err = repo.UpdateUserField(username, "coins", newCoinBalance)
    if err != nil {
        return -1, err
    }

    return groupID, nil
}

// Get a user's most recent joined tournament
func (repo *DynamoDBRepository) GetLatestTournamentForUser(username string) (string, error) {
    user, err := repo.GetUserByUsername(username)
    if user == nil {
        return "", nil
    }
    if err != nil {
        return "", err
    }

    latestTournamentID := user.Latest_Tournament_ID
    if latestTournamentID == "" {
        return "", err
    }

    return latestTournamentID, nil
}

// Checks wheter a tournament is finished or not
func (repo *DynamoDBRepository) IsTournamentActive(tournamentID string) (bool, error) {
    tournament, err := repo.GetTournamentByID(tournamentID)
    if err != nil {
        return false, err
    }

    return !tournament.Finished, nil
}

// Increments both the user's progress_level and the tournament's score by 1 - adds coins
func (repo *DynamoDBRepository) IncrementUserScoreInTournament(username, tournamentID string) (int, int, int, error) {
    updateExpression := "SET score = score + :val"

    input := &dynamodb.UpdateItemInput{
        TableName: aws.String("UserInTournament"),
        Key: map[string]*dynamodb.AttributeValue{
            "username":     {S: aws.String(username)},
            "tournament_id": {S: aws.String(tournamentID)},
        },
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":val": {N: aws.String("1")}, // Increment score by 1
        },
        UpdateExpression: aws.String(updateExpression),
        ReturnValues:     aws.String("ALL_NEW"), // Return updated attributes
    }

    result, err := repo.client.UpdateItem(input)
    if err != nil {
        return 0, 0, 0, err
    }

    // Fetch the user's level and coins
    user, err := repo.GetUserByUsername(username)
    if user == nil {
        return 0, 0, 0, nil
    }
    if err != nil {
        return 0, 0, 0, err
    }

    // Increment user's level by one and coins by 100
    err = repo.UpdateUserField(username, "progress_level", user.Progress_Level+1)
    if err != nil {
        return 0, 0, 0, err
    }

    err = repo.UpdateUserField(username, "coins", user.Coins+100)
    if err != nil {
        return 0, 0, 0, err
    }

    score, err := strconv.Atoi(*result.Attributes["score"].N)
    if err != nil {
        return 0, 0, 0, err
    }

    return user.Progress_Level + 1, user.Coins + 100, score, nil
}


// Update a user-tournament touple information
func (repo *DynamoDBRepository) UpdateUserInTournamentRank(username, tournamentID string, rank int) error {
    updateExpression := "SET #rk = :rank"

    input := &dynamodb.UpdateItemInput{
        TableName: aws.String("UserInTournament"),
        Key: map[string]*dynamodb.AttributeValue{
            "username":     {S: aws.String(username)},
            "tournament_id": {S: aws.String(tournamentID)},
        },
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":rank": {N: aws.String(strconv.Itoa(rank))},
        },
        ExpressionAttributeNames: map[string]*string{
            "#rk": aws.String("rank"),
        },
        UpdateExpression: aws.String(updateExpression),
    }

    _, err := repo.client.UpdateItem(input)
    return err
}

// Find the top four players in the latest tournament and assign ranks to them
func (repo *DynamoDBRepository) FindRankedPlayersForLatestTournament() ([]models.UserInTournament, error) {
    latestTournamentID, err := repo.GetLatestTournament()
    if err != nil {
        log.Printf("Failed to fetch latest tournament: %v", err)
        return nil, err
    }

    // Fetch all UserInTournament records for the latest tournament
    usersInTournament, err := repo.GetUsersInTournament(latestTournamentID)
    if err != nil {
        log.Printf("Failed to fetch users in tournament: %v", err)
        return nil, err
    }

    // Sort usersInTournament based on score in descending order
    if len(usersInTournament) >= 1 {
        sort.Slice(usersInTournament, func(i, j int) bool {
            return usersInTournament[i].Score > usersInTournament[j].Score
        })
    }

    // Assign ranks to the top four players
    for i := 0; i < len(usersInTournament) && i < 4; i++ {
        if i < 4{
            usersInTournament[i].Rank = i + 1
        } else {
            usersInTournament[i].Rank = -1
        }
    }

    // Update the UserInTournament records with ranks
    for _, user := range usersInTournament {
        err = repo.UpdateUserInTournamentRank(user.Username, user.TournamentID, user.Rank)
        if err != nil {
            log.Printf("Failed to update user in tournament rank: %v", err)
            return nil, err
        }
    }

    userCount := len(usersInTournament)
    if(userCount == 0){
        return []models.UserInTournament{}, nil
    }
    if(userCount < 4){
        return usersInTournament[:userCount], nil
    }

    return usersInTournament[:4], nil
}

// Find the latest tournament and set its finished field to true
func (repo *DynamoDBRepository) EndLatestTournament() (string, error) {
    latestTournamentID, err := repo.GetLatestTournament()
    if err != nil {
        log.Printf("Failed to fetch latest tournament: %v", err)
        return "", err
    }

    err = repo.UpdateTournamentField(latestTournamentID, "finished", true)
    if err != nil {
        log.Printf("Failed to update tournament finished field: %v", err)
        return "", err
    }

    return latestTournamentID, nil
}

// Mark a player's reward as claimed
func (repo *DynamoDBRepository) UpdateUserInTournamentClaimed(username, tournamentID string, claimed bool) error {
    updateExpression := "SET #cl = :claimed"

    input := &dynamodb.UpdateItemInput{
        TableName: aws.String("UserInTournament"),
        Key: map[string]*dynamodb.AttributeValue{
            "username":     {S: aws.String(username)},
            "tournament_id": {S: aws.String(tournamentID)},
        },
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":claimed": {BOOL: aws.Bool(claimed)},
        },
        ExpressionAttributeNames: map[string]*string{
            "#cl": aws.String("claimed"),
        },
        UpdateExpression: aws.String(updateExpression),
    }

    _, err := repo.client.UpdateItem(input)
    return err
}

// Check if a tournament is finished
func (repo *DynamoDBRepository) IsTournamentFinished(tournamentID string) (bool, error) {
    tournament, err := repo.GetTournamentByID(tournamentID)
    if err != nil {
        return false, err
    }

    return tournament.Finished, nil
}

// Get the user's latest group ID
// Returns -1 if the user does not exist or -2 if the user has not registered to a tournament
func (repo *DynamoDBRepository) GetLatestGroupIdForUser(username string) (int, error) {
    user, err := repo.GetUserByUsername(username)
    if user == nil {
        return -1, nil
    }
    if err != nil {
        return -1, err
    }

    latest_Group_ID := user.Latest_Group_ID
    

    if latest_Group_ID == -1 {
        return -2, err
    }

    return latest_Group_ID, nil
}

// Check whether a user has claimed the reward for the latest tournament
func (repo *DynamoDBRepository) DidUserClaimReward(username string) (bool, error) {
    // Get the user's latest_tournament_id
    latestTournamentID, err := repo.GetLatestTournamentForUser(username)

    if err != nil {
        return false, err
    }

    if latestTournamentID == "" {
        return true, nil
    }

    // Check if the user claimed the reward for the latest tournament
    input := &dynamodb.GetItemInput{
        TableName: aws.String("UserInTournament"),
        Key: map[string]*dynamodb.AttributeValue{
            "tournament_id": {
                S: aws.String(latestTournamentID),
            },
            "username": {
                S: aws.String(username),
            },
        },
    }

    result, err := repo.client.GetItem(input)
    if err != nil {
        return false, err
    }

    if result == nil || result.Item == nil {
        return false, nil
    }

    rewardClaimedAttr := result.Item["claimed"]
    rankAttr := result.Item["rank"]

    log.Printf("Item: %v", result.Item)
    if rewardClaimedAttr == nil || rewardClaimedAttr.BOOL == nil || rankAttr == nil || rankAttr.N == nil {
        return false, nil
    }

    rewardClaimed := *rewardClaimedAttr.BOOL
    rank := *rankAttr.N

    log.Printf("Reward-Rank: %v - %v", rewardClaimed, rank)

    return (rewardClaimed == true || rank == "0"), nil
}
// Get the top 1000 users from the same country as the given username
func (repo *DynamoDBRepository) GetCountryLeaderboard(username string) ([]models.User, error) {
    // Get the country of the given username
    country, err := repo.GetCountryForUser(username)
    if err != nil {
        return nil, err
    }

    // Scan users from the same country
    input := &dynamodb.ScanInput{
        TableName: aws.String("User"),
        FilterExpression: aws.String("country = :country"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":country": {
                S: aws.String(country),
            },
        },
    }

    result, err := repo.client.Scan(input)
    if err != nil {
        return nil, err
    }

    // Filter and sort users
    var users []models.User
    for _, item := range result.Items {
        user := models.User{}
        err := dynamodbattribute.UnmarshalMap(item, &user)
        if err != nil {
            return nil, err
        }
        users = append(users, user)
    }

    // Sort users by progress level in descending order
    sort.SliceStable(users, func(i, j int) bool {
        return users[i].Progress_Level > users[j].Progress_Level
    })

    // Limit to the first 1000 users (or less if fewer users exist)
    maxUsers := 1000
    if len(users) < maxUsers {
        maxUsers = len(users)
    }
    return users[:maxUsers], nil
}
func (repo *DynamoDBRepository) GetCountryForUser(username string) (string, error) {
    input := &dynamodb.GetItemInput{
        TableName: aws.String("User"),
        Key: map[string]*dynamodb.AttributeValue{
            "username": {
                S: aws.String(username),
            },
        },
        ProjectionExpression: aws.String("country"),
    }

    result, err := repo.client.GetItem(input)
    if err != nil {
        return "", err
    }

    if result.Item == nil {
        return "", err
    }

    countryAttr := result.Item["country"]
    if countryAttr == nil || countryAttr.S == nil {
        return "", err
    }

    return *countryAttr.S, nil
}

// Get the top 1000 users globally
func (repo *DynamoDBRepository) GetGlobalLeaderboard() ([]models.User, error) {
    // Scan all users
    input := &dynamodb.ScanInput{
        TableName: aws.String("User"),
    }

    result, err := repo.client.Scan(input)
    if err != nil {
        return nil, err
    }

    // Filter and sort users
    var users []models.User
    for _, item := range result.Items {
        user := models.User{}
        err := dynamodbattribute.UnmarshalMap(item, &user)
        if err != nil {
            return nil, err
        }
        users = append(users, user)
    }

    // Sort users by progress level in descending order
    sort.SliceStable(users, func(i, j int) bool {
        return users[i].Progress_Level > users[j].Progress_Level
    })

    // Limit to the first 1000 users (or less if fewer users exist)
    maxUsers := 1000
    if len(users) < maxUsers {
        maxUsers = len(users)
    }
    return users[:maxUsers], nil
}
