package main

import (
	"fmt"
	"testing"
	"errors"
)

var (
	metaStoreTestCassandra *CassandraMetaStore
)

func TestCassandraGetWithAuth(t *testing.T) {
	serr := setupCassandraMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}

	defer teardownCassandraMeta()
	metaFail, errA := metaStoreTestCassandra.Get(&RequestVars{Authorization: testAuth, Oid: noAuthOid})
	if errA == nil {
		t.Fatalf("Error Should not have access to OID: %s", metaFail.Oid)
	}

	meta, err := metaStoreTestCassandra.Get(&RequestVars{Authorization: testAuth, Oid: contentOid})
	if err != nil {
		t.Fatalf("Error retreiving meta: %s", err)
	}

	if meta.Oid != contentOid {
		t.Errorf("expected to get content oid, got: %s", meta.Oid)
	}

	if meta.Size != contentSize {
		t.Errorf("expected to get content size, got: %d", meta.Size)
	}
}

func TestCassandraGetWithoutAuth(t *testing.T) {
	serr := setupCassandraMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}

	defer teardownCassandraMeta()

	_, err := metaStoreTestCassandra.Get(&RequestVars{Authorization: badAuth, Oid: contentOid})
	if !isAuthError(err) {
		t.Errorf("expected auth error, got: %s", err)
	}
}

func TestCassandraPutWithAuth(t *testing.T) {
	serr := setupCassandraMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}

	defer teardownCassandraMeta()

	meta, err := metaStoreTestCassandra.Put(&RequestVars{Authorization: testAuth, Oid: nonexistingOid, Size: 42})
	if err != nil {
		t.Errorf("expected put to succeed, got : %s", err)
	}

	if meta.Existing {
		t.Errorf("expected meta to not have existed")
	}

	meta, err = metaStoreTestCassandra.Get(&RequestVars{Authorization: testAuth, Oid: nonexistingOid})
	if err != nil {
		t.Errorf("expected to be able to retreive new put, got : %s", err)
	}

	if meta.Oid != nonexistingOid {
		t.Errorf("expected oids to match, got: %s", meta.Oid)
	}

	if meta.Size != 42 {
		t.Errorf("expected sizes to match, got: %d", meta.Size)
	}

	meta, err = metaStoreTestCassandra.Put(&RequestVars{Authorization: testAuth, Oid: nonexistingOid, Size: 42})
	if err != nil {
		t.Errorf("expected put to succeed, got : %s", err)
	}

	if !meta.Existing {
		t.Errorf("expected meta to now exist")
	}
}

func TestCassandraPuthWithoutAuth(t *testing.T) {
	serr := setupCassandraMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}
	defer teardownCassandraMeta()

	_, err := metaStoreTestCassandra.Put(&RequestVars{Authorization: badAuth, Oid: contentOid, Size: 42})
	if !isAuthError(err) {
		t.Errorf("expected auth error, got: %s", err)
	}
}

func setupCassandraMeta() error {
	store, err := NewCassandraMetaStore()
	if err != nil {
		fmt.Printf("error initializing test meta store: %s\n", err)
		return errors.New(fmt.Sprintf("error initializing test meta store: %s\n", err))
	}

	metaStoreTestCassandra = store
	if err := metaStoreTestCassandra.AddUser(testUser, testPass); err != nil {
		teardownCassandraMeta()
		fmt.Printf("error adding test user to meta store: %s\n", err)
		return errors.New(fmt.Sprintf("error adding test user to meta store: %s\n", err))
	}

	rv := &RequestVars{Authorization: testAuth, Oid: contentOid, Size: contentSize}
	if _, err := metaStoreTestCassandra.Put(rv); err != nil {
		teardownCassandraMeta()
		fmt.Printf("error seeding cassandra test meta store: %s\n", err)
		return errors.New(fmt.Sprintf("error seeding cassandra test meta store: %s\n", err))
	}
	return nil
}

func teardownCassandraMeta() {
	DropCassandra(NewCassandraSession().Client)
}
