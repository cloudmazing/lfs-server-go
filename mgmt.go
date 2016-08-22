package main

import (
	"encoding/json"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"net/http"
	"strings"
)

var (
	cssBox      *rice.Box
	jsBox       *rice.Box
	templateBox *rice.Box
)

type pageData struct {
	Name       string
	Config     *Configuration
	ConfigDump map[string]interface{}
	Users      []*MetaUser
	Objects    []*MetaObject
	Projects   []*MetaProject
}

func (a *App) addMgmt(r *mux.Router) {
	r.HandleFunc("/mgmt", basicAuth(a.indexHandler)).Methods("GET")
	r.HandleFunc("/mgmt/objects", basicAuth(a.objectsHandler)).Methods("GET")
	r.HandleFunc("/mgmt/projects", basicAuth(a.projectsHandler)).Methods("GET")
	r.HandleFunc("/mgmt/addProject", basicAuth(a.addProject)).Methods("POST")
	r.HandleFunc("/mgmt/users", basicAuth(a.usersHandler)).Methods("GET")
	r.HandleFunc("/mgmt/add", basicAuth(a.addUserHandler)).Methods("POST")
	r.HandleFunc("/mgmt/del", basicAuth(a.delUserHandler)).Methods("POST")
	r.HandleFunc("/mgmt/searchOid", basicAuth(a.searchOidHandler)).Methods("GET")

	cssBox = rice.MustFindBox("mgmt/css")
	jsBox = rice.MustFindBox("mgmt/js")
	templateBox = rice.MustFindBox("mgmt/templates")
	r.HandleFunc("/mgmt/css/{file}", basicAuth(cssHandler))
	r.HandleFunc("/mgmt/js/{file}", basicAuth(jsHandler))
}

func cssHandler(w http.ResponseWriter, r *http.Request) {
	file := mux.Vars(r)["file"]
	f, err := cssBox.Open(file)
	if err != nil {
		writeStatus(w, r, 404)
		return
	}

	w.Header().Set("Content-Type", "text/css")

	io.Copy(w, f)
	f.Close()
}

func jsHandler(w http.ResponseWriter, r *http.Request) {
	file := mux.Vars(r)["file"]
	f, err := jsBox.Open(file)
	if err != nil {
		writeStatus(w, r, 404)
		return
	}

	w.Header().Set("Content-Type", "text/javascript")

	io.Copy(w, f)
	f.Close()
}

func basicAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if Config.AdminUser == "" || Config.AdminPass == "" {
			writeStatus(w, r, 404)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", "Basic realm=mgmt")
			writeStatus(w, r, 401)
			return
		}

		if user != Config.AdminUser || pass != Config.AdminPass {
			w.Header().Set("WWW-Authenticate", "Basic realm=mgmt")
			writeStatus(w, r, 401)
			return
		}

		h(w, r)
		logRequest(r, 200)
	}
}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	if isJson(r) {
		w.Header().Set("Content-Type", "application/json")
		_json, err := json.Marshal(pageData{Name: "index", Config: Config, ConfigDump: Config.DumpConfig()})
		if err != nil {
			writeStatus(w, r, 500)
		}
		w.Write(_json)
	} else {
		if err := render(w, "config.tmpl", pageData{Name: "index", Config: Config, ConfigDump: Config.DumpConfig()}); err != nil {
			writeStatus(w, r, 404)
		}
	}
}

func (a *App) searchOidHandler(w http.ResponseWriter, r *http.Request) {
	searchedOid := r.URL.Query().Get("oid")
	if len(searchedOid) < 1 {
		writeStatus(w, r, 404)
	}
	cs, err := FindMetaStore()
	if err != nil {
		writeStatus(w, r, 404)
	}
	oids, err := cs.Objects()
	if err != nil {
		writeStatus(w, r, 404)
	}
	for _, oid := range oids {
		if strings.Contains(oid.Oid, searchedOid) {
			w.Header().Set("Content-Type", "application/json")
			_json, err := json.Marshal(oid)
			if err != nil {
				writeStatus(w, r, 500)
			}
			w.Write(_json)
		}
	}
}

func (a *App) objectsHandler(w http.ResponseWriter, r *http.Request) {
	objects, err := a.metaStore.Objects()
	if err != nil {
		fmt.Fprintf(w, "Error retrieving objects: %s", err)
		return
	}
	if isJson(r) {
		// fmt.Println(r.Header)
		w.Header().Set("Content-Type", "application/json")
		_json, err := json.Marshal(objects)
		if err != nil {
			writeStatus(w, r, 500)
		}
		w.Write(_json)
	} else {
		if err := render(w, "objects.tmpl", pageData{Name: "objects", Objects: objects}); err != nil {
			writeStatus(w, r, 404)
		}
	}
}

func (a *App) projectsHandler(w http.ResponseWriter, r *http.Request) {
	projects, err := a.metaStore.Projects()
	if err != nil {
		fmt.Fprintf(w, "Error retrieving objects: %s", err)
		return
	}
	if isJson(r) {
		w.Header().Set("Content-Type", "application/json")
		_json, err := json.Marshal(projects)
		if err != nil {
			writeStatus(w, r, 500)
		}
		w.Write(_json)
	} else {
		if err := render(w, "projects.tmpl", pageData{Name: "projects", Projects: projects}); err != nil {
			writeStatus(w, r, 404)
		}
	}
}

func (a *App) usersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := a.metaStore.Users()
	if err != nil {
		fmt.Fprintf(w, "Error retrieving users: %s", err)
		return
	}

	if isJson(r) {
		w.Header().Set("Content-Type", "application/json")
		_json, err := json.Marshal(users)
		if err != nil {
			writeStatus(w, r, 500)
		}
		w.Write(_json)
	} else {
		if err := render(w, "users.tmpl", pageData{Name: "users", Users: users}); err != nil {
			writeStatus(w, r, 404)
		}
	}
}

func (a *App) addProject(w http.ResponseWriter, r *http.Request) {
	projectName := r.FormValue("name")
	if projectName == "" {
		fmt.Fprintf(w, "Invalid project name: %s", projectName)
		return
	}

	if err := a.metaStore.AddProject(projectName); err != nil {
		fmt.Fprintf(w, "Error adding project: %s", err)
		return
	}

	http.Redirect(w, r, "/mgmt/projects", 302)
}

func (a *App) addUserHandler(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("name")
	pass := r.FormValue("password")
	if user == "" || pass == "" {
		fmt.Fprintf(w, "Invalid username or password")
		return
	}

	if err := a.metaStore.AddUser(user, pass); err != nil {
		fmt.Fprintf(w, "Error adding user: %s", err)
		return
	}

	http.Redirect(w, r, "/mgmt/users", 302)
}

func (a *App) delUserHandler(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("name")
	if user == "" {
		fmt.Fprintf(w, "Invalid username")
		return
	}

	if err := a.metaStore.DeleteUser(user); err != nil {
		fmt.Fprintf(w, "Error deleting user: %s", err)
		return
	}

	http.Redirect(w, r, "/mgmt/users", 302)
}

func render(w http.ResponseWriter, tmpl string, data pageData) error {
	bodyString, err := templateBox.String("body.tmpl")
	if err != nil {
		return err
	}

	contentString, err := templateBox.String(tmpl)
	if err != nil {
		return err
	}

	t := template.Must(template.New("main").Parse(bodyString))
	t.New("content").Parse(contentString)

	return t.Execute(w, data)
}

func isJson(r *http.Request) bool {
	var isJson bool
	isJson = false
	for _, t := range r.Header["Accept"] {
		if strings.Contains(t, "application/json") {
			isJson = true
		}
	}
	return isJson
}
