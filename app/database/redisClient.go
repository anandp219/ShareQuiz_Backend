package database

import (
	"github.com/go-redis/redis"
)

//RedisClient exported redis client
var RedisClient *redis.Client

//InitRedis initialises reddit
func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
