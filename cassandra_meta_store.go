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
	counter := make(map[string]interface{}, 1)
	self.client.Query("select count(*) as count from projects where name = ?", project).MapScan(counter)
	if counter["count"].(int64) > 0 {
		// already there
		return nil
	}
	err := self.client.Query("insert into projects (name) values(?)", project).Exec()
	return err
}

func (self *CassandraMetaStore) addOidToProject(oid string, project string) error {
	// Cannot bind on collections
	q := fmt.Sprintf("update projects set oids = oids + {'%s'} where name = '%s'", oid, project)
	err := self.client.Query(q).Exec()
	return err
}

func (self *CassandraMetaStore) createOid(oid string, size int64) error {
	return self.client.Query("insert into oids (oid, size) values (?, ?)", oid, size).Exec()
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
	q := fmt.Sprintf("update projects set oids = oids - {'%s'} where name = '%s'", oid, project)
	return self.client.Query(q).Exec()
}

func (self *CassandraMetaStore) removeProject(projectName string) error {
	return self.client.Query("delete from projects where name = ?", projectName).Exec()
}

func (self *CassandraMetaStore) findProject(projectName string) (*MetaProject, error) {
	if projectName == "" {
		return nil, errProjectNotFound
	}
	q := self.client.Query("select * from projects where name = ?", projectName)
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
	q := self.client.Query("select oid, size from oids where oid = ? limit 1", oid)
	b := cqlr.BindQuery(q)
	var mo MetaObject
	b.Scan(&mo)
	defer b.Close()
	if mo.Oid == "" {
		return nil, errObjectNotFound
	}
	return &mo, nil
}

/*
Oid finder - returns a []*MetaObject
*/
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

/*
Project finder - returns a []*MetaProject
*/
func (self *CassandraMetaStore) findAllProjects() ([]*MetaProject, error) {
	itr := self.cassandraService.Client.Query("select name, oids from projects;").Iter()
	var oids []string
	var name string
	project_list := make([]*MetaProject, 0)
	//	var project_list []*MetaProject
	for itr.Scan(&name, &oids) {
		project_list = append(project_list, &MetaProject{Name: name, Oids: oids})
	}
	itr.Close()
	if len(project_list) == 0 {
		return nil, errProjectNotFound
	}
	return project_list, nil
}

/*
provides access to the put crud operation
Usage:
Put(&RequestVars{
Oid: "oid string",
Size: 64,
User: "admin",
Password: "admin",
epo: "my-repo",
Authorization: "Basic YWRtaW46YWRtaW4=",
})

*/
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

/*
Provides access to the get crud operation
Usage:
Get(&RequestVars{
Oid: "oid string",
Size: 64,
User: "admin",
Password: "admin",
epo: "my-repo",
Authorization: "Basic YWRtaW46YWRtaW4=",
})
*/
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
	q := self.client.Query("select * from users where username = ?", user)
	b := cqlr.BindQuery(q)
	b.Scan(&mu)
	if mu.Name == "" {
		return nil, errUserNotFound
	}
	return &mu, nil
}

/*
Adds a user to the system, only for use when not using ldap
*/
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

	return self.client.Query("insert into users (username, password) values(?, ?)", user, encryptedPass).Exec()
}

/*
Removes a user from the system, only for use when not using ldap
Usage: DeleteUser("testuser")
*/
func (self *CassandraMetaStore) DeleteUser(user string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	return self.client.Query("delete from users where username = ?", user).Exec()
}

/*
returns all users, only for use when not using ldap
*/
func (self *CassandraMetaStore) Users() ([]*MetaUser, error) {
	if Config.Ldap.Enabled {
		return []*MetaUser{}, errNotImplemented
	}
	var mu MetaUser
	users := make([]*MetaUser, 0)
	q := self.client.Query("select username from users")
	b := cqlr.BindQuery(q)
	for b.Scan(&mu) {
		users = append(users, &mu)
	}
	return users, nil
}

/*
returns all Oids
*/
func (self *CassandraMetaStore) Objects() ([]*MetaObject, error) {
	ao, err := self.findAllOids()
	if err != nil {
		logger.Log(kv{"fn": "cassandra_meta_store", "msg": err.Error()})
	}
	return ao, err
}

/*
Returns a []*MetaProject
*/
func (self *CassandraMetaStore) Projects() ([]*MetaProject, error) {
	ao, err := self.findAllProjects()
	if err != nil {
		logger.Log(kv{"fn": "cassandra_meta_store", "msg": err.Error()})
	}
	return ao, err
}

/*
AddProject (create a new project using POST)
Only implemented on MySQL meta store
*/
func (self *CassandraMetaStore) AddProject(name string) error {
	return errNotImplemented
}

/*
Auth routine.  Requires an auth string like
"Basic YWRtaW46YWRtaW4="
*/
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
		logger.Log(kv{"fn": "cassandra_meta_store", "msg": fmt.Sprintf("Auth error: %s", err.Error())})
		return false
	}

	match, err := checkPass([]byte(mu.Password), []byte(password))
	if err != nil {
		logger.Log(kv{"fn": "cassandra_meta_store", "msg": fmt.Sprintf("Decrypt error: %s", err.Error())})
	}
	return match
}
