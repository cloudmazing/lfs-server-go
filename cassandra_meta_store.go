package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"strings"
	"github.com/relops/cqlr"
)

type CassandraMetaStore struct {
	cassandraService *CassandraService
}

func NewCassandraMetaStore(cassandraService ...*CassandraService) (*CassandraMetaStore, error) {
	if len(cassandraService) == 0 {
		cassandraService = append(cassandraService, NewCassandraSession())
	}
	return &CassandraMetaStore{cassandraService: cassandraService[0]}, nil
}

func (self *CassandraMetaStore) Close() {
	defer self.cassandraService.Client.Close()
	return
}

func (self *CassandraMetaStore) createProject(project string) error {
	q := fmt.Sprintf("insert into projects (name) values('%s');", project)
	err := self.cassandraService.Client.Query(q).Exec()
	return err
}

func (self *CassandraMetaStore) addOidToProject(oid string, project string) error {
	q := fmt.Sprintf("update projects set oids = oids + {'%s'} where name = '%s';", oid, project)
	err := self.cassandraService.Client.Query(q).Exec()
	return err
}

func (self *CassandraMetaStore) createOid(oid string, size int64) error {
	q := fmt.Sprintf("insert into oids (oid, size) values ('%s', %d);", oid, size)
	return self.cassandraService.Client.Query(q).Exec()
}

func (self *CassandraMetaStore) removeOid(oid string) error {
	q := fmt.Sprintf("select project from projects where oids contains '%s';", oid)
	return self.cassandraService.Client.Query(q).Exec()
}

func (self *CassandraMetaStore) removeProject(v *RequestVars) error {
	q := fmt.Sprintf("delete from projects where name = '%s;", v.Repo)
	return self.cassandraService.Client.Query(q).Exec()
}

func (self *CassandraMetaStore) findProject(projectName string) (*MetaProject, error) {
	if projectName == "" {
		return nil, errProjectNotFound
	}
	q := self.cassandraService.Client.Query(`select * from projects where name = ?`, projectName)
	b := cqlr.BindQuery(q)
	var ct MetaProject
	b.Scan(&ct)
	if ct.Name == "" {
		return nil, errProjectNotFound
	}
	return &ct, nil
}

func (self *CassandraMetaStore) findOid(oid string) (*MetaObject, error) {
	q := fmt.Sprintf("select oid, size from oids where oid = '%s' limit 1;", oid)
	itr := self.cassandraService.Client.Query(q).Iter()
	defer itr.Close()
	var size int64
	var lOid string
	for itr.Scan(&lOid, &size) {
		if lOid == "" {
			return nil, errObjectNotFound
		}
		return &MetaObject{Oid: lOid, Size: size}, nil
	}
	return nil, errObjectNotFound
}

func (self *CassandraMetaStore) findAllOids() ([]*MetaObject, error) {
	itr := self.cassandraService.Client.Query("select oid, size from oids;").Iter()
	var oid string
	var size int64
	oid_list := make([]*MetaObject, 0)
	for itr.Scan(&oid, &size) {
		oid_list = append(oid_list, &MetaObject{Oid: oid, Size: size})
	}
	itr.Close()
	return oid_list, nil
}

func (self *CassandraMetaStore) findAllProjects() ([]*MetaProject, error) {
	itr := self.cassandraService.Client.Query("select name, oids from projects;").Iter()
	var oids []string
	var name string
	project_list := make([]*MetaProject, 0)
	for itr.Scan(&name, &oids) {
		project_list = append(project_list, &MetaProject{Name: name, Oids: oids})
	}
	itr.Close()
	return project_list, nil
}

func (self *CassandraMetaStore) Put(v *RequestVars) (*MetaObject, error) {
	if !self.authenticate(v.Authorization) {
		logger.Log(kv{"fn": "cassandra_meta_store", "msg": "Unauthorized"})
		return nil, newAuthError()
	}
	if meta, err := self.Get(v); err == nil {
		meta.Existing = true
		return meta, nil
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	meta := MetaObject{Oid: v.Oid, Size: v.Size, Existing: false}
	err := enc.Encode(meta)
	perror(self.createOid(v.Oid, v.Size))
	if v.Repo != "" {
		// find or create project
		_, ferr := self.findProject(v.Repo)
		if ferr != nil {
			// project does not exist, create it
			perror(self.createProject(v.Repo))
		}
		perror(self.addOidToProject(v.Oid, v.Repo))
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (self *CassandraMetaStore) Get(v *RequestVars) (*MetaObject, error) {
	if !self.authenticate(v.Authorization) {
		logger.Log(kv{"fn": "cassandra_meta_store", "msg": "Unauthorized"})
		return nil, newAuthError()
	}
	r, err := self.findOid(v.Oid)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	meta := MetaObject{Oid: r.Oid, Size: r.Size}
	err = enc.Encode(meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

/*
finds a user
Usage: FindUser("testuser")
*/
func (self *CassandraMetaStore) findUser(user string) (*MetaUser, error) {
	var mu MetaUser
	q := self.cassandraService.Client.Query(`select * from users where username = ?`, user)
	b := cqlr.BindQuery(q)
	b.Scan(&mu)
	if mu.Name == "" {
		return nil, errUserNotFound
	}
	return &mu, nil
}

func (self *CassandraMetaStore) AddUser(user, pass string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	_, uErr := self.findUser(user)
	// return nil if the user is already there
	if uErr == nil {
		return nil
	}
	encryptedPass, err := encryptPass([]byte(pass))
	if err != nil {
		return err
	}

	err = self.cassandraService.Client.Query("insert into users (username, password) values(?, ?);", user, encryptedPass).Exec()
	return err
}

func (self *CassandraMetaStore) DeleteUser(user string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	return self.cassandraService.Client.Query("delete from users where username = ?;", user).Exec()
}

func (self *CassandraMetaStore) Users() ([]*MetaUser, error) {
	if Config.Ldap.Enabled {
		return []*MetaUser{}, errNotImplemented
	}
	users := make([]*MetaUser, 0)
	itr := self.cassandraService.Client.Query("select username from users;").Iter()
	defer itr.Close()
	var username string
	for itr.Scan(&username) {
		users = append(users, &MetaUser{Name: username})
	}
	return users, nil
}

func (self *CassandraMetaStore) Objects() ([]*MetaObject, error) {
	ao, err := self.findAllOids()
	if err != nil {
		logger.Log(kv{"fn": "cassandra_meta_store", "msg": err.Error()})
	}
	return ao, err
}

func (self *CassandraMetaStore) authenticate(authorization string) bool {
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
		logger.Log(kv{"fn": "cassandra_meta_store.authenticate", "msg": err.Error()})
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
	mu, err := self.findUser(user)
	if err != nil {
		logger.Log(kv{"fn": "cassandra_meta_store", "msg": fmt.Sprintf("Auth error: %S", err.Error())})
		return false
	}

	match, err := checkPass([]byte(mu.Password), []byte(password))
	if err != nil {
		logger.Log(kv{"fn": "redis_meta_store", "msg": fmt.Sprintf("Decrypt error: %S", err.Error())})
	}
	return match
}

func (self *CassandraMetaStore) Projects() ([]*MetaProject, error) {
	ao, err := self.findAllProjects()
	if err != nil {
		logger.Log(kv{"fn": "cassandra_meta_store", "msg": err.Error()})
	}
	return ao, err
}
