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

func (rr *RedisRepo) CreateLeaderboard(leaderboardName string) error {
	_, err := rr.client.ZAdd(rr.ctx, leaderboardName, &redis.Z{}).Result()
	return err
}

func (rr *RedisRepo) DeleteLeaderboard(leaderboardName string) error {
	_, err := rr.client.Del(rr.ctx, leaderboardName).Result()
	return err
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

func main() {
	repo, err := NewRedisRepo("localhost:6379")
	if err != nil {
		log.Fatalf("Error creating RedisRepo: %v", err)
	}

	// Create or use existing leaderboard
	leaderboardName := "global_leaderboard"
	err = repo.CreateLeaderboard(leaderboardName)
	if err != nil {
		log.Fatalf("Error creating leaderboard: %v", err)
	}

	// Add scores to the leaderboard
	username := "user123"
	score := 1000
	err = repo.AddScoreToLeaderboard(leaderboardName, username, score)
	if err != nil {
		log.Printf("Error adding score to leaderboard: %v", err)
	} else {
		log.Printf("Score added to leaderboard: %s - %s - %d", leaderboardName, username, score)
	}
}