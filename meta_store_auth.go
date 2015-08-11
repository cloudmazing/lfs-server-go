package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/mavricknz/ldap"
	"net/url"
	"strconv"
	"strings"
)

func ldapHost() *url.URL {
	_url, err := url.Parse(Config.Ldap.Server)
	perror(err)
	return _url
}

func NewLdapConnection() *ldap.LDAPConnection {
	lh := ldapHost()
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
	var ldapCon *ldap.LDAPConnection
	if strings.Contains(lh.String(), "ldaps") {
		ldapCon = ldap.NewLDAPSSLConnection(hoster[0], port(), &tls.Config{InsecureSkipVerify: true})
	} else {
		ldapCon = ldap.NewLDAPConnection(hoster[0], port())
	}
	err := ldapCon.Connect()
	perror(err)
	return ldapCon
}

func LdapSearch(search *ldap.SearchRequest) (*ldap.SearchResult, error) {
	ldapCon := NewLdapConnection()
	s, er := ldapCon.Search(search)
	defer ldapCon.Close()
	if er != nil {
		logger.Log(kv{"fn": "meta_store_auth.LdapSearch", "msg": fmt.Sprintf("LDAP ERR: %S", er.Error())})
		return nil, er
	}
	if s == nil {
		return nil, errNoLdapSearchResults
	}
	s.String()
	return s, er
}

// boolean bind request
func LdapBind(user string, password string) bool {
	ldapCon := NewLdapConnection()
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
		logger.Log(kv{"fn": "meta_store_auth", "msg": fmt.Sprintf("LDAP ERR: %S", err.Error())})
		return false
	}
	return LdapBind(dn, password)
}

func findUserDn(user string) (string, error) {
	fltr := fmt.Sprintf("(&(objectClass=%s)(%s=%s))", Config.Ldap.UserObjectClass, Config.Ldap.UserCn, user)
	//	m := fmt.Sprintf("LDAP Search Host '%s' Filter '%s' base '%s'\n", ldapHost().String(), fltr, Config.Ldap.Base)
	//	logger.Log(kv{"fn": "meta_store_auth.findUserDn", "msg": m})
	base := fmt.Sprintf("%s=%s,%s", Config.Ldap.UserCn, user, Config.Ldap.Base)
	search := &ldap.SearchRequest{
		BaseDN: base,
		Filter: fltr,
	}
	r, err := LdapSearch(search)
	if err != nil {
		logger.Log(kv{"fn": "meta_store_auth.findUserDn", "msg_error": err.Error()})
		return "", err
	}
	if len(r.Entries) > 0 {
		//		logger.Log(kv{"fn": "meta_store_auth.findUserDn", "Found DN": r.Entries[0].DN})
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
