package sqlice_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/pixelrazor/sqlice"
)

func TestFilter(t *testing.T) {
	tests := map[string]struct {
		input, output  interface{}
		filter         squirrel.Sqlizer
		expectedOutput interface{}
	}{
		"nil filter": {
			input:          []struct{ A int }{{A: 1}, {A: 2}, {A: 3}},
			output:         &[]struct{ A int }{},
			expectedOutput: &[]struct{ A int }{{A: 1}, {A: 2}, {A: 3}},
		},
		"Eq": {
			input: []struct {
				A int
				B string `db:"bar"`
			}{{A: 1, B: "one"}, {A: 2, B: "two"}, {A: 3, B: "three"}},
			output: &[]struct {
				A int
				B string `db:"bar"`
			}{},
			expectedOutput: &[]struct {
				A int
				B string `db:"bar"`
			}{{A: 2, B: "two"}},
			filter: squirrel.Eq{"A": 2, "bar": "two"},
		},
		"And": {
			input: []struct {
				A int
				B string `db:"bar"`
			}{{A: 1, B: "one"}, {A: 2, B: "two"}, {A: 3, B: "three"}},
			output: &[]struct {
				A int
				B string `db:"bar"`
			}{},
			expectedOutput: &[]struct {
				A int
				B string `db:"bar"`
			}{{A: 2, B: "two"}},
			filter: squirrel.And{squirrel.Eq{"A": 2}, squirrel.Eq{"bar": "two"}},
		},
		"Or": {
			input: []struct {
				A int
				B string `db:"bar"`
			}{{A: 1, B: "one"}, {A: 2, B: "two"}, {A: 3, B: "three"}},
			output: &[]struct {
				A int
				B string `db:"bar"`
			}{},
			expectedOutput: &[]struct {
				A int
				B string `db:"bar"`
			}{{A: 1, B: "one"}, {A: 2, B: "two"}},
			filter: squirrel.Or{squirrel.Eq{"A": 2}, squirrel.Eq{"A": 1}},
		},
		"Gt": {
			input: []struct {
				A int
			}{{A: 1}, {A: 2}, {A: 3}},
			output: &[]struct {
				A int
			}{},
			expectedOutput: &[]struct {
				A int
			}{{A: 2}, {A: 3}},
			filter: squirrel.Gt{"A": 1},
		},
		"Lt": {
			input: []struct {
				B string `db:"bar"`
			}{{B: "a"}, {B: "b"}, {B: "c"}},
			output: &[]struct {
				B string `db:"bar"`
			}{},
			expectedOutput: &[]struct {
				B string `db:"bar"`
			}{{B: "a"}, {B: "b"}},
			filter: squirrel.Lt{"bar": "c"},
		},
		"LtOrEq": {
			input: []struct {
				A float64
			}{{A: 1}, {A: 2}, {A: 3}},
			output: &[]struct {
				A float64
			}{},
			expectedOutput: &[]struct {
				A float64
			}{{A: 1}, {A: 2}},
			filter: squirrel.LtOrEq{"A": 2.0},
		},
		"GtOrEq": {
			input: []struct {
				A int
				B string `db:"bar"`
			}{{A: 1, B: "one"}, {A: 2, B: "two"}, {A: 3, B: "three"}},
			output: &[]struct {
				A int
				B string `db:"bar"`
			}{},
			expectedOutput: &[]struct {
				A int
				B string `db:"bar"`
			}{{A: 2, B: "two"}, {A: 3, B: "three"}},
			filter: squirrel.GtOrEq{"A": 2},
		},
		"NotEq": {
			input: []struct {
				B []string `db:"bar"`
			}{{B: []string{"a1", "a2"}}, {B: []string{"a1", "a3"}}, {B: []string{"a1", "a4"}}},
			output: &[]struct {
				B []string `db:"bar"`
			}{},
			expectedOutput: &[]struct {
				B []string `db:"bar"`
			}{{B: []string{"a1", "a2"}}, {B: []string{"a1", "a4"}}},
			filter: squirrel.NotEq{"bar": []string{"a1", "a3"}},
		},
		"ValueFilterer": {
			input: []struct {
				B []string `db:"bar"`
			}{{B: []string{"a1", "a2"}}, {B: []string{"a1", "a3", "a5"}}, {B: []string{"a1", "a4"}}},
			output: &[]struct {
				B []string `db:"bar"`
			}{},
			expectedOutput: &[]struct {
				B []string `db:"bar"`
			}{{B: []string{"a1", "a3", "a5"}}},
			filter: sqlice.ValueFilterFunc(func(i interface{}) bool {
				v, ok := i.(struct {
					B []string `db:"bar"`
				})
				if !ok {
					return false
				}
				return len(v.B) == 3
			}),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := sqlice.Filter(test.input, test.output, test.filter)
			if err != nil {
				t.Fatal("Expected no error, got:", err)
			}
			if !reflect.DeepEqual(test.output, test.expectedOutput) {
				t.Errorf("Expected '%v' got '%v'", test.expectedOutput, test.output)
			}
		})
	}
}

func TestFilter_ErrorConditions(t *testing.T) {
	tests := map[string]struct {
		input, output interface{}
		filter        squirrel.Sqlizer
		err           error // if this is nil, the test just insures that any error occured
	}{
		"nil input and output": {},
		"nil input": {
			output: &[]struct{}{},
			filter: squirrel.Eq{"testColumn": "value"},
		},
		"nil output": {
			input:  []struct{}{},
			filter: squirrel.Eq{"testColumn": "value"},
		},
		"input not slice": {
			input:  "not a slice",
			output: &[]struct{}{},
			filter: squirrel.Eq{"testColumn": "value"},
		},
		"output not slice pointer": {
			input:  []struct{}{},
			output: []struct{}{},
			filter: squirrel.Eq{"testColumn": "value"},
		},
		"input/output type mismatch": {
			input:  []struct{ A int }{},
			output: &[]struct{ B string }{},
			filter: squirrel.Eq{"testColumn": "value"},
		},
		"input slice type not sortable": {
			input:  []string{},
			output: &[]string{},
			filter: squirrel.Eq{"A": 3},
		},
		"filter field not in struct": {
			input:  []struct{ A int }{{A: 1}, {A: 2}, {A: 3}},
			output: &[]struct{ A int }{},
			filter: squirrel.Eq{"testColumn": "value"},
		},
		"unexported field ignored": {
			input:  []struct{ A, b int }{{A: 1}, {A: 2}, {A: 3}},
			output: &[]struct{ A, b int }{},
			filter: squirrel.Eq{"b": 5},
		},
		"filter field wrong type 1": {
			input:  []struct{ A int }{{A: 1}, {A: 2}, {A: 3}},
			output: &[]struct{ A int }{},
			filter: squirrel.Eq{"A": "value"},
		},
		"filter field wrong type 2": {
			input:  []struct{ A int }{{A: 1}, {A: 2}, {A: 3}},
			output: &[]struct{ A int }{},
			filter: squirrel.Eq{"A": 1.2},
		},
		"filter field wrong type 3": {
			input:  []struct{ A []string }{},
			output: &[]struct{ A []string }{},
			filter: squirrel.Eq{"A": []int{}},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := sqlice.Filter(test.input, test.output, test.filter)
			if test.err != nil && !errors.Is(err, test.err) {
				t.Fatalf("Expected %v, got %v", test.err, err)
			} else if err == nil {
				t.Fatal("Expected an error, got nil")
			}
		})
	}
}
