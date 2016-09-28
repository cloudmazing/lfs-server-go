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
		t.Errorf("expected AddProject to succeed, got : %s", err)
	}
}

func TestMySQLPutWithAuth(t *testing.T) {
	serr := setupMySQLMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}

	meta, err := metaStoreTestMySQL.Put(&RequestVars{Authorization: testAuth, Oid: nonexistingOid, Size: 42})
	if err != nil {
		t.Errorf("expected put to succeed, got: %s", err)
	}

	if meta.Existing {
		t.Errorf("expected meta to not have existed")
	}

	_, err = metaStoreTestMySQL.Get(&RequestVars{Authorization: testAuth, Oid: nonexistingOid})
	if err == nil {
		t.Errorf("expected new put to not be committed yet")
	}

	meta, err = metaStoreTestMySQL.GetPending(&RequestVars{Authorization: testAuth, Oid: nonexistingOid})
	if err != nil {
		t.Errorf("expected new put to be pending, got: %s", err)
	}

	if meta.Oid != nonexistingOid {
		t.Errorf("expected oids to match, got: %s", meta.Oid)
	}

	if meta.Size != 42 {
		t.Errorf("expected sizes to match, got: %d", meta.Size)
	}

	meta, err = metaStoreTestMySQL.Commit(&RequestVars{Authorization: testAuth, Oid: nonexistingOid, Size: 42})
	if err != nil {
		t.Errorf("expected commit to succeed, got: %s", err)
	}

	if !meta.Existing {
		t.Errorf("expected existing to become true after commit")
	}

	_, err = metaStoreTestMySQL.Get(&RequestVars{Authorization: testAuth, Oid: nonexistingOid})
	if err != nil {
		t.Errorf("expected new put to be committed now, got: %s", err)
	}

	if !meta.Existing {
		t.Errorf("expected existing to be true for a committed object")
	}

	meta, err = metaStoreTestMySQL.Put(&RequestVars{Authorization: testAuth, Oid: nonexistingOid, Size: 42})
	if err != nil {
		t.Errorf("expected putting an duplicate object to succeed, got: %s", err)
	}

	if !meta.Existing {
		t.Errorf("expecting existing to be true for a duplicate object")
	}
}

func TestMySQLPutWithoutAuth(t *testing.T) {
	serr := setupMySQLMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}

	_, err := metaStoreTestMySQL.Put(&RequestVars{Authorization: badAuth, User: testUser, Oid: contentOid, Size: 42})
	if !isAuthError(err) {
		t.Errorf("expected auth error, got: %s", err)
	}

	_, err = metaStoreTestMySQL.Put(&RequestVars{User: testUser, Oid: contentOid, Size: 42, Repo: testRepo})
	if !isAuthError(err) {
		t.Errorf("expected auth error, got: %s", err)
	}
}

func TestMySQLGetWithAuth(t *testing.T) {
	serr := setupMySQLMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}

	meta, err := metaStoreTestMySQL.Get(&RequestVars{Authorization: testAuth, Oid: noAuthOid})
	if err == nil {
		t.Fatalf("expected get to fail with unknown oid, got: %s", meta.Oid)
	}

	meta, err = metaStoreTestMySQL.Get(&RequestVars{Authorization: testAuth, Oid: contentOid})
	if err != nil {
		t.Fatalf("expected get to succeed, got: %s", err)
	}

	if meta.Oid != contentOid {
		t.Errorf("expected to get content oid, got: %s", meta.Oid)
	}

	if meta.Size != contentSize {
		t.Errorf("expected to get content size, got: %d", meta.Size)
	}
}

func TestMySQLGetWithoutAuth(t *testing.T) {
	serr := setupMySQLMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}

	_, err := metaStoreTestMySQL.Get(&RequestVars{Authorization: badAuth, Oid: noAuthOid})
	if !isAuthError(err) {
		t.Errorf("expected auth error, got: %s", err)
	}

	_, err = metaStoreTestMySQL.Get(&RequestVars{Oid: noAuthOid})
	if !isAuthError(err) {
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
	mysqlStore.client.Exec("TRUNCATE TABLE pending_oids")

	rv := &RequestVars{Authorization: testAuth, Oid: contentOid, Size: contentSize, Repo: testRepo}

	if _, err := metaStoreTestMySQL.Put(rv); err != nil {
		fmt.Printf("error seeding mysql test meta store: %s\n", err)
		return errors.New(fmt.Sprintf("error seeding mysql test meta store: %s\n", err))
	}
	if _, err := metaStoreTestMySQL.Commit(rv); err != nil {
		fmt.Printf("error seeding mysql test meta store: %s\n", err)
		return errors.New(fmt.Sprintf("error seeding mysql test meta store: %s\n", err))
	}

	return nil
}
