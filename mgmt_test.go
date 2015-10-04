package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

type values map[string]string

func testRequestHeader(t *testing.T, r *http.Request, header string, want string) {
	if value := r.Header.Get(header); want != value {
		t.Errorf("Header %s = %s, want: %s", header, value, want)
	}
}

func testResponseHeader(t *testing.T, r *http.Response, header string, want string) {
	if value := r.Header.Get(header); want != value {
		t.Errorf("Header %s = %s, want: %s", header, value, want)
	}
}

func TestMgmtGetObjects_Json(t *testing.T) {
	req, err := http.NewRequest("GET", lfsServer.URL+"/mgmt/objects", nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	header := map[string][]string{"Accept": {"application/json"}, "Accept-Encoding": {"gzip", "text"}}
	req.Header = header
	req.SetBasicAuth(testUser, testPass)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("response code failed. Expected 200, got %d", res.StatusCode)
	}
	var metas []*MetaObject
	data, _ := ioutil.ReadAll(res.Body)
	// validate the request header
	testRequestHeader(t, req, "Accept", "application/json")
	// validate that we are returning the correct header
	testResponseHeader(t, res, "Content-Type", "application/json")
	json.Unmarshal(data, &metas)
	var good bool
	var meta *MetaObject
	good = false
	for _, m := range metas {
		if m.Oid == contentOid {
			meta = m
			good = true
		}
	}
	if !good {
		t.Errorf("expected oid to be %+v, got %+v", contentOid, meta.Oid)
	}
}
func TestMgmtGetProjects_Json(t *testing.T) {
	_, err := testMetaStore.Put(&RequestVars{Repo: testRepo, User: testUser, Oid: contentOid, Authorization: testAuth})
	if err != nil {
		fmt.Println("got an err", err.Error())
	}
	req, err := http.NewRequest("GET", lfsServer.URL+"/mgmt/projects", nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	header := map[string][]string{"Accept": {"application/json"}, "Accept-Encoding": {"gzip", "text"}}
	req.Header = header
	req.SetBasicAuth(testUser, testPass)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}
	var metas []*MetaProject
	var meta *MetaProject
	data, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(data, &metas)
	var good bool
	good = false
	for _, m := range metas {
		if m.Name == testRepo {
			meta = m
			good = true
		}
	}
	if !good {
		t.Errorf("expected project name to be %+v, got %+v", testRepo, meta)
	}
}
