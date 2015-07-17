package main

import (
	"gopkg.in/redis.v3"
)

type RedisService struct {
	Client *redis.Client
}

func NewRedisClient() *RedisService {
	client := redis.NewClient(&redis.Options{Addr: RedisConfig.Addr, Password: RedisConfig.Password, DB: RedisConfig.DB})
	_, err := client.Ping().Result()
	perror(err)
	return &RedisService{Client: client}
}
