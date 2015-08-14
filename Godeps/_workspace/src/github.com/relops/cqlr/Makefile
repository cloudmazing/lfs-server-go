test/kitchen_sink_data.go: test/tmpl/kitchen_sink_data.go.tmpl test/kitchen_sink_generator.go
	cd test; go run kitchen_sink_generator.go

test/kitchen_sink.cql: test/tmpl/kitchen_sink.cql.tmpl test/kitchen_sink_generator.go
	cd test; go run kitchen_sink_generator.go

schema: test/kitchen_sink.cql
	-cqlsh -f test/keyspace.cql
	cqlsh -k cqlr -f test/schema.cql
	cqlsh -k cqlr -f test/kitchen_sink.cql

test: test/kitchen_sink_data.go schema
	go test -v .
	go test -v ./test

sink: test/kitchen_sink_data.go schema
	go test -v ./test

test_data: test/kitchen_sink_data.go