# sqlice
[![Go Reference](https://pkg.go.dev/badge/github.com/pixelrazor/sqlice.svg)](https://pkg.go.dev/github.com/pixelrazor/sqlice) 
[![Go Report Card](https://goreportcard.com/badge/github.com/pixelrazor/sqlice)](https://goreportcard.com/report/github.com/pixelrazor/sqlice) 
[![Build Status](https://github.com/pixelrazor/sqlice/actions/workflows/build.yml/badge.svg)](https://github.com/pixelrazor/sqlice/actions/workflows/build.yml)

sqlice extends the functinailty of [squirrel](https://github.com/Masterminds/squirrel) and allows you to your database filtering on slices!
This makes it easy to:
 - provide filtering to functions not backed by database storage, while keeping it consistent with your functions that do
 - easily mock out your database interfaces and have your filters work the same

sqlice supports the following filters from the squirrel package:afadfaqwerqwrd
 - Eq
 - NotEq
 - Lt
 - LtOrEq
 - Gt
 - GtOrEq
 - And
 - Or
 - Like
 - NotLike
 - ILike
 - NotILike

 ## Usage

 sqlice will use the name of the struct fields to match with the columns/keys in the filters.

```go
type FooBar struct {
    A int
}

values := []FooBar{{A: 4}, {A: 2}, {A: 1}, {A: 3}}
var filteredValues []FooBar
err := sqlice.Filter(values, &filteredValues, Squirrel.Gt{"A": 2})
if err != nil {
    panic(err)
}
fmt.Println(filteredValues) // {{A: 4}, {A: 3}}
```

The comparison is case insensitive. The following examples shows comparing against a custom named field

```go
type FooBar struct {
    A int `db:"myColumnName"`
}

values := []FooBar{{A: 4}, {A: 2}, {A: 1}, {A: 3}}
var filteredValues []FooBar
err := sqlice.Filter(values, &filteredValues, Squirrel.Lt{"myColumnName": 3})
if err != nil {
    panic(err)
}
fmt.Println(filteredValues) // {{A: 2}, {A: 1}}
```

The struct tag used by this package is identical to the ones used by [sqlx](https://github.com/jmoiron/sqlx). This is done intentionally to help
ensure that you can use sqlice without needing to do any modifications to your structs

 ## Extending your own filters

 To use your own Sqlizers with sqlice, just add the `FilterValue(interface{})bool` method to your type!
