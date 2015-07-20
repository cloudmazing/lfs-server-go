package main
import (
	"strings"
	"encoding/base64"
	"errors"
)
// authenticate uses the authorization string to determine whether
// or not to proceed. This server assumes an HTTP Basic auth format.
func authenticate(authorization string) bool {
	if Config.IsPublic() {
		return true
	}

	if authorization == "" {
		return false
	}

	if !strings.HasPrefix(authorization, "Basic ") {
		return false
	}

	c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authorization, "Basic "))
	if err != nil {
		return false
	}
	cs := string(c)
	i := strings.IndexByte(cs, ':')
	if i < 0 {
		return false
	}
	user, password := cs[:i], cs[i+1:]
	return LdapBind(user, password)
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
