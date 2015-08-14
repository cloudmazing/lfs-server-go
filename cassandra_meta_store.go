package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
	"strings"
)

type CassandraMetaStore struct {
	cassandraService *CassandraService
	client           *gocql.Session
}

func NewCassandraMetaStore(cassandraService ...*CassandraService) (*CassandraMetaStore, error) {
	if len(cassandraService) == 0 {
		cassandraService = append(cassandraService, NewCassandraSession())
	}
	cs := cassandraService[0]
	return &CassandraMetaStore{cassandraService: cs, client: cs.Client}, nil
}

func (self *CassandraMetaStore) Close() {
	defer self.client.Close()
	return
}

func (self *CassandraMetaStore) createProject(project string) error {
	err := self.client.Query(`insert into projects (name) values(?)`, project).Exec()
	return err
}

func (self *CassandraMetaStore) addOidToProject(oid string, project string) error {
	err := self.client.Query(`update projects set oids = oids + {?} where name = ?`, oid, project).Exec()
	return err
}

func (self *CassandraMetaStore) createOid(oid string, size int64) error {
	return self.client.Query(`insert into oids (oid, size) values (?, ?)`, oid, size).Exec()
}

func (self *CassandraMetaStore) removeOid(oid string) error {
	/*
		Oids are shared amongst projects, so this will need to find out the following:
		1. What projects (if any) have the requested OID.
		2. If other projects are still using the OID, then do not delete it from the main OID listing
	*/
	//	return self.client.Query("update projects set oids = oids - {?} where oids contains ?", oid).Exec()
	return self.client.Query("delete from oids where oid = ?", oid).Exec()
}

func (self *CassandraMetaStore) removeOidFromProject(oid, project string) error {
	/*
		Oids are shared amongst projects, so this will need to find out the following:
		1. What projects (if any) have the requested OID.
		2. If other projects are still using the OID, then do not delete it from the main OID listing
	*/
	return self.client.Query("update projects set oids = oids - {?} where oids contains ?", oid).Exec()
}

func (self *CassandraMetaStore) removeProject(projectName string) error {
	return self.client.Query(`delete from projects where name = ?`, projectName).Exec()
}

func (self *CassandraMetaStore) findProject(projectName string) (*MetaProject, error) {
	if projectName == "" {
		return nil, errProjectNotFound
	}
	q := self.client.Query(`select * from projects where name = ?`, projectName)
	b := cqlr.BindQuery(q)
	var ct MetaProject
	b.Scan(&ct)
	defer b.Close()
	if ct.Name == "" {
		return nil, errProjectNotFound
	}
	return &ct, nil
}

func (self *CassandraMetaStore) findOid(oid string) (*MetaObject, error) {
	q := self.client.Query(`select oid, size from oids where oid = ? limit 1`, oid)
	b := cqlr.BindQuery(q)
	var mo MetaObject
	b.Scan(&mo)
	defer b.Close()
	if mo.Oid == "" {
		return nil, errObjectNotFound
	}
	return &mo, nil
}

func (self *CassandraMetaStore) findAllOids() ([]*MetaObject, error) {
	q := self.client.Query(`select oid, size from oids`)
	b := cqlr.BindQuery(q)
	mos := make([]*MetaObject, 0)
	var mo MetaObject
	for b.Scan(&mo) {
		mos = append(mos, &mo)
	}
	defer b.Close()
	return mos, nil
}

func (self *CassandraMetaStore) findAllProjects() ([]*MetaProject, error) {
	q := self.client.Query(`select name, oids from project`)
	b := cqlr.BindQuery(q)
	project_list := make([]*MetaProject, 0)
	var mp MetaProject
	for b.Scan(&mp) {
		project_list = append(project_list, &mp)
	}
	defer b.Close()
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
	q := self.client.Query(`select * from users where username = ?`, user)
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

	return self.client.Query(`insert into users (username, password) values(?, ?)`, user, encryptedPass).Exec()
}

func (self *CassandraMetaStore) DeleteUser(user string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	return self.client.Query(`delete from users where username = ?`, user).Exec()
}

func (self *CassandraMetaStore) Users() ([]*MetaUser, error) {
	if Config.Ldap.Enabled {
		return []*MetaUser{}, errNotImplemented
	}
	users := make([]*MetaUser, 0)
	itr := self.client.Query(`select username from users`).Iter()
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

func (self *CassandraMetaStore) Projects() ([]*MetaProject, error) {
	ao, err := self.findAllProjects()
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
