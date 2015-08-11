package main

import "errors"

var (
	errProjectNotFound     = errors.New("Project not found")
	errObjectNotFound      = errors.New("Object not found")
	errLdapUserNotFound    = errors.New("Unable to find user in LDAP")
	errUserNotFound        = errors.New("Unable to find user")
	errNoLdapSearchResults = errors.New("No results from LDAP")
	errLdapSearchFailed    = errors.New("Failed searching LDAP")
	errHashMismatch        = errors.New("Content has does not match OID")
	errSizeMismatch        = errors.New("Content size does not match")
	errWriteS3             = errors.New("Erred writing to S3")
	errNotImplemented      = errors.New("Not Implemented when using LDAP")
	errMissingParams       = errors.New("Missing params")
)

func isErrObjectNotFound(err error) bool {
	type errObjectNotFound interface {
		errObjectNotFound() bool
	}
	if ae, ok := err.(errObjectNotFound); ok {
		return ae.errObjectNotFound()
	}
	return false
}
func isErrProjectNotFound(err error) bool {
	type errProjectNotFound interface {
		errProjectNotFound() bool
	}
	if ae, ok := err.(errProjectNotFound); ok {
		return ae.errProjectNotFound()
	}
	return false
}
