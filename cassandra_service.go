package main

import (
	"fmt"
	"github.com/gocql/gocql"
)

type CassandraService struct {
	Client *gocql.Session
}

// TODO: Add auth for cassandra
func NewCassandraSession() *CassandraService {
	cluster := gocql.NewCluster(Config.Cassandra.Hosts)
	q := fmt.Sprintf("create keyspace if not exists %s_%s with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };", Config.Cassandra.Keyspace, GoEnv)
	session, err := cluster.CreateSession()
	err = session.Query(q).Exec()
	session.Close()
	cluster.Keyspace = fmt.Sprintf("%s_%s", Config.Cassandra.Keyspace, GoEnv)
	cluster.Consistency = gocql.Quorum
	session, err = cluster.CreateSession()
	perror(initializeCassandra(session))
	perror(err)
	logger.Log(kv{"fn": "cassandra_service", "msg": fmt.Sprintf("Connecting to host '%s'\n", Config.Cassandra.Hosts)})
	logger.Log(kv{"fn": "cassandra_service", "msg": fmt.Sprintf("Cassandra.namespace '%s_%s'\n", Config.Cassandra.Keyspace, GoEnv)})
	return &CassandraService{Client: session}
}

func initializeCassandra(session *gocql.Session) error {
	// projects table
	q := fmt.Sprintf("create table if not exists projects (name text PRIMARY KEY, oids SET<text>);")
	err := session.Query(q).Exec()
	if err != nil {
		return err
	}

	// create an index so we can search on oids
	q = fmt.Sprintf("create index if not exists on projects(oids);")
	err = session.Query(q).Exec()
	if err != nil {
		return err
	}

	// Oids table
	q = fmt.Sprintf("create table if not exists oids(oid text primary key, size bigint);")
	session.Query(q).Exec()
	if err != nil {
		return err
	}

	// user management
	q = fmt.Sprintf("create table if not exists users(username text primary key, password text);")
	return session.Query(q).Exec()
}

func DropCassandra(session *gocql.Session) error {
	config := Config.Cassandra
	m := fmt.Sprintf("%s_%s", config.Keyspace, GoEnv)
	q := fmt.Sprintf("drop keyspace %s;", m)
	c := NewCassandraSession().Client
	return c.Query(q).Exec()
}
