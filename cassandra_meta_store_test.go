package main

import (
	"errors"
	"fmt"
	"testing"
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

func TestCassandraOids(t *testing.T) {
	serr := setupCassandraMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}
	defer teardownCassandraMeta()

	allOids, _ := metaStoreTestCassandra.findAllOids()
	cb := len(allOids)

	createOidErr := metaStoreTestCassandra.createOid(nonexistingOid, 1)
	if createOidErr != nil {
		t.Errorf("Failed to create OID")
	}

	allOids, _ = metaStoreTestCassandra.findAllOids()
	if cb == len(allOids) {
		t.Errorf("Failed add OID")
	}

	mo, findOidErr := metaStoreTestCassandra.findOid(nonexistingOid)
	if findOidErr != nil {
		t.Errorf("Failed find OID")
	}
	if mo == nil || mo.Oid != nonexistingOid {
		t.Errorf("Failed find OID, it does not match")
	}

	delOidErr := metaStoreTestCassandra.removeOid(nonexistingOid)
	if delOidErr != nil {
		t.Errorf("Failed remove OID")
	}

}

func TestCassandraProjects(t *testing.T) {
	serr := setupCassandraMeta()
	if serr != nil {
		t.Errorf(serr.Error())
	}
	defer teardownCassandraMeta()

	createErr := metaStoreTestCassandra.createProject(extraRepo)
	if createErr != nil {
		t.Errorf("Failed to create project")
	}

	proj, findPErr := metaStoreTestCassandra.findProject(extraRepo)
	if findPErr != nil {
		t.Errorf("Failed to find project")
	}
	if proj.Name != extraRepo {
		t.Errorf("Failed to find project, got wrong name in response")
	}

	listProjects, err := metaStoreTestCassandra.findAllProjects()
	if err != nil {
		t.Errorf("Failed getting cassandra projects")
	}
	for _, p := range listProjects {
		fmt.Println("project", p)
	}

	projects, err := metaStoreTestCassandra.Projects()
	if err != nil {
		t.Errorf("Failed getting cassandra projects")
	}
	for _, p := range projects {
		fmt.Println("project", p)
	}

	delErr := metaStoreTestCassandra.removeProject(extraRepo)
	if delErr != nil {
		t.Errorf("Failed to delete project")
	}

	_, findPErrEmpty := metaStoreTestCassandra.findProject(extraRepo)
	if findPErrEmpty == nil {
		t.Errorf("findProject should have raised an error")
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
