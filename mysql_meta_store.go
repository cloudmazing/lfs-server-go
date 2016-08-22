package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/gob"
	// "encoding/json"
	"fmt"
	// "gopkg.in/gorp.v1"
	"strings"
)

/*
MySQLMetaStore struct.
*/
type MySQLMetaStore struct {
	mysqlService *MySQLService
	client       *sql.DB
}

/*
NewMySQLMetaStore (method update the MySQLMetaStore struct)
*/
func NewMySQLMetaStore(mysqlService ...*MySQLService) (*MySQLMetaStore, error) {
	if len(mysqlService) == 0 {
		mysqlService = append(mysqlService, NewMySQLSession())
	}

	mysql := mysqlService[0]
	return &MySQLMetaStore{mysqlService: mysql, client: mysql.Client}, nil
}

/*
Close (method close mysql connection)
*/
func (m *MySQLMetaStore) Close() {
	defer m.client.Close()
	return
}

/*
Oid finder - returns a []*MetaObject
*/
func (m *MySQLMetaStore) findAllOids() ([]*MetaObject, error) {
	rows, _ := m.client.Query("select oid, size from oids;")

	var oid string
	var size int64

	var oidList []*MetaObject

	for rows.Next() {
		err := rows.Scan(&oid, &size)
		if err != nil {
			logger.Log(kv{"fn": "findProject", "msg": err})
		}
		oidList = append(oidList, &MetaObject{Oid: oid, Size: size})
	}

	defer rows.Close()
	return oidList, nil
}

/*
OID Maps
*/
func (m *MySQLMetaStore) mapOid(id int64) ([]string, error) {
	rows, err := m.client.Query("select oid from oid_maps where projectID = ?", id)

	var (
		oid string
	)
	var oidList []string

	if err != nil {
		logger.Log(kv{"fn": "findProject", "msg": fmt.Sprintf("Oid not found %s", err)})
		return nil, err
	}

	for rows.Next() {
		err := rows.Scan(&oid)
		if err != nil {
			logger.Log(kv{"fn": "findProject", "msg": err})
			return nil, err
		}
		oidList = append(oidList, oid)
	}
	defer rows.Close()

	return oidList, nil
}

/*
Project finder - returns a []*MetaProject
*/
func (m *MySQLMetaStore) findAllProjects() ([]*MetaProject, error) {
	count, err := m.client.Query("select count(*) as count from projects")
	var c int
	for count.Next() {
		err = count.Scan(&c)

		if err != nil || c == 0 {
			return nil, nil
		}
	}

	rows, err := m.client.Query("select id, name from projects")

	var name string
	var id int64

	var projectList []*MetaProject

	for rows.Next() {
		err = rows.Scan(&id, &name)

		if err != nil {
			logger.Log(kv{"fn": "findProject", "msg": err})
		}

		oid, _ := m.mapOid(id)
		projectList = append(projectList, &MetaProject{Name: name, Oids: oid})
	}

	defer rows.Close()

	if len(projectList) == 0 {
		return nil, errProjectNotFound
	}
	return projectList, nil
}

// Create project
func (m *MySQLMetaStore) createProject(name string) error {
	_, err := m.client.Exec("insert into projects (name) values (?)", name)
	if err != nil {
		logger.Log(kv{"fn": "createProject", "msg": fmt.Sprintf("MySQL insert query failed with error %s", err)})
		return err
	}
	return nil
}

// Create oid
func (m *MySQLMetaStore) createOid(oid string, size int64) error {
	_, err := m.client.Exec("insert into oids (oid, size) values (?, ?)", oid, size)

	if err != nil {
		logger.Log(kv{"fn": "createOid", "msg": fmt.Sprintf("MySQL insert query failed with error %s", err)})
		return nil
	}
	return nil
}

// Find project
func (m *MySQLMetaStore) findProject(projectName string) (*MetaProject, error) {
	if projectName == "" {
		return nil, errProjectNotFound
	}

	var project MetaProject
	var (
		id  int64
		oid string
	)

	// Get projectname and its ids
	err := m.client.QueryRow("select * from projects where name = ?", projectName).Scan(&id, &project.Name)

	if err != nil {
		logger.Log(kv{"fn": "findProject", "msg": fmt.Sprintf("Project not found %s", err)})
		return nil, err
	}

	// get oids
	rows, err := m.client.Query("select oid from oid_maps where projectID = ?", id)

	if err != nil {
		logger.Log(kv{"fn": "findProject", "msg": fmt.Sprintf("Oid not found %s", err)})
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&oid)
		if err != nil {
			logger.Log(kv{"fn": "findProject", "msg": err})
		}
		project.Oids = append(project.Oids, oid)
	}

	err = rows.Err()
	if err != nil {
		logger.Log(kv{"fn": "findProject", "msg": fmt.Sprintf("Error while looping through rows %s", err)})
	}

	if project.Name == "" {
		return nil, errProjectNotFound
	}
	logger.Log(kv{"fn": "findProject", "msg": fmt.Sprintf("Project %s", &project)})
	return &project, nil
}

// Add oid to project
func (m *MySQLMetaStore) addOidToProject(oid string, project string) error {
	// Cannot bind on collections
	var (
		id   int64
		name string
	)
	err := m.client.QueryRow("select * from projects where name = ?", project).Scan(&id, &name)
	_, err = m.client.Exec("insert into oid_maps (oid, projectID) values (?, ?)", oid, id)
	logger.Log(kv{"fn": "addOidToProject", "msg": err})
	return err
}

// Find oid
func (m *MySQLMetaStore) findOid(oid string) (*MetaObject, error) {
	var mo MetaObject
	err := m.client.QueryRow("select * from oids where oid = ?", oid).Scan(&mo.Oid, &mo.Size)

	if err != nil {
		return nil, err
	}

	if mo.Oid == "" {
		return nil, errObjectNotFound
	}

	return &mo, nil
}

/*
Put (HTTP PUT handler)
create OID and map to projects
*/
func (m *MySQLMetaStore) Put(v *RequestVars) (*MetaObject, error) {
	if !m.authenticate(v.Authorization) {
		logger.Log(kv{"fn": "mysql_meta_store", "msg": "Unauthorized"})
		return nil, newAuthError()
	}

	if meta, err := m.Get(v); err == nil {
		meta.Existing = true
		return meta, nil
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	meta := MetaObject{Oid: v.Oid, Size: v.Size, Existing: false}
	err := enc.Encode(meta)
	perror(m.createOid(v.Oid, v.Size))
	if v.Repo != "" {
		// find or create project
		_, ferr := m.findProject(v.Repo)
		if ferr != nil {
			// project does not exist, create it
			return nil, errProjectNotFound
		}
		perror(m.addOidToProject(v.Oid, v.Repo))
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

/*
Get (HTTP Get handler)
*/
func (m *MySQLMetaStore) Get(v *RequestVars) (*MetaObject, error) {
	if !m.authenticate(v.Authorization) {
		logger.Log(kv{"fn": "mysql_meta_store", "msg": "Unauthorized"})
		return nil, newAuthError()
	}
	r, err := m.findOid(v.Oid)

	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	logger.Log(kv{"fn": "Get", "msg": r})
	meta := MetaObject{Oid: r.Oid, Size: r.Size}
	err = enc.Encode(meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

/*
AddUser (Add a new user)
Not implemented in mysql_meta_store
*/
func (m *MySQLMetaStore) AddUser(user, pass string) error {
	return errNotImplemented
}

/*
AddProject (Add a new project)
*/
func (m *MySQLMetaStore) AddProject(name string) error {
	err := m.createProject(name)
	return err
}

/*
DeleteUser (Delete a user)
Not implemented
*/
func (m *MySQLMetaStore) DeleteUser(user string) error {
	return errNotImplemented
}

/*
Users (get list of users)
Not implemented
*/
func (m *MySQLMetaStore) Users() ([]*MetaUser, error) {
	return []*MetaUser{}, errNotImplemented
}

/*
Objects (get all oids)
return meta object
*/
func (m *MySQLMetaStore) Objects() ([]*MetaObject, error) {
	ao, err := m.findAllOids()
	if err != nil {
		logger.Log(kv{"fn": "mysql_meta_store", "msg": err.Error()})
	}
	return ao, err
}

/*
Projects (get all projects)
return meta project object
*/
func (m *MySQLMetaStore) Projects() ([]*MetaProject, error) {
	ao, err := m.findAllProjects()
	if err != nil {
		logger.Log(kv{"fn": "mysql_meta_store", "msg": err.Error()})
	}
	return ao, err
}

/*
Auth routine.  Requires an auth string like
"Basic YWRtaW46YWRtaW4="
*/
func (m *MySQLMetaStore) authenticate(authorization string) bool {
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
		logger.Log(kv{"fn": "mysql_meta_store.authenticate", "msg": err.Error()})
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

	logger.Log(kv{"fn": "mysql_meta_store", "msg": "Authentication failed, please make sure LDAP is set to true"})
	return false

}
