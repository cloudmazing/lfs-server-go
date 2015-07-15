package main

import (
	"testing"
	"github.com/bmizerany/assert"
)

//var redisTest = NewRedisClient()

func TestRedisTestLoads(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestRedisNewClient(t *testing.T) {
	client := NewRedisClient().Client
	r, err := client.Ping().Result()
	assert.Equal(t,"PONG", r)
	assert.Equal(t,nil, err)
}


func After() {
	println("Tear Down")
}
