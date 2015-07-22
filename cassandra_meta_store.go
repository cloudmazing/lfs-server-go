package main
import (
	"fmt"
	"encoding/gob"
	"bytes"
)

type CassandraMetaStore struct {
	cassandraService *CassandraService
}

func NewCassandraMetaStore() (*CassandraMetaStore, error) {
	return &CassandraMetaStore{cassandraService: NewCassandraSession()}, nil
}

func (self *CassandraMetaStore) Close() {
	return
}

func (self *CassandraMetaStore) createProject(project string) (error) {
	q := fmt.Sprintf("insert into projects (name) values('%s');", project)
	err := self.cassandraService.Client.Query(q).Exec()
	return err
}

func (self *CassandraMetaStore) addOidToProject(oid string, project string) (error) {
	q := fmt.Sprintf("update projects set oids = oids + {'%s'} where name = '%s';", oid, project)
	err := self.cassandraService.Client.Query(q).Exec()
	return err
}

func (self *CassandraMetaStore) createOid(oid string, size int64) (error) {
	q := fmt.Sprintf("insert into oids (oid, size) values ('%s', %d);", oid, size)
	return self.cassandraService.Client.Query(q).Exec()
}

func (self *CassandraMetaStore) removeOid(oid string) (error) {
	q := fmt.Sprintf("select project from projects where oids contains '%s';", oid)
	return self.cassandraService.Client.Query(q).Exec()
}

func (self *CassandraMetaStore) removeProject(v *RequestVars) (error) {
	q := fmt.Sprintf("delete from projects where name = '%s;", v.Repo)
	return self.cassandraService.Client.Query(q).Exec()
}

func (self *CassandraMetaStore) findProject(projectName string) (string, error) {
	if projectName == "" {
		return "", errProjectNotFound
	}
	q := fmt.Sprintf("select name from projects where name = '%s' limit 1;", projectName)
	itr := self.cassandraService.Client.Query(q).Iter()
	defer itr.Close()
	var project string
	for itr.Scan(&project) {
		if project == "" {return "", errProjectNotFound}
		return project, nil
	}
	return "", errProjectNotFound
}

func (self *CassandraMetaStore) findOid(oid string) (*MetaObject, error) {
	q := fmt.Sprintf("select oid, size from oids where oid = '%s' limit 1;", oid)
	itr := self.cassandraService.Client.Query(q).Iter()
	defer itr.Close()
	var size int64
	var lOid string
	for itr.Scan(&lOid, &size) {
		if lOid == "" {return nil, errObjectNotFound}
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

func (self *CassandraMetaStore) Put(v *RequestVars) (*MetaObject, error) {
	if !authenticate(v.Authorization) {
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
	if !authenticate(v.Authorization) {
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

// TODO: Skip if using ldap
func (self *CassandraMetaStore) AddUser(user, pass string) error {
	var _user string
	itr := self.cassandraService.Client.Query("select username from users where username = ? limit 1;",user).Iter()
	defer itr.Close()
	for itr.Scan(&_user) {
		if len(_user) > 0 {return nil}
	}
	err := self.cassandraService.Client.Query("insert into users (username, password) values(?, ?);", user, pass).Exec()
	return err
}

// TODO: Skip if using ldap
func (self *CassandraMetaStore) DeleteUser(user string) error {
	return self.cassandraService.Client.Query("delete from users where username = ?;", user).Exec()
}

// TODO: Skip if using ldap
func (self *CassandraMetaStore) Users() ([]*MetaUser, error) {
	users := make([]*MetaUser,0)
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
		fmt.Println("ERROR:", err)
	}
	fmt.Println("OIDS", ao)
	fmt.Println("OIDS length", len(ao))
	return ao,err
}



