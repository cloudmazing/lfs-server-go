package cqlr

import (
	"errors"
	"github.com/gocql/gocql"
	"reflect"
	"strings"
)

type Binding struct {
	err        error
	qry        *gocql.Query
	iter       *gocql.Iter
	stmt       string
	arg        interface{}
	isCompiled bool
	strict     bool
	strategy   map[string]reflect.Value
	fun        func(gocql.ColumnInfo) (reflect.StructField, bool)
	typeMap    map[string]string
	fieldMap   map[string][]int
}

func BindQuery(q *gocql.Query) *Binding {
	return &Binding{
		qry:      q,
		strategy: make(map[string]reflect.Value),
		fieldMap: make(map[string][]int),
	}
}

func Bind(s string, v interface{}) *Binding {
	return &Binding{
		stmt:     s,
		arg:      v,
		strategy: make(map[string]reflect.Value),
		fieldMap: make(map[string][]int),
	}
}

func (b *Binding) Bind(v interface{}) *Binding {
	if b.qry == nil {
		b.arg = v
	} else {
		b.qry.Bind(v)
	}
	return b
}

func (b *Binding) Exec(s *gocql.Session) error {
	return s.Bind(b.stmt, b.bind).Exec()
}

func (b *Binding) Query(s *gocql.Session) *gocql.Query {
	return s.Bind(b.stmt, b.bind)
}

func (b *Binding) Use(f func(gocql.ColumnInfo) (reflect.StructField, bool)) *Binding {
	b.fun = f
	return b
}

func (b *Binding) Map(m map[string]string) *Binding {
	b.typeMap = m
	return b
}

func (b *Binding) Strict() *Binding {
	b.strict = true
	return b
}

func (b *Binding) Close() error {
	if b.err != nil {
		return b.err
	}

	if err := b.iter.Close(); err != nil {
		return err
	}

	return nil
}

func (b *Binding) Scan(dest interface{}) bool {

	v := reflect.ValueOf(dest)

	if v.Kind() != reflect.Ptr || v.IsNil() {
		return false
	}

	if b.iter == nil {
		b.iter = b.qry.Iter()
	}

	cols := b.iter.Columns()
	if !b.isCompiled {
		if err := b.compile(v, cols); err != nil {
			b.err = err
			return false
		}
	}

	values := make([]interface{}, len(cols))

	for i, col := range cols {
		f, ok := b.strategy[col.Name]

		if ok {
			values[i] = f.Addr().Interface()
		}
	}

	return b.iter.Scan(values...)
}

func (b *Binding) bind(q *gocql.QueryInfo) ([]interface{}, error) {
	values := make([]interface{}, len(q.Args))
	value := reflect.ValueOf(b.arg)

	if !b.isCompiled {
		if err := b.compile(value, q.Args); err != nil {
			return nil, err
		}
	}

	for i, col := range q.Args {
		f, ok := b.strategy[col.Name]

		if b.strict && !ok {
			return nil, ErrMissingStrategy
		}

		if ok {
			if f.CanInterface() {
				values[i] = f.Interface()
			} else if f.CanAddr() {
				values[i] = f.Addr().Interface()
			}
		}

		// TODO Going forwards, passing a nil value to the gocql driver
		// might be a valid approach, but for now, we're going to try
		// avoid confusing people with reflect panics
		if values[i] == nil {
			return nil, ErrMissingStrategy
		}
	}

	return values, nil
}

func (b *Binding) compile(v reflect.Value, cols []gocql.ColumnInfo) error {

	indirect := reflect.Indirect(v)

	s := indirect.Type()

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		tag := f.Tag.Get("cql")
		if tag != "" {
			b.strategy[tag] = indirect.Field(i)
		} else {
			b.fieldMap[strings.ToLower(f.Name)] = f.Index
		}
	}

	if b.fun != nil {
		for _, col := range cols {
			staticField, ok := b.fun(col)
			if ok {
				b.strategy[col.Name] = indirect.FieldByIndex(staticField.Index)
			}
		}
	}

	if b.typeMap != nil && len(b.typeMap) > 0 {
		for _, col := range cols {
			fieldName, ok := b.typeMap[col.Name]
			if ok {
				f := indirect.FieldByName(fieldName)
				b.strategy[col.Name] = f
			}
		}
	}

	for _, col := range cols {

		_, ok := b.strategy[col.Name]
		if !ok {
			index, ok := b.fieldMap[col.Name]
			if !ok {
				index, ok = b.fieldMap[strings.ToLower(col.Name)]
			}
			if ok {
				f := indirect.FieldByIndex(index)
				if f.IsValid() {
					b.strategy[col.Name] = f
				}
			}
		}
	}

	if b.strict {
		if len(b.strategy) != len(cols) {
			return ErrMissingStrategy
		}
	}

	b.isCompiled = true

	return nil
}

var (
	ErrMissingStrategy = errors.New("insufficient column mapping")
)
