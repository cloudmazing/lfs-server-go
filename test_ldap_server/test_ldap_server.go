// Just returns success when username = user and password = password
// Otherwise, the response is a failure
// used for testing
package main

import (
	"github.com/memikequinn/lfs-server-go/Godeps/_workspace/src/github.com/nmcclain/ldap"
	"log"
	"net"
	"strings"
)

func main() {
	s := ldap.NewServer()
	handler := ldapHandler{}
	searchHandler := searchSimple{}
	log.Println("Starting server on localhost:1389")
	s.BindFunc("", handler)
	s.SearchFunc("", searchHandler)
	if err := s.ListenAndServe("localhost:1389"); err != nil {
		log.Fatal("LDAP Server Failed: %s", err.Error())
	}
}

type ldapHandler struct {
}

type searchSimple struct {
}

func (h ldapHandler) Bind(bindDN, bindSimplePw string, conn net.Conn) (ldap.LDAPResultCode, error) {
	log.Printf("BindDN: %s, bindSimplePw: %s\n", bindDN, bindSimplePw)
	s := searchSimple{}
	req := ldap.SearchRequest{BaseDN: "o=testers,o=company", Filter: "(cn=admin*)"}
	result, err := s.Search(bindDN, req, conn)
	if err != nil {
		log.Println("Unable to find user", bindDN)
		return ldap.LDAPResultInvalidCredentials, err
	}
	found := false
	for i := 0; i < len(result.Entries); i++ {
		if strings.Contains(result.Entries[i].DN, bindDN) {
			log.Println("Found DN", result.Entries[i].DN)
			found = true
		}
	}
	if found == false {
		log.Println("Unable to find result for DN", bindDN)
		return ldap.LDAPResultInvalidCredentials, nil
	}
	if bindSimplePw == "admin" {
		log.Printf("Authorized: BindDN: %s, bindSimplePw: %s\n", bindDN, bindSimplePw)
		return ldap.LDAPResultSuccess, nil
	}
	log.Printf("Unauthorized: BindDN: %s, bindSimplePw: %s\n", bindDN, bindSimplePw)
	return ldap.LDAPResultInvalidCredentials, nil
}

func (s searchSimple) Search(boundDN string, searchReq ldap.SearchRequest, conn net.Conn) (ldap.ServerSearchResult, error) {
	log.Println("Searching")
	entries := []*ldap.Entry{
		&ldap.Entry{"cn=ned,o=testers,o=company", []*ldap.EntryAttribute{
			&ldap.EntryAttribute{"cn", []string{"ned"}},
			&ldap.EntryAttribute{"o", []string{"ate"}},
			&ldap.EntryAttribute{"uidNumber", []string{"5000"}},
			&ldap.EntryAttribute{"accountstatus", []string{"active"}},
			&ldap.EntryAttribute{"uid", []string{"ned"}},
			&ldap.EntryAttribute{"description", []string{"ned via sa"}},
			&ldap.EntryAttribute{"objectclass", []string{"posixaccount"}},
		}},
		&ldap.Entry{"cn=admin,o=testers,o=company", []*ldap.EntryAttribute{
			&ldap.EntryAttribute{"cn", []string{"admin"}},
			&ldap.EntryAttribute{"o", []string{"ate"}},
			&ldap.EntryAttribute{"uidNumber", []string{"5001"}},
			&ldap.EntryAttribute{"accountstatus", []string{"active"}},
			&ldap.EntryAttribute{"uid", []string{"admin"}},
			&ldap.EntryAttribute{"description", []string{"admin via sa"}},
			&ldap.EntryAttribute{"objectclass", []string{"posixaccount", "user"}},
		}},
		&ldap.Entry{"cn=trent,o=testers,o=company", []*ldap.EntryAttribute{
			&ldap.EntryAttribute{"cn", []string{"trent"}},
			&ldap.EntryAttribute{"o", []string{"ate"}},
			&ldap.EntryAttribute{"uidNumber", []string{"5005"}},
			&ldap.EntryAttribute{"accountstatus", []string{"active"}},
			&ldap.EntryAttribute{"uid", []string{"trent"}},
			&ldap.EntryAttribute{"description", []string{"trent via sa"}},
			&ldap.EntryAttribute{"objectclass", []string{"posixaccount"}},
		}},
		&ldap.Entry{"cn=randy,o=testers,o=company", []*ldap.EntryAttribute{
			&ldap.EntryAttribute{"cn", []string{"randy"}},
			&ldap.EntryAttribute{"o", []string{"ate"}},
			&ldap.EntryAttribute{"uidNumber", []string{"5555"}},
			&ldap.EntryAttribute{"accountstatus", []string{"active"}},
			&ldap.EntryAttribute{"uid", []string{"randy"}},
			&ldap.EntryAttribute{"objectclass", []string{"posixaccount"}},
		}},
	}
	return ldap.ServerSearchResult{entries, []string{}, []ldap.Control{}, ldap.LDAPResultSuccess}, nil
}
