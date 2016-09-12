package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"time"

	"encoding/base64"
	"fmt"
	"github.com/boltdb/bolt"
	"strings"
)

// MetaStore implements a metadata storage. It stores user credentials and Meta information
// for objects. The storage is handled by boltdb.
type MetaStore struct {
	db *bolt.DB
}

var (
	errNoBucket = errors.New("Bucket not found")
)

var (
	usersBucket    = []byte("users")
	objectsBucket  = []byte("objects")
	projectsBucket = []byte("projects")
)

// NewMetaStore creates a new MetaStore using the boltdb database at dbFile.
func NewMetaStore(dbFile string) (*MetaStore, error) {
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(usersBucket); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists(objectsBucket); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists(projectsBucket); err != nil {
			return err
		}

		return nil
	})

	return &MetaStore{db: db}, nil
}

// Get retrieves the Meta information for an object given information in
// RequestVars
func (s *MetaStore) Get(rv *RequestVars) (*MetaObject, error) {
	if !s.authenticate(rv.Authorization) {
		return nil, newAuthError()
	}

	var meta MetaObject
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(objectsBucket)
		if bucket == nil {
			return errNoBucket
		}

		value := bucket.Get([]byte(rv.Oid))
		if len(value) == 0 {
			return errObjectNotFound
		}

		dec := gob.NewDecoder(bytes.NewBuffer(value))
		return dec.Decode(&meta)
	})

	if err != nil {
		logger.Log(kv{"fn": "meta_store", "msg": err.Error()})
		return nil, err
	}

	return &meta, nil
}

func (s *MetaStore) findProject(projectName string) (*MetaProject, error) {
	// var projects []*MetaProject
	var project *MetaProject
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(projectsBucket)
		if bucket == nil {
			return errNoBucket
		}
		val := bucket.Get([]byte(projectName))
		if len(val) < 1 {
			return errProjectNotFound
		}
		dec := gob.NewDecoder(bytes.NewBuffer(val))
		return dec.Decode(&project)
	})
	if err != nil {
		return nil, err
	}
	if project.Name != "" {
		return project, nil
	}
	return nil, errProjectNotFound
}

// Currently the OIDS are nil
func (s *MetaStore) createProject(rv *RequestVars) error {
	if _, err := s.findProject(rv.Repo); err == nil {
		// already there
		return nil
	}

	if rv.Repo == "" {
		return nil
	}
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(projectsBucket)
		if bucket == nil {
			// should never get here unless the db is jacked
			return errNoBucket
		}
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		meta := MetaProject{Name: rv.Repo, Oids: []string{rv.Oid}}
		err := enc.Encode(meta)
		// Just a bunch o keys
		err = bucket.Put([]byte(rv.Repo), buf.Bytes())
		if err != nil {
			return err
		}

		return nil
	})
	return err
}

// Put writes meta information from RequestVars to the store.
func (s *MetaStore) Put(rv *RequestVars) (*MetaObject, error) {
	if !s.authenticate(rv.Authorization) {
		return nil, newAuthError()
	}

	// Check if it exists first
	if meta, err := s.Get(rv); err == nil {
		meta.Existing = true
		return meta, nil
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	meta := MetaObject{Oid: rv.Oid, Size: rv.Size, ProjectNames: []string{rv.Repo}}
	err := enc.Encode(meta)
	if err != nil {
		return nil, err
	}
	// create the project first, if we can
	if rv.Repo != "" {
		err := s.createProject(rv)
		if err != nil {
			logger.Log(kv{"fn": "Put", "err": err.Error()})
			return nil, err
		}
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(objectsBucket)
		if bucket == nil {
			return errNoBucket
		}

		err = bucket.Put([]byte(rv.Oid), buf.Bytes())
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &meta, nil
}

// Close closes the underlying boltdb.
func (s *MetaStore) Close() {
	s.db.Close()
}

// AddUser adds user credentials to the meta store.
func (s *MetaStore) AddUser(user, pass string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		if bucket == nil {
			return errNoBucket
		}
		encryptedPass, err := encryptPass([]byte(pass))
		if err != nil {
			return err
		}
		if val := bucket.Get([]byte(user)); len(val) > 0 {
			return nil // Already there
		}
		return bucket.Put([]byte(user), []byte(encryptedPass))
	})
	return err
}

// DeleteUser removes user credentials from the meta store.
func (s *MetaStore) DeleteUser(user string) error {
	if Config.Ldap.Enabled {
		return errNotImplemented
	}
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		if bucket == nil {
			return errNoBucket
		}

		err := bucket.Delete([]byte(user))
		return err
	})

	return err
}

// Users returns all MetaUsers in the meta store
func (s *MetaStore) Users() ([]*MetaUser, error) {
	if Config.Ldap.Enabled {
		return []*MetaUser{}, errNotImplemented
	}
	var users []*MetaUser

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		if bucket == nil {
			return errNoBucket
		}

		bucket.ForEach(func(k, v []byte) error {
			users = append(users, &MetaUser{Name: string(k)})
			return nil
		})
		return nil
	})

	return users, err
}

// Objects returns all MetaObjects in the meta store
func (s *MetaStore) Objects() ([]*MetaObject, error) {
	var objects []*MetaObject

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(objectsBucket)
		if bucket == nil {
			return errNoBucket
		}

		bucket.ForEach(func(k, v []byte) error {
			var meta MetaObject
			dec := gob.NewDecoder(bytes.NewBuffer(v))
			err := dec.Decode(&meta)
			if err != nil {
				return err
			}
			objects = append(objects, &meta)
			return nil
		})
		return nil
	})
	return objects, err
}

// authenticate uses the authorization string to determine whether
// or not to proceed. This server assumes an HTTP Basic auth format.
func (s *MetaStore) authenticate(authorization string) bool {
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
		logger.Log(kv{"fn": "meta_store.authenticate", "msg": err.Error()})
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
	value := ""

	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		if bucket == nil {
			return errNoBucket
		}

		value = string(bucket.Get([]byte(user)))
		return nil
	})
	match, err := checkPass([]byte(value), []byte(password))
	if err != nil {
		logger.Log(kv{"fn": "meta_store.authenticate", "msg": fmt.Sprintf("Decrypt error: %s", err.Error())})
	}
	return match
}

func (s *MetaStore) Projects() ([]*MetaProject, error) {
	var projects []*MetaProject
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(projectsBucket)
		if bucket == nil {
			return errNoBucket
		}

		bucket.ForEach(func(k, v []byte) error {
			var meta MetaProject
			dec := gob.NewDecoder(bytes.NewBuffer(v))
			err := dec.Decode(&meta)
			if err != nil {
				return err
			}
			projects = append(projects, &meta)
			return nil
		})
		return nil
	})
	return projects, err
}

/*
AddProject (create a new project using POST)
Only implemented on MySQL meta store
*/
func (s *MetaStore) AddProject(name string) error {
	return errNotImplemented
}
