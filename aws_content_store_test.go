package main

import (
	"bytes"
	"fmt"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"io/ioutil"
	"os"
	"testing"
)

var awsContentStore *AwsContentStore

func TestAwsContentStorePut(t *testing.T) {
	setupAwsTest()
	defer teardownAwsTest()

	m := &MetaObject{
		Oid:  "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72",
		Size: 12,
	}

	b := bytes.NewBuffer([]byte("test content"))

	if err := awsContentStore.Put(m, b); err != nil {
		t.Fatalf("expected put to succeed, got: %s", err)
	}

	if err := awsContentStore.Exists(m); !err {
		t.Fatalf("expected content to exist after putting")
	}
}

func TestAwsContentStorePutHashMismatch(t *testing.T) {
	setupAwsTest()
	defer teardownAwsTest()

	m := &MetaObject{
		Oid:  "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72",
		Size: 12,
	}

	b := bytes.NewBuffer([]byte("bogus content"))

	if err := awsContentStore.Put(m, b); err == nil {
		t.Fatal("expected put with bogus content to fail")
	}
}

func TestAwsContentStorePutSizeMismatch(t *testing.T) {
	setupAwsTest()
	defer teardownAwsTest()

	m := &MetaObject{
		Oid:  "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72",
		Size: 14,
	}

	b := bytes.NewBuffer([]byte("test content"))

	if err := awsContentStore.Put(m, b); err == nil {
		t.Fatal("expected put with bogus size to fail")
	}

}

func TestAwsContentStoreGet(t *testing.T) {
	setupAwsTest()
	defer teardownAwsTest()

	m := &MetaObject{
		Oid:  "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72",
		Size: 12,
	}

	b := bytes.NewBuffer([]byte("test content"))

	if err := awsContentStore.Put(m, b); err != nil {
		t.Fatalf("expected put to succeed, got: %s", err)
	}

	r, err := awsContentStore.Get(m)
	if err != nil {
		t.Fatalf("expected get to succeed, got: %s", err)
	}

	by, _ := ioutil.ReadAll(r)
	if string(by) != "test content" {
		t.Fatalf("expected to read content, got: %s", string(by))
	}
}

func TestAwsContentStoreGetNonExisting(t *testing.T) {
	setupAwsTest()
	defer teardownAwsTest()

	_, err := awsContentStore.Get(&MetaObject{Oid: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if err == nil {
		t.Fatalf("expected to get an error, but content existed")
	}
}

func TestAwsContentStoreExists(t *testing.T) {
	setupAwsTest()
	defer teardownAwsTest()

	m := &MetaObject{
		Oid:  "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72",
		Size: 12,
	}

	b := bytes.NewBuffer([]byte("test content"))

	if awsContentStore.Exists(m) {
		t.Fatalf("expected content to not exist yet")
	}

	if err := awsContentStore.Put(m, b); err != nil {
		t.Fatalf("expected put to succeed, got: %s", err)
	}

	if !awsContentStore.Exists(m) {
		t.Fatalf("expected content to exist")
	}
}

func awsConnectForTest() *s3.Bucket {
	os.Setenv("AWS_ACCESS_KEY_ID", Config.AwsAccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", Config.AwsSecretAccessKey)
	auth, err := aws.EnvAuth()
	perror(err)
	return s3.New(auth, aws.Regions[Config.AwsRegion]).Bucket(Config.AwsBucketName)
}

func setupAwsTest() {
	bucket := awsConnectForTest()
	bucket.PutBucket(s3.Private)
	store, err := NewAwsContentStore()
	if err != nil {
		fmt.Printf("error initializing content store: %s\n", err)
		os.Exit(1)
	}
	awsContentStore = store
}

func teardownAwsTest() {
	bucket := awsConnectForTest()
	// remove all bucket contents
	items, err := bucket.List("", "", "", 1000)
	if err != nil {
		return
	}
	delItems := make([]string, 0)
	if len(items.Contents) > 0 {
		for _, item := range items.Contents {
			if len(item.Key) < 1 {
				continue
			}
			delItems = append(delItems, item.Key)
		}
	}
	if len(delItems) > 0 {
		oops := bucket.MultiDel(delItems)
		if oops != nil {
			fmt.Println("Oops", oops)
		}
	}
}
