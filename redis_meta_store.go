package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"strings"
)

type RedisMetaStore struct {
	redisService *RedisService
}

const OidHashName = "lfs-meta:project:oids"
const ProjectsHashName = "lfs-meta:projects"
const UsersHashName = "lfs-meta:users"
const AllOidsHashName = "lfs-meta:all:oids"
const PasswordKey = "password"
const UsernameKey = "username"
const SizeKey = "size"

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
	_, err := client.HSet(v.Oid, SizeKey, fmt.Sprintf("%d", v.Size)).Result()
	if err != nil {
		return nil, err
	}
	// add a record for the hash
	client.SAdd(AllOidsHashName, v.Oid)
	// Add the Oid to the project
	repo := fmt.Sprintf("%s:%s", v.User, v.Repo)
	if repo == "" {
		repo = "public"
	}
	_, err = client.SAdd(projectObjectKey(repo), v.Oid).Result()
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
		//		logger.Log(kv{"fn": "meta_store", "msg": "Unauthorized"})
		return nil, newAuthError()
	}

	client := self.redisService.Client
	oids, oid_err := client.SMembers(AllOidsHashName).Result()
	if oid_err != nil {
		//		logger.Log(kv{"fn": "meta_store", "msg": "Unable to find OID: " + oid_err.Error()})
		return nil, errObjectNotFound
	}
	var oid string
	if exists(v.Oid, oids) {
		oid = v.Oid
	}

	size, hg_err := client.HGet(oid, SizeKey).Int64()
	if hg_err != nil {
		//		logger.Log(kv{"fn": "meta_store", "msg": "Unable to find OID: " + v.Oid + " error " + hg_err.Error()})
		return nil, errObjectNotFound
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

func (self *RedisMetaStore) DeleteUser(user string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	client := self.redisService.Client
	// Delete the user records
	client.HDel(user, UsernameKey).Result()
	client.HDel(user, PasswordKey).Result()
	_, err := client.SRem(UsersHashName, user).Result()
	return err
}

func (self *RedisMetaStore) AddUser(user, pass string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	self.redisService.Client.HSet(user, UsernameKey, user).Result()
	// TODO: do something with the responses
	encryptedPass, err := encryptPass([]byte(pass))
	if err != nil {
		return err
	}
	self.redisService.Client.HSet(user, PasswordKey, encryptedPass).Result()
	self.redisService.Client.SAdd(UsersHashName, user).Result()
	return nil
}

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
		size, _ := client.HGet(oid, SizeKey).Int64()
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
			return true
		}
	}
	return false
}

func isErrNoRedisHash(err error) bool {
	type isNotThereError interface {
		errProjectNotFound() bool
	}
	if ae, ok := err.(isNotThereError); ok {
		return ae.errProjectNotFound()
	}
	return false
}

func isErrRedisObjectNotFound(err error) bool {
	type errObjectNotFound interface {
		errObjectNotFound() bool
	}
	if ae, ok := err.(errObjectNotFound); ok {
		return ae.errObjectNotFound()
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
	return "", errProjectNotFound
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
		logger.Log(kv{"fn": "redis_meta_store", "msg": "Unable to parse auth"})
		return false
	}
	user, password := cs[:i], cs[i+1:]
	if Config.Ldap.Enabled {
		return authenticateLdap(user, password)
	}

	mPass, err := self.redisService.Client.HGet(user, PasswordKey).Result()
	if err != nil {
		//		logger.Log(kv{"fn": "redis_meta_store", "msg": fmt.Sprintf("Auth error: %s", err.Error())})
		return false
	}
	match, err := checkPass([]byte(mPass), []byte(password))
	if err != nil {
		logger.Log(kv{"fn": "redis_meta_store", "msg": fmt.Sprintf("Decrypt error: %s", err.Error())})
	}
	return match
}

func (self *RedisMetaStore) Projects() ([]*MetaProject, error) {
	client := self.redisService.Client
	projects, _ := client.SMembers(ProjectsHashName).Result()
	project_list := make([]*MetaProject, 0)
	for _, project := range projects {
		oids, _ := self.findProjectOids(project)
		project_list = append(project_list, &MetaProject{Name: project, Oids: oids})
	}
	return project_list, nil
}
