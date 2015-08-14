package test

import (
	"bytes"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestKitchenSink(t *testing.T) {

	s := setup(t, "kitchen_sink")

	var buf bytes.Buffer
	fmt.Fprint(&buf, "INSERT INTO kitchen_sink (")

	colFragments := kitchenSinkColumns

	colClause := strings.Join(colFragments, ", ")
	fmt.Fprint(&buf, colClause)

	fmt.Fprint(&buf, ") VALUES (")

	placeHolderFragments := make([]string, len(colFragments))
	for i, _ := range placeHolderFragments {
		placeHolderFragments[i] = "?"
	}

	placeHolderClause := strings.Join(placeHolderFragments, ",")
	fmt.Fprint(&buf, placeHolderClause)
	fmt.Fprint(&buf, ")")

	insert := buf.String()

	if err := cqlr.Bind(insert, k).Exec(s); err != nil {
		t.Fatal(err)
	}

	buf.Reset()

	q := s.Query("SELECT * FROM kitchen_sink WHERE id = ?", k.Id)
	b := cqlr.BindQuery(q)

	var nk KitchenSink
	count := 0
	for b.Scan(&nk) {
		count++
	}

	assert.Equal(t, 1, count)
	assert.Equal(t, k, nk)
}

func setup(t *testing.T, table string) *gocql.Session {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "cqlr"
	s, err := cluster.CreateSession()

	assert.Nil(t, err, "Could not connect to keyspace")

	if err := s.Query(fmt.Sprintf("TRUNCATE %s", table)).Exec(); err != nil {
		t.Fatal(err)
	}

	return s
}
