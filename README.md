# gostream

`gostream` package provides a Stream API similar to Java, using the Go2 
Generic. This package is an experimental implementation for me to learn the Go2 Generic.

**CAUTION: this package is under construction**

## `Stream` interface

`Stream` interface is a generic interface. As a factory of `Stream` interface,
`Of` function creates a `Stream` from a slice. Or you can use `Builder` to 
create a `Stream`.

`Stream` provides following methods:

- `Filter`
- `Sorted`
- `Peek`
- `Limit`
- `Skip`
- `ForEach`
- `ToSlice`
- `Reduce`
- `ReduceToOptional`
- `Count`
- `AnyMatch`
- `AllMatch`
- `NoneMatch`

With the Go2 Generic, an interface cannot provided so-called **generic method** (in Java terms): instead, this package provides top-level functions of which the first
parameter is a `Stream`:
 
- `Map`
- `FlatMap`
- `Distinct`
- `Sorted`
- `Reduce`
- `Empty`
- `Iterate`
- `IteratN`
- `Generate`
- `Concat`
