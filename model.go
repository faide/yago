package yago

import (
	"github.com/aacanakin/qb"
)

// A Model provides handy access to a struct definition.
type Model interface {
	GetMapper() Mapper
}

// ScalarField A simple scalar field
type ScalarField struct {
	Column qb.ColumnElem
}

// NewScalarField returns a new ScalarField
func NewScalarField(column qb.ColumnElem) ScalarField {
	return ScalarField{
		Column: column,
	}
}

// Like returns a LIKE clause
func (f ScalarField) Like(pattern string) qb.Clause {
	return f.Column.Like(pattern)
}

// NotIn returns a NOT IN clause
func (f ScalarField) NotIn(values ...interface{}) qb.Clause {
	return f.Column.NotIn(values...)
}

// In returns a IN clause
func (f ScalarField) In(values ...interface{}) qb.Clause {
	return f.Column.In(values...)
}

// NotEq returns a != clause
func (f ScalarField) NotEq(value interface{}) qb.Clause {
	return f.Column.NotEq(value)
}

// Eq returns a = clause
func (f ScalarField) Eq(value interface{}) qb.Clause {
	return f.Column.Eq(value)
}

// Gt returns a > clause
func (f ScalarField) Gt(value interface{}) qb.Clause {
	return f.Column.Gt(value)
}

// St returns a < clause
func (f ScalarField) St(value interface{}) qb.Clause {
	return f.Column.St(value)
}

// Gte returns a >= clause
func (f ScalarField) Gte(value interface{}) qb.Clause {
	return f.Column.Gte(value)
}

// Ste returns a <= clause
func (f ScalarField) Ste(value interface{}) qb.Clause {
	return f.Column.Ste(value)
}