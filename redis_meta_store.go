package main
import (
	"encoding/gob"
	"bytes"
	"strings"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
)

type RedisMetaStore struct {
	redisService *RedisService
	KeyHashName  string
}

var (
	errNoRedisHash = errors.New("Redis hash not found in redis")
	errRedisObjectNotFound = errors.New("Object not found in redis")
)


const OidHashName = "lfs-meta-oids"
const UsersHashName = "lfs-meta-users"
const AllOidsHashName = "all:oids"
const PasswordKey = "password"
const UsernameKey = "username"

func NewRedisMetaStore() (*RedisMetaStore, error) {
	return &RedisMetaStore{KeyHashName: OidHashName, redisService: NewRedisClient()}, nil
}

func projectObjectKey(repo string) (string) {
	return fmt.Sprintf("%s:%s", OidHashName, repo)
}

func size_i(size string) (int64) {
	r, _ := strconv.ParseInt(fmt.Sprintf("%s", size), 0, 0)
	return r
}
func (self *RedisMetaStore) Put(v *RequestVars) (*MetaObject, error) {
	if !self.authenticate(v.Authorization) {
		return nil, newAuthError()
	}

	// Check if the oid exists first
	if meta, err := self.Get(v); err == nil {
		meta.Existing = true
		return meta, nil
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	meta := MetaObject{Oid: v.Oid, Size: v.Size}
	err := enc.Encode(meta)
	if err != nil {
		return nil, err
	}
	client := self.redisService.Client
	// Add the Oid Record
	_, err = client.HMSet(meta.Oid, "size", fmt.Sprintf("%d", meta.Size)).Result()
	if err != nil {
		return nil, err
	}
	// Add the Oid to the project
	_, err = client.SAdd(projectObjectKey(v.Repo), meta.Oid).Result()

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
	// first we check to see if the object is a member of the project
	project_oids, err := client.SMembers(projectObjectKey(v.Repo)).Result()
	if project_oids == nil {
		logger.Log(kv{"fn": "meta_store", "msg": errRedisObjectNotFound.Error()})
		return nil, errNoRedisHash
	}
	if err != nil {
		logger.Log(kv{"fn": "meta_store", "msg": err.Error()})
		return nil, err
	}
	// response into slices, find if one exists
	oid_exists := func() bool {
		for _, l_oid := range project_oids {
			if l_oid == v.Oid {
				return true
			}
		}
		return false
	}()
	size, _ := client.HGet(v.Oid, "size").Result()
	meta := &MetaObject{Oid: v.Oid, Size: size_i(size), Existing: true}
	// if it exists then we return it
	if oid_exists {
		dec := gob.NewDecoder(bytes.NewBuffer([]byte(v.Oid)))
		// put the meta object into the decoder
		dec.Decode(&meta)
	} else {
		logger.Log(kv{"fn": "meta_store", "msg": errRedisObjectNotFound.Error()})
		return nil, errObjectNotFound
	}
	return meta, nil
}

func (self *RedisMetaStore) Close() {
	return
}

// TODO: Should probably not be used when using ldap
func (self *RedisMetaStore) DeleteUser(user string) error {
	client := self.redisService.Client
	// Delete the user records
	client.HDel(user, "username").Result()
	client.HDel(user, "password").Result()
	_, err := client.SRem(UsersHashName, user).Result()
	return err
}

// TODO: Should probably not be used when using ldap
func (self *RedisMetaStore) AddUser(user, pass string) error {
	self.redisService.Client.HSet(user, UsernameKey, user).Result()
	// TODO: do something with the responses
	self.redisService.Client.HSet(user, PasswordKey, pass).Result()
	self.redisService.Client.SAdd(UsersHashName, user).Result()
	return nil
}
// TODO: Should probably not be used when using ldap
func (self *RedisMetaStore) Users() ([]*MetaUser, error) {
	users, _ := self.redisService.Client.SMembers(UsersHashName).Result()
	fmt.Println(users)
	mus := make([]*MetaUser, len(users))
	for _, user := range users {
		fmt.Println(user)
		mu := &MetaUser{Name: user}
		_ = append(mus, mu)
	}
	return mus, nil
}
func (self *RedisMetaStore) Objects() ([]*MetaObject, error) {
	client := self.redisService.Client
	members, _ := client.SMembers(AllOidsHashName).Result()
	mus := make([]*MetaObject, len(members))
	for _, oid := range mus {
		client.HGet(oid.Oid, "size").Result()
		mu := &MetaObject{Oid: oid.Oid, Size: oid.Size, Existing: true}
		_ = append(mus, mu)
	}
	return mus, nil
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

	c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authorization, "Basic "))
	if err != nil {
		return false
	}
	cs := string(c)
	i := strings.IndexByte(cs, ':')
	if i < 0 {
		return false
	}
	user, password := cs[:i], cs[i+1:]
	return LdapBind(user, password)
}

