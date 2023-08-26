package repositories

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

type RedisRepo struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisRepo(addr string) (*RedisRepo, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       0,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return &RedisRepo{
		client: client,
		ctx:    ctx,
	}, nil
}

func (rr *RedisRepo) Close() error {
	return rr.client.Close()
}

// Clear all keys in the database
func (rr *RedisRepo) DeleteLeaderboards(leaderboardName string) error {
	// Find and delete keys matching the pattern tournament_id:group_id
	pattern := leaderboardName + ":*"
	iter := rr.client.Scan(rr.ctx, 0, pattern, 0).Iterator()
	for iter.Next(rr.ctx) {
		key := iter.Val()
		_, err := rr.client.Del(rr.ctx, key).Result()
		if err != nil {
			return err
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	return nil
}

// Add a user's score to the leaderboard
func (rr *RedisRepo) AddScoreToLeaderboard(leaderboardKey string, username string, score int) error {
	return rr.client.ZAdd(rr.ctx, leaderboardKey, &redis.Z{
		Score:  float64(score),
		Member: username,
	}).Err()
}

// Get the rank of a user in the leaderboard
func (rr *RedisRepo) GetUserRank(leaderboardKey string, username string) (int64, error) {
	rank, err := rr.client.ZRevRank(rr.ctx, leaderboardKey, username).Result()
	if err != nil {
		return -1, err
	}
	return rank + 1, nil
}

// Get the leaderboard with scores and ranks
func (rr *RedisRepo) GetLeaderboardWithRanks(leaderboardKey string, start, stop int64) ([]map[string]interface{}, error) {
	zRange, err := rr.client.ZRevRangeWithScores(rr.ctx, leaderboardKey, start, stop).Result()
	if err != nil {
		return nil, err
	}

	// Convert the leaderboard to a slice of maps
	leaderboard := make([]map[string]interface{}, len(zRange))
	for i, z := range zRange {
		leaderboard[i] = map[string]interface{}{
			"rank":   i + 1,
			"member": z.Member,
			"score":  int(z.Score),
		}
	}

	return leaderboard, nil
}

// Place a given user into a leaderboard
func (rr *RedisRepo) EnterLeaderboardGroup(leaderboardName, username string, initialScore int) error {
	leaderboardKey := leaderboardName
	log.Printf("Entering user %s into leaderboard %s with initial score %d", username, leaderboardKey, initialScore)

	// Use ZAdd to add the user with an initial score to the leaderboard
	return rr.client.ZAdd(rr.ctx, leaderboardKey, &redis.Z{
		Score:  float64(initialScore),
		Member: username,
	}).Err()
}

// Increment a user's score in a leaderboard
func (rr *RedisRepo) IncrementGroupScore(leaderboardName string, username string) error {
	leaderboardKey := leaderboardName
	return rr.client.ZIncrBy(rr.ctx, leaderboardKey, 1, username).Err()
}

// Get the rank of a user in a leaderboard
func (rr *RedisRepo) GetGroupUserRank(leaderboardName string, username string) (int64, error) {
	leaderboardKey := leaderboardName
	rank, err := rr.client.ZRevRank(rr.ctx, leaderboardKey, username).Result()
	log.Printf("Rank: %d", rank)
	if err != nil {
		return -1, err
	}
	return rank + 1, nil
}

// Get the full leaderboard with scores and ranks
func (rr *RedisRepo) GetGroupLeaderboardWithRanks(leaderboardName string, start, stop int64) ([]map[string]interface{}, error) {
	leaderboardKey := leaderboardName
	zRange, err := rr.client.ZRevRangeWithScores(rr.ctx, leaderboardKey, start, stop).Result()
	if err != nil {
		return nil, err
	}

	// Convert the leaderboard to a slice of maps
	leaderboard := make([]map[string]interface{}, len(zRange))
	for i, z := range zRange {
		leaderboard[i] = map[string]interface{}{
			"rank":   i + 1,
			"member": z.Member,
			"score":  int(z.Score),
		}
	}

	return leaderboard, nil
}

