package main

import (
	"fmt"
	"github.com/memikequinn/lfs-server-go/Godeps/_workspace/src/golang.org/x/crypto/bcrypt"
	"net/http"
	"reflect"
	"runtime"
)

func perror(err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(2)
		fmt.Println(fmt.Sprintf("%s:%d, error: %s", file, line, err.Error()))
		panic(err)
	}
}

func attributes(m interface{}) map[string]reflect.Type {
	typ := reflect.TypeOf(m)
	// if a pointer to a struct is passed, get the type of the dereferenced object
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// create an attribute data structure as a map of types keyed by a string.
	attrs := make(map[string]reflect.Type)
	// Only structs are supported so return an empty result if the passed object
	// isn't a struct
	if typ.Kind() != reflect.Struct {
		return attrs
	}

	// loop through the struct's fields and set the map
	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if !p.Anonymous {
			attrs[p.Name] = p.Type
		}
	}

	return attrs
}

func encryptPass(password []byte) (string, error) {
	// Hashing the password with the cost of 10
	hashedPassword, err := bcrypt.GenerateFromPassword(password, 10)
	return string(hashedPassword), err
}

func checkPass(hashedPassword, password []byte) (bool, error) {
	// Comparing the password with the hash
	err := bcrypt.CompareHashAndPassword(hashedPassword, password)
	// no error means success
	return (err == nil), nil
}

// cloneRequest returns a clone of the provided *http.Request. The clone is a
// shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}
	return r2
}
