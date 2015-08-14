// +build ignore

package main

import (
	"bytes"
	"fmt"
	log "github.com/cihub/seelog"
	"io/ioutil"
	"os"
	"text/template"
)

var logConfig = `
<seelog type="sync">
	<outputs formatid="main">
		<console/>
	</outputs>
	<formats>
		<format id="main" format="%Date(2006-02-01 03:04:05.000) - %Msg%n"/>
	</formats>
</seelog>`

func init() {
	logger, err := log.LoggerFromConfigAsString(logConfig)

	if err != nil {
		fmt.Printf("Could not load seelog configuration: %s\n", err)
		return
	}

	log.ReplaceLogger(logger)
}

type TypeInfo struct {
	Pre string
	Cql string
	Lit string
	Ex  string
}

var types = []TypeInfo{
	TypeInfo{Pre: "String", Cql: "text", Lit: "string", Ex: "\"x\""},
	TypeInfo{Pre: "Int32", Cql: "int", Lit: "int32", Ex: "1"},
	TypeInfo{Pre: "Int64", Cql: "bigint", Lit: "int64", Ex: "1"},
	TypeInfo{Pre: "Float32", Cql: "float", Lit: "float32", Ex: "1.1"},
	TypeInfo{Pre: "Float64", Cql: "double", Lit: "float64", Ex: "1.1"},
	TypeInfo{Pre: "Timestamp", Cql: "timestamp", Lit: "time.Time", Ex: "time.Now().UTC().Truncate(time.Millisecond)"},
	TypeInfo{Pre: "Timeuuid", Cql: "timeuuid", Lit: "gocql.UUID", Ex: "gocql.TimeUUID()"},
	TypeInfo{Pre: "Uuid", Cql: "uuid", Lit: "gocql.UUID", Ex: "gocql.TimeUUID()"},
	TypeInfo{Pre: "Boolean", Cql: "boolean", Lit: "bool", Ex: "true"},
	TypeInfo{Pre: "Decimal", Cql: "decimal", Lit: "*inf.Dec", Ex: "inf.NewDec(1,1)"},
	TypeInfo{Pre: "Bytes", Cql: "blob", Lit: "[]byte", Ex: "[]byte(\"x\")"},
}

func main() {
	params := make(map[string]interface{})
	params["types"] = types

	t, err := template.New("kitchen_sink.cql.tmpl").ParseFiles("tmpl/kitchen_sink.cql.tmpl")
	if err != nil {
		log.Errorf("Could not open template: %s", err)
		return
	}

	var b bytes.Buffer
	t.Execute(&b, params)

	if err := ioutil.WriteFile("kitchen_sink.cql", b.Bytes(), os.ModePerm); err != nil {
		log.Errorf("Could not write templated file: %s", err)
		return
	}

	log.Info("Regenerated test schema")

	t, err = template.New("kitchen_sink_data.go.tmpl").ParseFiles("tmpl/kitchen_sink_data.go.tmpl")
	if err != nil {
		log.Errorf("Could not open template: %s", err)
		return
	}

	b.Reset()
	t.Execute(&b, params)

	if err := ioutil.WriteFile("kitchen_sink_data.go", b.Bytes(), os.ModePerm); err != nil {
		log.Errorf("Could not write templated file: %s", err)
		return
	}

	log.Info("Regenerated test input data")
}
