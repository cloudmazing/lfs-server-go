package main

import (
	"errors"
	"fmt"
	"testing"
)

var (
	metaStoreTestMySQL *MySQLMetaStore
)

func TestMySQLConfiguration(t *testing.T) {
	Config.MySQL = &MySQLConfig{
		Enabled:  true,
		Host:     "127.0.0.1:3306",
		Database: "lfs_server_go_test",
	}

	mysqlStore, err := NewMySQLMetaStore()
	if mysqlStore != nil {
		t.Errorf("expected MySQL configration validation error, got : %s", err)
	}
}

func TestMySQLAddProjects(t *testing.T) {
	serr := setupMySQLMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}

	err := metaStoreTestMySQL.AddProject(testRepo)

	if err != nil {
		metaStoreTestMySQL.Close()
		t.Errorf("expected AddProject to succeed, got : %s", err)
	}
}

func TestMySQLPutWithAuth(t *testing.T) {

	meta, err := metaStoreTestMySQL.Put(&RequestVars{Authorization: testAuth, Oid: contentOid, Size: 42, Repo: testRepo})
	if err != nil {
		metaStoreTestMySQL.Close()
		t.Errorf("expected put to succeed, got : %s", err)
	}

	if meta.Existing {
		metaStoreTestMySQL.Close()
		t.Errorf("expected meta to not have existed")
	}

	meta, err = metaStoreTestMySQL.Get(&RequestVars{Authorization: testAuth, Oid: contentOid})
	if err != nil {
		metaStoreTestMySQL.Close()
		t.Errorf("expected to be able to retreive new put, got : %s", err)
	}

	if meta.Oid != contentOid {
		metaStoreTestMySQL.Close()
		t.Errorf("expected oids to match, got: %s", meta.Oid)
	}

	if meta.Size != 42 {
		metaStoreTestMySQL.Close()
		t.Errorf("expected sizes to match, got: %d", meta.Size)
	}
}

func TestMySQLPutWithoutAuth(t *testing.T) {

	_, err := metaStoreTestMySQL.Put(&RequestVars{Authorization: badAuth, User: testUser, Oid: contentOid, Size: 42, Repo: testRepo})
	if !isAuthError(err) {
		metaStoreTestMySQL.Close()
		t.Errorf("expected auth error, got: %s", err)
	}

}

func TestMySQLGetWithAuth(t *testing.T) {

	metaFail, err := metaStoreTestMySQL.Get(&RequestVars{Authorization: testAuth, Oid: noAuthOid})
	if err == nil {
		metaStoreTestMySQL.Close()
		t.Fatalf("Error Should not have access to OID: %s", metaFail.Oid)
	}

	meta, err := metaStoreTestMySQL.Get(&RequestVars{Authorization: testAuth, Oid: contentOid})
	if err != nil {
		metaStoreTestMySQL.Close()
		t.Fatalf("Error retreiving meta: %s", err)
	}

	if meta.Oid != contentOid {
		metaStoreTestMySQL.Close()
		t.Errorf("expected to get content oid, got: %s", meta.Oid)
	}

	if meta.Size != 42 {
		metaStoreTestMySQL.Close()
		t.Errorf("expected to get content size, got: %d", meta.Size)
	}
}

func TestMySQLGetWithoutAuth(t *testing.T) {

	_, err := metaStoreTestMySQL.Get(&RequestVars{Authorization: badAuth, Oid: noAuthOid})
	if !isAuthError(err) {
		metaStoreTestMySQL.Close()
		t.Errorf("expected auth error, got: %s", err)
	}
}

func setupMySQLMeta() error {
	// Setup Config
	Config.Ldap = &LdapConfig{Enabled: true, Server: "ldap://localhost:1389", Base: "o=company",
		UserObjectClass: "posixaccount", UserCn: "uid", BindPass: "admin"}
	Config.MySQL = &MySQLConfig{
		Enabled:  true,
		Host:     "127.0.0.1:3306",
		Username: "lfs_server",
		Password: "pass123",
		Database: "lfs_server_go",
	}

	mysqlStore, err := NewMySQLMetaStore()
	if err != nil {
		fmt.Printf("error initializing test meta store: %s\n", err)
		return errors.New(fmt.Sprintf("error initializing test meta store: %s\n", err))
	}

	metaStoreTestMySQL = mysqlStore

	// Clean up any test
	mysqlStore.client.Exec("TRUNCATE TABLE oid_maps")
	mysqlStore.client.Exec("TRUNCATE TABLE oids")
	mysqlStore.client.Exec("TRUNCATE TABLE projects")

	return nil
}
