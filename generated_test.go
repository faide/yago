package yago_test

import (
	"testing"

	"github.com/m4rw3r/uuid"
	"github.com/orus-io/yago"
	"github.com/stretchr/testify/assert"
)

func getUUID() uuid.UUID {
	id, err := uuid.V4()
	if err != nil {
		panic(err)
	}
	return id
}

var (
	uuid1          = getUUID()
	SQLValuesTests = []struct {
		m      yago.Mapper
		s      yago.MappedStruct
		fields []string
		expect map[string]interface{}
	}{
		{
			NewPersonStructMapper(),
			&PersonStruct{ID: uuid1, FirstName: "John", LastName: "Reece"},
			[]string{PersonStructFirstName},
			map[string]interface{}{
				PersonStructIDColumnName:        uuid1,
				PersonStructFirstNameColumnName: "John",
			},
		},
		{
			NewPersonStructMapper(),
			&PersonStruct{FirstName: "John", LastName: "Reece"},
			[]string{PersonStructLastName},
			map[string]interface{}{
				PersonStructLastNameColumnName: "Reece",
			},
		},
	}
)

func TestSQLValues(t *testing.T) {
	for _, tt := range SQLValuesTests {
		assert.Equal(t, tt.expect, tt.m.SQLValues(tt.s, tt.fields...))
	}
}
