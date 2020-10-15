# gostream

`gostream` package provides a Stream API similar to Java, using the Go2 
Generic. This package is an experimental implementation for me to learn the Go2 Generic.

**CAUTION: this package is under construction**

## `Stream` interface

`Stream` interface is a generic interface. As a factory of `Stream` interface,
following functions are available:

- `Of` function creates a `Stream` from a slice. 
- `Builder` can be used to create a `Stream` by adding elements.
- `FileLines` function returns a `Stream` of lines of a file.

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
- `FindAny`
- `Parallel`

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
- `Sum`

For `CollectByCollector` function, following functions as a `Collector` are provided:

- `ToSliceCollector`
- `ToSetCollector`
- `JoiningCollector`
- `MappingCollector`
- `FlatMappingCollector`
- `FilteringCollector`
- `GroupingByCollector`
- `GroupingByToSliceCollector`
- `PartitioningByToSliceCollector`
- `PartitioningByCollector`
- `ToMapCollector`
- `SummarizingCollector`
- `SummingCollector`
- `CountingCollector`
- `ReducingCollector`
- `ReducingToOptionalCollector`
- `MaxByCollector`
- `MinByCollector`
