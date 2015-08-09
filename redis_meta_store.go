package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
)

type RedisMetaStore struct {
	redisService *RedisService
}

var (
	errNoRedisProject      = errors.New("Project not found in redis")
	errRedisObjectNotFound = errors.New("Object not found in redis")
)

const OidHashName = "lfs-meta:project:oids"
const ProjectsHashName = "lfs-meta:projects"
const UsersHashName = "lfs-meta:users"
const AllOidsHashName = "lfs-meta:all:oids"
const PasswordKey = "password"
const UsernameKey = "username"

func NewRedisMetaStore(client ...*RedisService) (*RedisMetaStore, error) {
	if len(client) == 0 {
		client = append(client, NewRedisClient())
	}
	return &RedisMetaStore{redisService: client[0]}, nil
}

func (self *RedisMetaStore) Put(v *RequestVars) (*MetaObject, error) {
	if !self.authenticate(v.Authorization) {
		return nil, newAuthError()
	}

	// Check if the oid exists first
	if meta, err := self.Get(v); err == nil {
		if !isErrNoRedisHash(err) || !isErrRedisObjectNotFound(err) {
			meta.Existing = true
			return meta, nil
		}
	}

	client := self.redisService.Client
	// Create the project
	_, aErr := client.SAdd(ProjectsHashName, fmt.Sprintf("%s:%s", v.User, v.Repo)).Result()
	if aErr != nil {
		return nil, aErr
	}

	// Add the Oid Record
	_, err := client.HSet(v.Oid, "size", fmt.Sprintf("%d", v.Size)).Result()
	if err != nil {
		return nil, err
	}
	// add a record for the hash
	client.SAdd(AllOidsHashName, v.Oid)
	// Add the Oid to the project
	_, err = client.SAdd(projectObjectKey(v.Repo), v.Oid).Result()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	meta := MetaObject{Oid: v.Oid, Size: v.Size}
	err = enc.Encode(meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (self *RedisMetaStore) Get(v *RequestVars) (*MetaObject, error) {

	if !self.authenticate(v.Authorization) {
		logger.Log(kv{"fn": "meta_store", "msg": "Unauthorized"})
		return nil, newAuthError()
	}

	client := self.redisService.Client
	oids, oid_err := client.SMembers(AllOidsHashName).Result()
	if oid_err != nil {
		logger.Log(kv{"fn": "meta_store", "msg": "Unable to find OID: " + oid_err.Error()})
		return nil, oid_err
	}
	var oid string
	if exists(v.Oid, oids) {
		oid = v.Oid
	}

	size, hg_err := client.HGet(oid, "size").Int64()
	if hg_err != nil {
		logger.Log(kv{"fn": "meta_store", "msg": "Unable to find OID: " + v.Oid})
		return nil, hg_err
	}

	meta := &MetaObject{Oid: v.Oid, Size: size, Existing: true}
	// if it exists then we return it
	dec := gob.NewDecoder(bytes.NewBuffer([]byte(v.Oid)))
	// put the meta object into the decoder
	dec.Decode(&meta)
	return meta, nil
}

func (self *RedisMetaStore) Close() {
	defer self.redisService.Client.Close()
	return
}

// TODO: Should probably not be used when using ldap
func (self *RedisMetaStore) DeleteUser(user string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	client := self.redisService.Client
	// Delete the user records
	client.HDel(user, "username").Result()
	client.HDel(user, "password").Result()
	_, err := client.SRem(UsersHashName, user).Result()
	return err
}

// TODO: Should probably not be used when using ldap
func (self *RedisMetaStore) AddUser(user, pass string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	self.redisService.Client.HSet(user, UsernameKey, user).Result()
	// TODO: do something with the responses
	self.redisService.Client.HSet(user, PasswordKey, pass).Result()
	self.redisService.Client.SAdd(UsersHashName, user).Result()
	return nil
}

// TODO: Should probably not be used when using ldap
func (self *RedisMetaStore) Users() ([]*MetaUser, error) {
	if Config.Ldap.Enabled {
		return []*MetaUser{}, errNotImplemented
	}
	var mus []*MetaUser
	users, _ := self.redisService.Client.SMembers(UsersHashName).Result()
	for _, user := range users {
		mus = append(mus, &MetaUser{Name: string(user)})
	}
	return mus, nil
}

func (self *RedisMetaStore) Objects() ([]*MetaObject, error) {
	client := self.redisService.Client
	oids, _ := client.SMembers(AllOidsHashName).Result()
	mus := make([]*MetaObject, 0)
	for _, oid := range oids {
		size, _ := client.HGet(oid, "size").Int64()
		mu := &MetaObject{Oid: oid, Size: size}
		mus = append(mus, mu)
	}
	return mus, nil
}

// Helpers
// Check the oid list for a given oid
func exists(s string, l []string) bool {
	for _, t := range l {
		if s == t {
			fmt.Sprintf("Found %s in %v\n", s, l)
			return true
		}
	}
	return false
}

func isErrNoRedisHash(err error) bool {
	type isNotThereError interface {
		errNoRedisProject() bool
	}
	if ae, ok := err.(isNotThereError); ok {
		return ae.errNoRedisProject()
	}
	return false
}

func isErrRedisObjectNotFound(err error) bool {
	type errRedisObjectNotFound interface {
		errRedisObjectNotFound() bool
	}
	if ae, ok := err.(errRedisObjectNotFound); ok {
		return ae.errRedisObjectNotFound()
	}
	return false
}

func projectObjectKey(repo string) string {
	return fmt.Sprintf("%s:%s", OidHashName, repo)
}

func (self *RedisMetaStore) findProject(v *RequestVars) (string, error) {
	client := self.redisService.Client
	projects, perr := client.SMembers(ProjectsHashName).Result()
	perror(perr)
	for _, p := range projects {
		if p == v.Repo {
			return p, nil
		}
	}
	return "", errNoRedisProject
}

func (self *RedisMetaStore) findProjectOids(project string) ([]string, error) {
	client := self.redisService.Client
	return client.SMembers(projectObjectKey(project)).Result()
}

// authenticate uses the authorization string to determine whether
// or not to proceed. This server assumes an HTTP Basic auth format.
func (self *RedisMetaStore) authenticate(authorization string) bool {
	if Config.IsPublic() {
		return true
	}

	if authorization == "" {
		return false
	}

	if !strings.HasPrefix(authorization, "Basic ") {
		return false
	}

	c, err := base64.URLEncoding.DecodeString(strings.TrimPrefix(authorization, "Basic "))
	if err != nil {
		logger.Log(kv{"fn": "redis_meta_store.authenticate", "msg": err.Error()})
		return false
	}
	cs := string(c)
	i := strings.IndexByte(cs, ':')
	if i < 0 {
		return false
	}
	user, password := cs[:i], cs[i+1:]
	if Config.Ldap.Enabled {
		return authenticateLdap(user, password)
	}

	mPass, err := self.redisService.Client.HGet(user, "password").Result()
	if err != nil {
		logger.Log(kv{"fn": "redis_meta_store", "msg": fmt.Sprintf("Auth error: %S", err.Error())})
		return false
	}
	if password != "" && string(mPass) == string(password) {
		return true
	}
	return false
}

func (self *RedisMetaStore) Projects() ([]*MetaProject, error) {
	return []*MetaProject{}, nil
}
