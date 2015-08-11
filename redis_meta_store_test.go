package main

import (
	"fmt"
	"gopkg.in/redis.v3"
	"os"
	"testing"
)

var (
	redisRedisMetaStore *RedisMetaStore
)

func TestRedisGetWithAuth(t *testing.T) {
	setupRedisMetaStore()
	defer teardownRedisMetaStore()

	meta, err := redisRedisMetaStore.Get(&RequestVars{Authorization: testAuth, Oid: contentOid})
	if err != nil {
		t.Fatalf("Error retreiving meta: %s", err)
	}

	if meta.Oid != contentOid {
		t.Errorf("expected to get content oid, got: %s", meta.Oid)
	}

	if meta.Size != contentSize {
		t.Errorf("expected to get content size, got: %d", meta.Size)
	}
}

func TestRedisGetWithoutAuth(t *testing.T) {
	setupRedisMetaStore()
	defer teardownRedisMetaStore()

	_, err := redisRedisMetaStore.Get(&RequestVars{Authorization: badAuth, Oid: contentOid})
	if !isAuthError(err) {
		t.Errorf("expected auth error, got: %s", err)
	}
}

func TestRedisPutWithAuth(t *testing.T) {
	setupRedisMetaStore()
	defer teardownRedisMetaStore()

	meta, err := redisRedisMetaStore.Put(&RequestVars{Authorization: testAuth, Oid: nonexistingOid, Size: 42})
	if err != nil {
		t.Errorf("expected put to succeed, got : %s", err)
	}

	if meta.Existing {
		t.Errorf("expected meta to not have existed")
	}

	meta, err = redisRedisMetaStore.Get(&RequestVars{Authorization: testAuth, Oid: nonexistingOid})
	if err != nil {
		t.Errorf("expected to be able to retreive new put, got : %s", err)
	}

	if meta.Oid != nonexistingOid {
		t.Errorf("expected oids to match, got: %s", meta.Oid)
	}

	if meta.Size != 42 {
		t.Errorf("expected sizes to match, got: %d", meta.Size)
	}

	meta, err = redisRedisMetaStore.Put(&RequestVars{Authorization: testAuth, Oid: nonexistingOid, Size: 42})
	if err != nil {
		t.Errorf("expected put to succeed, got : %s", err)
	}

	if !meta.Existing {
		t.Errorf("expected meta to now exist")
	}
}

func TestRedisPutWithoutAuth(t *testing.T) {
	setupRedisMetaStore()
	defer teardownRedisMetaStore()

	_, err := redisRedisMetaStore.Put(&RequestVars{Authorization: badAuth, Oid: contentOid, Size: 42})
	if !isAuthError(err) {
		t.Errorf("expected auth error, got: %s", err)
	}
}

func setupRedisMetaStore() {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: Config.Redis.Password, DB: Config.Redis.DB})
	_, err := client.Ping().Result()
	perror(err)
	store, err := &RedisMetaStore{redisService: &RedisService{Client: client}}, nil
	if err != nil {
		fmt.Printf("error initializing test meta store: %s\n", err)
		os.Exit(1)
	}

	redisRedisMetaStore = store
	if err := redisRedisMetaStore.AddUser(testUser, testPass); err != nil {
		teardownRedisMetaStore()
		fmt.Printf("error adding test user to meta store: %s\n", err)
		os.Exit(1)
	}

	rv := &RequestVars{Authorization: testAuth, Oid: contentOid, Size: contentSize}
	if _, err := redisRedisMetaStore.Put(rv); err != nil {
		teardownRedisMetaStore()
		fmt.Printf("error seeding test meta store: %s\n", err)
		os.Exit(1)
	}
}

func teardownRedisMetaStore() {
	redisRedisMetaStore.redisService.Client.HDel(contentOid).Result()
	redisRedisMetaStore.redisService.Client.HDel("lfs-meta:project:oids::").Result()
	redisRedisMetaStore.redisService.Client.SRem(fmt.Sprintf(ProjectsHashName, "::"), ":").Result()
	redisRedisMetaStore.redisService.Client.SRem(ProjectsHashName, ":").Result()
	redisRedisMetaStore.redisService.Client.SRem(AllOidsHashName, "f97e1b2936a56511b3b6efc99011758e4700d60fb1674d31445d1ee40b663f24").Result()
	redisRedisMetaStore.redisService.Client.Del("f97e1b2936a56511b3b6efc99011758e4700d60fb1674d31445d1ee40b663f24").Result()
	redisRedisMetaStore.redisService.Client.Del("aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f").Result()
}
