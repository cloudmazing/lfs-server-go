package main

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"reflect"
	"strings"
)

type CassandraConfig struct {
	Hosts    string `config:"localhost"` //separated by ","
	Keyspace string `config:"lfs_server_go"`
	Username string `config:""`
	Password string `config:""`
}

type AwsConfig struct {
	AccessKeyId     string `config:""`
	SecretAccessKey string `config:""`
	Region          string `config:"USWest"`
	BucketName      string `config:"lfs-server-go-objects"`
	BucketAcl       string `config:"bucket-owner-full-control"`
}

type LdapConfig struct {
	Enabled         bool   `config:false`
	Server          string `config:"ldap://localhost:1389"`
	Base            string `config:"dc=testers,c=test,o=company"`
	UserObjectClass string `config:"(&(objectclass=person))"`
	UserCn          string `config:"uid"`
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
	Listen       string `config:"tcp://:8080"`
	Host         string `config:"localhost:8080"`
	ContentPath  string `config:"lfs-content"`
	AdminUser    string `config:"admin"`
	AdminPass    string `config:"admin"`
	Cert         string `config:""`
	Key          string `config:""`
	Scheme       string `config:"http"`
	Public       bool   `config:"public"`
	MetaDB       string `config:"lfs-test.db"`
	BackingStore string `config:"bolt"`
	ContentStore string `config:"filesystem"`
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
	//	file, err := ioutil.ReadFile("config.ini")
	fmt.Println("Loading config...")
	cfg, err := ini.Load("config.ini")
	if err != nil {
		panic(fmt.Sprint("unable to read config.ini, %v", err))
	}
	if GoEnv == "" {
		fmt.Println("GO_ENV is not set, defaulting to test")
		GoEnv = "test"
	}

	awsConfig := &AwsConfig{AccessKeyId: "", SecretAccessKey: "", Region: "USWest",
		BucketName: "lfs-server-go-objects", BucketAcl: "bucket-owner-full-control"}
	ldapConfig := &LdapConfig{Enabled: false, Server: "ldap://localhost:1389", Base: "dc=testers,c=test,o=company",
		UserObjectClass: "(&(objectclass=person))", UserCn: "uid"}
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
		Ldap:         ldapConfig,
		Aws:          awsConfig,
		Cassandra:    cassandraConfig,
		Redis:        &RedisConfig{Host: "localhost", Port: 6379, Password: "", DB: 0},
	}
	err = cfg.Section("Main").MapTo(configuration)
	err = cfg.Section("Aws").MapTo(configuration.Aws)
	err = cfg.Section("Ldap").MapTo(configuration.Ldap)
	err = cfg.Section("Cassandra").MapTo(configuration.Cassandra)
	err = cfg.Section("Redis").MapTo(configuration.Redis)
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
