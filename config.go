package main

import (
	"fmt"
	"github.com/fatih/structs"
	"gopkg.in/ini.v1"
	"os"
	"runtime"
	"strings"
)

type CassandraConfig struct {
	Hosts    string `json:"hosts"`
	Keyspace string `json:"keyspace"`
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}

type AwsConfig struct {
	AccessKeyId     string `json:"accesskeyid"`
	SecretAccessKey string `json:"secretaccesskey"`
	Region          string `json:"region"`
	BucketName      string `json:"bucketname"`
	BucketAcl       string `json:"bucketacl"`
	Enabled         bool   `json:"enabled"`
}

type LdapConfig struct {
	Enabled         bool   `json:"enabled"`
	Server          string `json:"server"`
	Base            string `json:"base"`
	UserObjectClass string `json:"userobjectclass"`
	UserCn          string `json:"usercn"`
	BindDn          string `json:"binddn"`
	BindPass        string `json:"bindpass"`
}

/*
MySQLConfig (MySQL configuration struct)
  => Host     :- MySQL host e.g 127.0.0.1:3306
  => Database :- Name of the database default to lfs_server_go
  => Username :- DB username
  => Password :- DB password
*/
type MySQLConfig struct {
	Host     string `json:"host"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}

// Configuration holds application configuration. Values will be pulled from
// environment variables, prefixed by keyPrefix. Default values can be added
// via tags.
type Configuration struct {
	Listen       string           `json:"listen"`
	Host         string           `json:"host"`
	UrlContext   string           `json:"url_context"`
	ContentPath  string           `json:"content_path"`
	AdminUser    string           `json:"admin_user"`
	AdminPass    string           `json:"admin_pass"`
	Cert         string           `json:"cert"`
	Key          string           `json:"key"`
	Scheme       string           `json:"scheme"`
	Public       bool             `json:"public"`
	MetaDB       string           `json:"metadb"`
	BackingStore string           `json:"backing_store"`
	ContentStore string           `json:"content_store"`
	LogFile      string           `json:"logfile"`
	NumProcs     int              `json:"numprocs"`
	Aws          *AwsConfig       `json:"aws"`
	Cassandra    *CassandraConfig `json:"cassandra"`
	Ldap         *LdapConfig      `json:"ldap"`
	MySQL        *MySQLConfig     `json:"mysql"`
}

func (c *Configuration) IsHTTPS() bool {
	return strings.Contains(Config.Scheme, "https")
}

func (c *Configuration) IsPublic() bool {
	return Config.Public
}

// Config is the global app configuration
//var Config = &Configuration{}
var GoEnv = os.Getenv("GO_ENV")
var Config = &Configuration{}

// iterate thru config.ini and parse it
// always called when initializing Config
func init() {
	configFile := os.Getenv("LFS_SERVER_GO_CONFIG")
	if configFile == "" {
		fmt.Println("LFS_SERVER_GO_CONFIG is not set, Using config file %v", configFile)
		configFile = "config.ini"
	}

	cfg, err := ini.Load(configFile)
	if err != nil {
		panic(fmt.Sprintf("unable to read config.ini, %v", err))
	}
	if GoEnv == "" {
		GoEnv = "production"
	}

	//Force scheme to be a valid value
	if cfg.Section("Main").Key("Scheme").String() != "" {
		val := cfg.Section("Main").Key("Scheme").String()
		switch val {
		case
			"http", "https":
			val = val
		default:
			val = "http"
		}
	}

	awsConfig := &AwsConfig{
		AccessKeyId:     "",
		SecretAccessKey: "",
		Region:          "USWest",
		BucketName:      "lfs-server-go-objects",
		BucketAcl:       "bucket-owner-full-control",
		Enabled:         false,
	}
	ldapConfig := &LdapConfig{
		Server:          "ldap://localhost:1389",
		Base:            "dc=testers,c=test,o=company",
		UserObjectClass: "person",
		Enabled:         false,
		UserCn:          "uid",
		BindDn:          "",
		BindPass:        "",
	}
	cassandraConfig := &CassandraConfig{
		Hosts:    "localhost",
		Keyspace: "lfs_server_go",
		Username: "",
		Password: "",
		Enabled:  false,
	}
	mysqlConfig := &MySQLConfig{
		Host:     "",
		Database: "lfs_server_go",
		Username: "",
		Password: "",
		Enabled:  false,
	}
	configuration := &Configuration{
		Listen:       "tcp://:8080",
		Host:         "localhost:8080",
		UrlContext:   "",
		ContentPath:  "lfs-content",
		AdminUser:    "admin",
		AdminPass:    "admin",
		Cert:         "",
		Key:          "",
		Scheme:       "http",
		Public:       true,
		MetaDB:       "lfs-test.db",
		BackingStore: "bolt",
		ContentStore: "filesystem",
		NumProcs:     runtime.NumCPU(),
		Ldap:         ldapConfig,
		Aws:          awsConfig,
		Cassandra:    cassandraConfig,
		MySQL:        mysqlConfig,
	}
	err = cfg.Section("Main").MapTo(configuration)
	err = cfg.Section("Aws").MapTo(configuration.Aws)
	err = cfg.Section("Ldap").MapTo(configuration.Ldap)
	err = cfg.Section("Cassandra").MapTo(configuration.Cassandra)
	err = cfg.Section("MySQL").MapTo(configuration.MySQL)
	Config = configuration
}

func (c *Configuration) DumpConfig() map[string]interface{} {
	m := structs.Map(Config)
	return m
}
