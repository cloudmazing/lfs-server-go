package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/nmcclain/ldap"
	"net/url"
	"strconv"
	"strings"
)

func ldapHost() (*url.URL, error) {
	return url.Parse(Config.Ldap.Server)
}

func NewLdapConnection() (*ldap.Conn, error) {
	var err error
	lh, err := ldapHost()
	if err != nil {
		logger.Log(kv{"fn": "NewLdapConnection", "error": err.Error()})
	}
	hoster := strings.Split(lh.Host, ":")
	port := func() uint16 {
		if len(hoster) < 2 {
			return uint16(389)
		} else {
			var e error
			port, e := strconv.Atoi(hoster[1])
			if e != nil {
				panic(e)
			}
			return uint16(port)
		}
	}
	var ldapCon *ldap.Conn
	if strings.Contains(lh.String(), "ldaps") {
		ldapCon, err = ldap.DialTLS("tcp", fmt.Sprintf("%s:%d", hoster[0], port()), &tls.Config{InsecureSkipVerify: true})
	} else {
		ldapCon, err = ldap.Dial("tcp", fmt.Sprintf("%s:%d", hoster[0], port()))
	}
	if err != nil {
		logger.Log(kv{"fn": "NewLdapConnection", "error": err.Error()})
		return nil, err
	}
	return ldapCon, nil
}

func LdapSearch(search *ldap.SearchRequest) (*ldap.SearchResult, error) {
	ldapCon, err := NewLdapConnection()
	if err != nil {
		logger.Log(kv{"fn": "LdapSearch", "search error": err.Error()})
		return nil, err
	}
	s, err := ldapCon.Search(search)
	defer ldapCon.Close()
	if err != nil {
		logger.Log(kv{"fn": "meta_store_auth.LdapSearch", "error": err.Error()})
		return nil, err
	}
	if (len(Config.Ldap.BindDn) + len(Config.Ldap.BindPass)) > 0 {
		err = ldapCon.Bind(Config.Ldap.BindDn, Config.Ldap.BindPass)
		if err != nil {
			logger.Log(kv{"fn": "LdapSearch", "Bind error": err.Error()})
			return nil, err
		}
	}
	if len(s.Entries) == 0 {
		return nil, errNoLdapSearchResults
	}
	return s, err
}

// boolean bind request
func LdapBind(user string, password string) bool {
	ldapCon, err := NewLdapConnection()
	if err != nil {
		logger.Log(kv{"fn": "LdapBind", "error": err.Error()})
		return false
	}
	reqE := ldapCon.Bind(user, password)
	defer ldapCon.Close()
	resp := false
	if reqE == nil {
		resp = true
	}
	return resp
}

// authenticate uses the authorization string to determine whether
// or not to proceed. This server assumes an HTTP Basic auth format.
func authenticateLdap(user, password string) bool {
	dn, err := findUserDn(user)
	if err != nil {
		logger.Log(kv{"fn": "meta_store_auth", "error": err.Error()})
		return false
	}
	return LdapBind(dn, password)
}

func findUserDn(user string) (string, error) {
	//	fmt.Printf("Looking for user '%s'\n", user)
	fltr := fmt.Sprintf("(&(objectclass=%s)(%s=%s))", Config.Ldap.UserObjectClass, Config.Ldap.UserCn, user)
	//	m := fmt.Sprintf("LDAP Search \"ldapsearch -x -H '%s' -b '%s' '%s'\"\n", Config.Ldap.Server, Config.Ldap.Base, fltr)
	//	logger.Log(kv{"fn": "meta_store_auth.findUserDn", "msg": m})
	search := &ldap.SearchRequest{
		BaseDN:     Config.Ldap.Base,
		Filter:     fltr,
		Scope:      1,
		Attributes: []string{"dn"},
	}
	r, err := LdapSearch(search)
	if err != nil {
		logger.Log(kv{"fn": "meta_store_auth.findUserDn", "msg_error": err.Error()})
		return "", err
	}
	if len(r.Entries) > 0 {
		return r.Entries[0].DN, nil
	}
	return "", errLdapUserNotFound
}

type authError struct {
	error
}

func (e authError) AuthError() bool {
	return true
}

func newAuthError() error {
	return authError{errors.New("Forbidden")}
}
