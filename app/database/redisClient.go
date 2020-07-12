package database

import (
	"os"

	"github.com/go-redis/redis"
)

//RedisClient exported redis client
var RedisClient *redis.Client

//InitRedis initialises reddit
func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
