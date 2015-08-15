package main

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"reflect"
	"runtime"
	"strings"
)

type CassandraConfig struct {
	Hosts    string
	Keyspace string
	Username string
	Password string
}

type AwsConfig struct {
	AccessKeyId     string
	SecretAccessKey string
	Region          string
	BucketName      string
	BucketAcl       string
}

type LdapConfig struct {
	Enabled         bool
	Server          string
	Base            string
	UserObjectClass string
	UserCn          string
	BindDn          string
	BindPass        string
}

type RedisConfig struct {
	Host     string
	Port     int64
	Password string
	DB       int64
}

// Configuration holds application configuration. Values will be pulled from
// environment variables, prefixed by keyPrefix. Default values can be added
// via tags.
type Configuration struct {
	Listen       string
	Host         string
	ContentPath  string
	AdminUser    string
	AdminPass    string
	Cert         string
	Key          string
	Scheme       string
	Public       bool
	MetaDB       string
	BackingStore string
	ContentStore string
	LogFile      string
	NumProcs     int
	Aws          *AwsConfig
	Cassandra    *CassandraConfig
	Ldap         *LdapConfig
	Redis        *RedisConfig
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

// iterate thru config.yaml and parse it
// always called when initializing Config
func init() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		panic(fmt.Sprint("unable to read config.ini, %v", err))
	}
	if GoEnv == "" {
		GoEnv = "production"
	}

	awsConfig := &AwsConfig{AccessKeyId: "", SecretAccessKey: "", Region: "USWest",
		BucketName: "lfs-server-go-objects", BucketAcl: "bucket-owner-full-control"}
	ldapConfig := &LdapConfig{Enabled: false, Server: "ldap://localhost:1389", Base: "dc=testers,c=test,o=company",
		UserObjectClass: "person", UserCn: "uid", BindDn: "", BindPass: ""}
	cassandraConfig := &CassandraConfig{Hosts: "localhost", Keyspace: "lfs_server_go", Username: "", Password: ""}
	configuration := &Configuration{
		Listen:       "tcp://:8080",
		Host:         "localhost:8080",
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
	}
	err = cfg.Section("Main").MapTo(configuration)
	err = cfg.Section("Aws").MapTo(configuration.Aws)
	err = cfg.Section("Ldap").MapTo(configuration.Ldap)
	err = cfg.Section("Cassandra").MapTo(configuration.Cassandra)
	// We have to do redis differently because the ini lib
	// tries to make int64 into time structs
	redis := cfg.Section("Redis").KeysHash()
	db, err := cfg.Section("Redis").GetKey("DB")
	if err != nil {
		cfg.Section("Redis").NewKey("DB", "0")
		db, _ = cfg.Section("Redis").GetKey("DB")
	}
	port, err := cfg.Section("Redis").GetKey("Port")
	if err != nil {
		cfg.Section("Redis").NewKey("Port", "6379")
		port, _ = cfg.Section("Redis").GetKey("Port")
	}
	configuration.Redis = &RedisConfig{
		Host:     redis["Host"],
		Password: redis["Password"],
		DB:       db.MustInt64(0),
		Port:     port.MustInt64(6379),
	}
	Config = configuration

}

func (c *Configuration) DumpConfig() map[string]string {
	configDump := make(map[string]string)
	for name, _ := range attributes(&Configuration{}) {
		valueE := reflect.ValueOf(Config).Elem()
		field := valueE.FieldByName(name)
		configDump[name] = field.String()
	}
	return configDump
}
