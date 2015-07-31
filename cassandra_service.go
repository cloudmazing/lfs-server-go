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
	cluster := gocql.NewCluster(Config.CassandraHosts)
	createCassandraKeyspace()
	cluster.Keyspace = fmt.Sprintf("%s_%s", Config.CassandraKeyspace, GoEnv)
	cluster.Consistency = gocql.Quorum
	m := fmt.Sprintf("Connecting to host '%s'\n", Config.CassandraHosts)
	logger.Log(kv{"fn": "cassandra_service", "msg": m})
	m = fmt.Sprintf("Cassandra namespace '%s_%s'\n", Config.CassandraKeyspace, GoEnv)
	logger.Log(kv{"fn": "cassandra_service", "msg": m})
	session, err := cluster.CreateSession()
	perror(err)
	//	defer session.Close()
	return &CassandraService{Client: session}
}

func createCassandraKeyspace() error {
	cluster := gocql.NewCluster(Config.CassandraHosts)
	q := fmt.Sprintf("create keyspace if not exists %s_%s with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };", Config.CassandraKeyspace, GoEnv)
	c, err := cluster.CreateSession()
	c.Query(q).Exec()
	defer c.Close()
	return err
}
func InitializeCassandra() error {
	//	cs := gocql.NewCluster(Config.CassandraHosts)
	c := NewCassandraSession().Client
	createCassandraKeyspace()
	// projects table
	q := fmt.Sprintf("create table if not exists projects (name text PRIMARY KEY, oids SET<text>);")
	err := c.Query(q).Exec()
	perror(err)
	// Oids table
	q = fmt.Sprintf("create table if not exists oids(oid text primary key, size bigint);")
	c.Query(q).Exec()
	perror(err)
	// user management
	q = fmt.Sprintf("create table if not exists users(username text primary key, password text);")
	c.Query(q).Exec()
	perror(err)
	return nil
}

func DropCassandra() error {
	m := fmt.Sprintf("%s_%s", Config.CassandraKeyspace, GoEnv)
	q := fmt.Sprintf("drop keyspace %s;", m)
	c := NewCassandraSession().Client
	return c.Query(q).Exec()
}
