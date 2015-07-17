package main

type RedisMetaStore struct {
	redisService *RedisService
}

func NewRedisMetaStore() (*RedisMetaStore, error) {
	return &RedisMetaStore{}, nil
}

func (self *RedisMetaStore) Put(v *RequestVars) (*MetaObject, error) {
	return &MetaObject{}, nil
}

func (self *RedisMetaStore) Get(v *RequestVars) (*MetaObject, error) {
	return &MetaObject{}, nil
}
func (self *RedisMetaStore) Close() {
	return
}
func (self *RedisMetaStore) DeleteUser(user string) error {
	return nil
}
func (self *RedisMetaStore) AddUser(user, pass string) error {
	return nil
}
func (self *RedisMetaStore) Users() ([]*MetaUser, error) {
	return []*MetaUser{}, nil
}
func (self *RedisMetaStore) Objects() ([]*MetaObject, error) {
	return []*MetaObject{}, nil
}
