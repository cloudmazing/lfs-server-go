package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const (
	ContentType = "binary/octet-stream"
)

// ContentStore provides a simple file system based storage.
type AwsContentStore struct {
	client  *s3.S3
	bucket  *s3.Bucket
	authId  string
	authKey string
}

// NewContentStore creates a ContentStore at the base directory.
func NewAwsContentStore() (*AwsContentStore, error) {
	os.Setenv("AWS_ACCESS_KEY_ID", Config.AwsAccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", Config.AwsSecretAccessKey)
	auth, err := aws.EnvAuth()
	if err != nil {
		logger.Log(kv{"fn": "AwsContentStore.NewAwsContentStore", "err": ": " + err.Error()})
		return &AwsContentStore{}, err
	}
	client := s3.New(auth, aws.Regions[Config.AwsRegion])
	bucket := client.Bucket(Config.AwsBucketName)
	self := &AwsContentStore{bucket: bucket, client: client}
	self.makeBucket()
	return self, nil
}

// Make the bucket if it does not exist
func (s *AwsContentStore) makeBucket() error {
	buckets, err := s.bucket.ListBuckets()
	if err != nil {
		logger.Log(kv{"fn": "AwsContentStore.makeBucket", "err": ": " + err.Error()})
		return err
	}
	var exists bool
	exists = false
	for _, b := range buckets.Buckets {
		if b.Name == s.bucket.Name {
			exists = true
		}
	}
	if !exists {
		err := s.bucket.PutBucket(s3.ACL(Config.AwsBucketAcl))
		return err
	}
	return nil
}

func (s *AwsContentStore) Get(meta *MetaObject) (io.Reader, error) {
	path := transformKey(meta.Oid)
	return s.bucket.GetReader(path)
}

func (s *AwsContentStore) getMetaData(meta *MetaObject) (*s3.Key, error) {
	path := transformKey(meta.Oid)
	return s.bucket.GetKey(path)
}

// TODO: maybe take write errors into account and buffer/resend to amazon?
func (s *AwsContentStore) Put(meta *MetaObject, r io.Reader) error {
	path := transformKey(meta.Oid)
	/*
		There is probably a better way to compute this but we need to write the file to memory to
		 compute the sha256 value and make sure what we're writing is correct.
		 If not, git wont be able to find it later
	*/
	hash := sha256.New()
	buf, _ := ioutil.ReadAll(r)
	hw := io.MultiWriter(hash)
	written, err := io.Copy(hw, bytes.NewReader(buf))
	if err != nil {
		logger.Log(kv{"fn": "AwsContentStore.Put", "err": ": " + err.Error()})
		return err
	}
	// Check that we've written out the entire file for computing the sha
	if written != meta.Size {
		return errSizeMismatch
	}
	shaStr := hex.EncodeToString(hash.Sum(nil))
	if shaStr != meta.Oid {
		return errHashMismatch
	}
	retStat := s.bucket.PutReader(path, bytes.NewReader(buf), meta.Size, ContentType, s3.PublicRead)
	k, kerr := s.getMetaData(meta)
	if kerr != nil {
		logger.Log(kv{"fn": "AwsContentStore.Put", "err": ": " + kerr.Error()})
		return errWriteS3
	}
	if k.Size != meta.Size {
		return errSizeMismatch
	}
	return retStat
}

func (s *AwsContentStore) Exists(meta *MetaObject) bool {
	path := transformKey(meta.Oid)
	// returns a 404 error if its not there
	_, err := s.bucket.GetKey(path)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false
		} else {
			logger.Log(kv{"fn": "AwsContentStore.Exists", "err": ": " + err.Error()})
			return false
		}
	}
	// if the object is not there, a 404 error is raised
	return true
}
