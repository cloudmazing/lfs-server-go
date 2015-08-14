cqlr
----

[![Build Status](https://travis-ci.org/relops/cqlr.png?branch=master)](https://travis-ci.org/relops/cqlr)

cqlr extends the [gocql][] runtime API and adds the ability to auto-bind a CQL iterator to a struct:

```go
type Tweet struct {
	Timeline string     `cql:"timeline"`
	Id       gocql.UUID `cql:"id"`
	Text     string     `cql:"text"`
}

var s *gocql.Session

q := s.Query(`SELECT text, id, timeline FROM tweet WHERE timeline = ?`, "me")
b := cqlr.BindQuery(q)

var t Tweet
for b.Scan(&t) {
	// Application specific code goes here
}
```

You can also bind structs to INSERT statements:

```go
tw := Tweet{
	Timeline: "me",
	Id:       gocql.TimeUUID(),
	Text:     "some random message",
}

var s *gocql.Session

b := Bind(`INSERT INTO tweet (timeline, id, text) VALUES (?, ?, ?)`, tw)
if err := b.Exec(s); err != nil {
	// .....
}
```

## Supported CQL Operations

* SELECT with `BindQuery()`, `Bind()` and `Scan()`
* INSERT, UPDATE and DELETE with `Bind()`
* Maps and Lists (without deltafication)

## Not Yet Supported CQL Operations

* CAS
* Counters
* Sets

## Feature Roadmap

(In no particular order of priority)

* Support for all CQL operations
* Batching
* Re-binding new struct instances to existing binding instances
* Protoype guess-based CQL CRUD  
* Investigate implementing the `skip metadata flag in EXECUTE` with the gocql driver
* Consider exposing a low level binary interface in gocql that handles query argument marshaling

## Supported Binding Mechanisms

Right now, cqlr supports the following mechanisms to bind iterators:

* Application supplied binding function
* Map of column name to struct field name
* By struct tags
* By field names

## Cassandra Support

Right now cqlr is known to work against Cassandra 2.0.9.

## Motivation

gocql users are looking for ways to automatically bind query results to application defined structs, but this functionality is not available in the core library. In addition, it is possible that the core library does not want to support this feature, because it significantly increases the functional scope of that codebase. So the goal of cqlr is to see if this functionality can be layered on top of the core gocql API in a re-useable way.

## Design

cqlr should sit on top of the core gocql runtime and concern itself only with struct binding. There are two modes of operation:

* Binding a struct's fields to the query parameters of a particular statement using `Bind()`
* Wrapping a gocql `*Query` instance to perform runtime introspection of the target struct in conjunction with meta data provided by the query result

The binding is specifically stateful so that down the line, the first loop execution can perform expensive introspection and subsequent loop invocations can benefit from this cached runtime metadata. So in a sense, it is a bit like [cqlc][], except that the metadata processing is done on the first loop, rather than at compile time.

## Status

Right now this is an experiment to try to come up with a design that people think is useful and can be implemented sanely.

[gocql]: https://github.com/gocql/gocql
[cqlc]: https://github.com/relops/cqlc
