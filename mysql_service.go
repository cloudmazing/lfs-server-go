package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/gorp.v1"
	"strings"
)

/*
MySQLService struct
*/
type MySQLService struct {
	Client *sql.DB
}

/*
Projects table struct
*/
type Projects struct {
	id   int64
	name string
}

/*
Oids table struct
*/
type Oids struct {
	oid  string
	size int64
}

/*
OidMaps table struct
*/
type OidMaps struct {
	oid       string
	projectID int64
}

/*
NewMySQLSession (method used in mysql_meta_store.go)
create requeired table and return sql client object
*/
func NewMySQLSession() *MySQLService {

	validate := validateConfig()

	if validate {
		// Create MySQL Client
		dqs := fmt.Sprintf("%s:%s@tcp(%s)/%s",
			Config.MySQL.Username,
			Config.MySQL.Password,
			Config.MySQL.Host,
			Config.MySQL.Database)

		// Open connection
		db, err := sql.Open("mysql", dqs)
		dbMap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
		perror(createTables(dbMap))
		perror(err)
		return &MySQLService{Client: db}
	}

	logger.Log(kv{"fn": "NewMySQLSession", "msg": "MySQL configuration validation failed"})
	return nil
}

func createTables(client *gorp.DbMap) error {
	client.AddTableWithName(Projects{}, "projects").SetKeys(true, "id").ColMap("name").SetUnique(true)
	client.AddTableWithName(Oids{}, "oids").SetKeys(false, "oid")
	client.AddTableWithName(OidMaps{}, "oid_maps")
	// dbmap.AddTableWithName(users{}, "users").SetKeys(false, "name")
	err := client.CreateTablesIfNotExists()

	if err != nil {
		return err
	}
	return nil
}

func validateConfig() bool {
	if len(strings.TrimSpace(Config.MySQL.Database)) == 0 && len(strings.TrimSpace(Config.MySQL.Host)) == 0 {
		logger.Log(kv{"fn": "NewMySQLSession", "msg": "Require Host and Database to connect MySQL "})
		return false
	}

	if len(strings.TrimSpace(Config.MySQL.Username)) == 0 && len(strings.TrimSpace(Config.MySQL.Password)) == 0 {
		logger.Log(kv{"fn": "NewMySQLSession", "msg": "Require Username and Password to connect MySQL "})
		return false
	}

	return true
}
