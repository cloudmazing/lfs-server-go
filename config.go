package main

import (
	"fmt"
	"github.com/vaughan0/go-ini"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

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
	Public       string `config:"public"`
	MetaDB       string `config:"lfs-test.db"`
	RedisUrl     string `config:"redis://localhost:6379/0"`
	BackingStore string `config:"bolt"`
	LdapServer   string `config:"ldap://localhost:1389"`
	LdapBase     string `config:"dc=testers,c=test,o=company"`
	CassandraHosts     string `config:"localhost"` //separated by ","
	CassandraKeyspace     string `config:"lfs_server_go"`
}

type RedisConfigT struct {
	Addr     string
	Password string
	DB       int64
}

func (c *Configuration) IsHTTPS() bool {
	return strings.Contains(Config.Scheme, "https")
}

func (c *Configuration) IsPublic() bool {
	t, _ := strconv.ParseBool(Config.Public)
	return t
}

// Config is the global app configuration
//var Config = &Configuration{}
var GoEnv = os.Getenv("GO_ENV")
var Config = &Configuration{}
var RedisConfig = &RedisConfigT{}

// iterate thru config.yaml and parse it
// always called when initializing Config
func init() {
	file, err := ini.LoadFile("config.ini")
	if err != nil {
		panic(fmt.Sprint("unable to read config.ini, %v", err))
	}
	if GoEnv == "" {
		fmt.Println("GO_ENV is not set, defaulting to test")
		GoEnv = "test"
	}
	typeE := reflect.TypeOf(Config).Elem()
	valueE := reflect.ValueOf(Config).Elem()
	for i := 0; i < typeE.NumField(); i++ {
		sf := typeE.Field(i)
		name := sf.Name
		tag := sf.Tag.Get("config")
		field := valueE.FieldByName(name)
		// only do what has been declared in the config
		e, ok := file.Get(GoEnv, name)
		if !ok || e == "" {
			field.SetString(tag)
		} else {
			field.SetString(e)
		}
	}
	RedisConfig = setRedisConfig()
}

func setRedisConfig() *RedisConfigT {
	_url, err := url.Parse(Config.RedisUrl)
	perror(err)
	db, _ := strconv.ParseInt(strings.Replace(_url.Path, "/","", -1), 0, 0)
	addr := _url.Host
	var password string
	var ok bool
	if _url.User != nil {
		password, ok = _url.User.Password()
		if !ok {
			password = ""
		}
	}
	return &RedisConfigT{Addr: addr, DB: db, Password: password}
}

func dumpConfig() {
	file, err := ini.LoadFile("config.ini")
	if err != nil {
		panic(fmt.Sprint("unable to read config.ini, %v", err))
	}
	for name, section := range file {
		fmt.Printf("Section %s, name: %s\n", section, name)
		for subname := range section {
			fmt.Printf("Subname: %s\n", subname)
		}
	}

}
