package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"io/ioutil"
)

// RequestVars contain variables from the HTTP request. Variables from routing, json body decoding, and
// some headers are stored.
type RequestVars struct {
	Oid           string
	Size          int64
	User          string
	Password      string
	Namespace     string
	Repo          string
	Authorization string
}

type BatchVars struct {
	Objects []*RequestVars `json:"objects"`
}

// MetaObject is object metadata as seen by the object and metadata stores.
type MetaObject struct {
	Oid          string   `json:"oid" cql:"oid"`
	Size         int64    `json:"size "cql:"size"`
	ProjectNames []string `json:"project_names"`
	Existing     bool
}

// MetaProject is project metadata
type MetaProject struct {
	Name string   `json:"name" cql:"name"`
	Oids []string `json:"oids" cql:"oids"`
}

// Representation is object metadata as seen by clients of the lfs server.
type Representation struct {
	Oid   string           `json:"oid"`
	Size  int64            `json:"size"`
	Links map[string]*link `json:"_links"`
}

// MetaUser encapsulates information about a meta store user
type MetaUser struct {
	Name     string `cql:"username"`
	Password string ` cql:"password"`
}

// Wrapper for MetaStore so we can use different types
type GenericMetaStore interface {
	Put(v *RequestVars) (*MetaObject, error)
	Get(v *RequestVars) (*MetaObject, error)
	Close()
	DeleteUser(user string) error
	AddUser(user, pass string) error
	AddProject(projectName string) error
	Users() ([]*MetaUser, error)
	Objects() ([]*MetaObject, error)
	Projects() ([]*MetaProject, error)
}

type GenericContentStore interface {
	Get(meta *MetaObject) (io.Reader, error)
	Put(meta *MetaObject, r io.Reader) error
	Exists(meta *MetaObject) bool
}

// ObjectLink builds a URL linking to the object.
func (v *RequestVars) ObjectLink() string {
	path := fmt.Sprintf("/%s/%s/objects/%s", v.Namespace, v.Repo, v.Oid)

	if Config.IsHTTPS() {
		return fmt.Sprintf("%s://%s%s", Config.Scheme, Config.Host, path)
	}

	return fmt.Sprintf("http://%s%s", Config.Host, path)
}

// link provides a structure used to build a hypermedia representation of an HTTP link.
type link struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header,omitempty"`
}

// App links a Router, ContentStore, and MetaStore to provide the LFS server.
type App struct {
	router       *mux.Router
	contentStore GenericContentStore
	metaStore    GenericMetaStore
}

// NewApp creates a new App using the ContentStore and MetaStore provided
func NewApp(content GenericContentStore, meta GenericMetaStore) *App {
	app := &App{contentStore: content, metaStore: meta}

	r := mux.NewRouter()

	r.HandleFunc("/{namespace}/{repo}/objects/batch", app.BatchHandler).Methods("POST").MatcherFunc(MetaMatcher)
	route := "/{namespace}/{repo}/objects/{oid}"
	r.HandleFunc(route, app.GetContentHandler).Methods("GET", "HEAD").MatcherFunc(ContentMatcher)
	r.HandleFunc(route, app.GetMetaHandler).Methods("GET", "HEAD").MatcherFunc(MetaMatcher)
	r.HandleFunc("/search/{oid}", app.GetSearchHandler).Methods("GET")
	r.HandleFunc(route, app.PutHandler).Methods("PUT").MatcherFunc(ContentMatcher)

	r.HandleFunc("/{namespace}/{repo}/objects", app.PostHandler).Methods("POST").MatcherFunc(MetaMatcher)
	app.addMgmt(r)
	app.router = r

	return app
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err == nil {
		context.Set(r, "RequestID", fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))
	}

	a.router.ServeHTTP(w, r)
}

// Serve calls http.Serve with the provided Listener and the app's router
func (a *App) Serve(l net.Listener) error {
	return http.Serve(l, a)
}

// GetContentHandler gets the content from the content store
func (a *App) GetContentHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.metaStore.Get(rv)
	if err != nil {
		logger.Log(kv{"fn": "GetContentHandler", "error": err.Error()})
		if isAuthError(err) {
			requireAuth(w, r)
		} else {
			writeStatus(w, r, 404)
		}
		return
	}

	content, err := a.contentStore.Get(meta)
	if err != nil {
		writeStatus(w, r, 404)
		return
	}

	io.Copy(w, content)
	logRequest(r, 200)
}

// GetSearchHandler (search handler used by pre-push hooks)
func (a *App) GetSearchHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.metaStore.Get(rv)
	logger.Log(kv{"fn": "GetSearchHandler", "meta": err})
	if err != nil {
		if isAuthError(err) {
			requireAuth(w, r)
		} else {
			writeStatus(w, r, 404)
		}
		return
	}

	logger.Log(kv{"fn": "GetSearchHandler", "meta": meta})
	writeStatus(w, r, 200)
}

// Deep read, starting at path
func readDir(path string) []string {
	contents, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}
	var files []string
	for _, bl := range contents {
		if !bl.IsDir() {
			files = append(files, bl.Name())
		} else {
			nd := fmt.Sprintf("%s", path+"/"+bl.Name())
			for _, x := range readDir(nd) {
				files = append(files, fmt.Sprintf("%s/%s", bl.Name(), x))
			}
		}
	}
	return files
}

// GetMetaHandler retrieves metadata about the object
func (a *App) GetMetaHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.metaStore.Get(rv)
	if err != nil {
		if isAuthError(err) {
			requireAuth(w, r)
		} else {
			writeStatus(w, r, 404)
		}
		return
	}

	w.Header().Set("Content-Type", metaMediaType)

	if r.Method == "GET" {
		enc := json.NewEncoder(w)
		enc.Encode(a.Represent(rv, meta, true, false))
	}

	logRequest(r, 200)
}

// PostHandler instructs the client how to upload data
func (a *App) PostHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.metaStore.Put(rv)

	if err != nil {
		if isAuthError(err) {
			requireAuth(w, r)
		} else {
			writeStatus(w, r, 404)
		}
		return
	}

	w.Header().Set("Content-Type", metaMediaType)

	sentStatus := 202
	if meta.Existing && a.contentStore.Exists(meta) {
		sentStatus = 200
	}
	w.WriteHeader(sentStatus)

	enc := json.NewEncoder(w)
	enc.Encode(a.Represent(rv, meta, meta.Existing, true))
	logRequest(r, sentStatus)
}

// BatchHandler provides the batch api
func (a *App) BatchHandler(w http.ResponseWriter, r *http.Request) {
	bv := unpackbatch(r)

	var responseObjects []*Representation

	// Create a response object
	for _, object := range bv.Objects {
		meta, err := a.metaStore.Get(object)
		if err == nil { // Object is found
			responseObjects = append(responseObjects, a.Represent(object, meta, true, false))
			continue
		}

		if isAuthError(err) {
			requireAuth(w, r)
			return
		}

		// Object is not found
		meta, err = a.metaStore.Put(object)
		if err == nil {
			responseObjects = append(responseObjects, a.Represent(object, meta, meta.Existing, true))
		}
	}

	w.Header().Set("Content-Type", metaMediaType)

	type ro struct {
		Objects []*Representation `json:"objects"`
	}

	respobj := &ro{responseObjects}

	enc := json.NewEncoder(w)
	enc.Encode(respobj)
	logRequest(r, 200)
}

// PutHandler receives data from the client and puts it into the content store
func (a *App) PutHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.metaStore.Get(rv)
	if err != nil {
		if isAuthError(err) {
			requireAuth(w, r)
		} else {
			writeStatus(w, r, 404)
		}
		return
	}

	if err := a.contentStore.Put(meta, r.Body); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, `{"message":"%s"}`, err)
		return
	}

	logRequest(r, 200)
}

// Represent takes a RequestVars and Meta and turns it into a Representation suitable
// for json encoding
func (a *App) Represent(rv *RequestVars, meta *MetaObject, download, upload bool) *Representation {
	rep := &Representation{
		Oid:   meta.Oid,
		Size:  meta.Size,
		Links: make(map[string]*link),
	}

	header := make(map[string]string)
	header["Accept"] = contentMediaType
	if !Config.IsPublic() {
		header["Authorization"] = rv.Authorization
	}
	if download {
		rep.Links["download"] = &link{Href: rv.ObjectLink(), Header: header}
	}

	if upload {
		rep.Links["upload"] = &link{Href: rv.ObjectLink(), Header: header}
	}
	return rep
}

// ContentMatcher provides a mux.MatcherFunc that only allows requests that contain
// an Accept header with the contentMediaType
func ContentMatcher(r *http.Request, m *mux.RouteMatch) bool {
	mediaParts := strings.Split(r.Header.Get("Accept"), ";")
	mt := mediaParts[0]
	return mt == contentMediaType
}

// MetaMatcher provides a mux.MatcherFunc that only allows requests that contain
// an Accept header with the metaMediaType
func MetaMatcher(r *http.Request, m *mux.RouteMatch) bool {
	mediaParts := strings.Split(r.Header.Get("Accept"), ";")
	mt := mediaParts[0]
	return mt == metaMediaType
}

func unpack(r *http.Request) *RequestVars {
	vars := mux.Vars(r)
	rv := &RequestVars{
		Namespace:     vars["namespace"],
		Repo:          vars["repo"],
		Oid:           vars["oid"],
		Authorization: r.Header.Get("Authorization"),
	}

	if r.Method == "POST" { // Maybe also check if +json
		var p RequestVars
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&p)
		if err != nil {
			return rv
		}

		rv.Oid = p.Oid
		rv.Size = p.Size
	}

	return rv
}

// TODO cheap hack, unify with unpack
func unpackbatch(r *http.Request) *BatchVars {
	vars := mux.Vars(r)

	var bv BatchVars

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&bv)
	if err != nil {
		return &bv
	}

	for i := 0; i < len(bv.Objects); i++ {
		bv.Objects[i].Namespace = vars["namespace"]
		bv.Objects[i].Repo = vars["repo"]
		bv.Objects[i].Authorization = r.Header.Get("Authorization")
	}

	return &bv
}

func writeStatus(w http.ResponseWriter, r *http.Request, status int) {
	message := http.StatusText(status)

	mediaParts := strings.Split(r.Header.Get("Accept"), ";")
	mt := mediaParts[0]
	if strings.HasSuffix(mt, "+json") {
		message = `{"message":"` + message + `"}`
	}

	w.WriteHeader(status)
	fmt.Fprint(w, message)
	logRequest(r, status)
}

func logRequest(r *http.Request, status int) {
	logger.Log(kv{"method": r.Method, "url": r.URL, "status": status, "request_id": context.Get(r, "RequestID")})
}

func isAuthError(err error) bool {
	type autherror interface {
		AuthError() bool
	}
	if ae, ok := err.(autherror); ok {
		return ae.AuthError()
	}
	return false
}

func requireAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Lfs-Authenticate", "Basic realm=lfs-server-go")
	writeStatus(w, r, 401)
}
