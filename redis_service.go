package main

import (
	"fmt"
	"gopkg.in/redis.v3"
)

type RedisService struct {
	Client *redis.Client
}

func NewRedisClient() *RedisService {
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", Config.Redis.Host, Config.Redis.Port), Password: Config.Redis.Password, DB: Config.Redis.DB})
	_, err := client.Ping().Result()
	perror(err)
	return &RedisService{Client: client}
}
