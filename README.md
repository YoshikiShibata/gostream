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
- `Min`
- `Max`
- `Count`
- `AnyMatch`
- `AllMatch`
- `NoneMatch`
- `FindFirst`

With the Go2 Generic, an interface cannot provided so-called **generic method** (in Java terms): instead, this package provides top-level functions of which the first
parameter is a `Stream`:
 
- `Map`
- `FlatMap`
- `Distinct`
- `Sorted`
- `Reduce`
- `Collect`
- `CollectByCollector`
- `Empty`
- `Iterate`
- `IteratN`
- `Generate`
- `Concat`

For `CollectByCollector` function, following functions as a `Collector` are provided:

- `ToSliceCollector`
- `ToSetCollector`
- `JoiningCollector`
- `MappingCollector`
- `FlatMappingCollector`
- `FilteringCollector`
- `GroupingByCollector`
- `GroupingByToSliceCollector`
