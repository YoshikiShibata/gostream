# gostream

`gostream` package provides a Stream API similar to Java, using Go generics.
Since Go 1.27 supports **generic methods** on concrete types, `Stream[T]` is
now a struct so that operations like `Map`/`FlatMap` can be chained in the
Java style.

**CAUTION**: This package was not intended for practical use. I created it to explore whether a Java-like Stream API could be implemented using Go generics, and I do not recommend using it in production.

## `Stream` struct

`Stream[T]` is a generic struct that wraps an internal implementation
(sequential `iter.Seq`-backed or parallel channel-backed). As a factory of
`Stream[T]`, the following functions are available:

- `Of` function creates a `Stream` from values.
- `Builder` can be used to create a `Stream` by adding elements.
- `FileLines` function returns a `Stream` of lines of a file.
- `Range` function returns a `Stream` by an incremental step of 1.
- `RangeClosed` function returns a `Stream` by an incremental step of 1.
- `Empty`, `Iterate`, `IterateN`, `Generate`, `Concat` also return a `Stream`.

`Stream[T]` provides the following methods (non-generic):

- `Filter`
- `Sorted`
- `Peek`
- `Limit`
- `Skip`
- `TakeWhile`
- `DropWhile`
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
- `Sequential`
- `IsParallel`
- `OnClose`
- `Close`

Thanks to Go 1.27 generic methods, `Stream[T]` also provides the following
methods whose type parameter differs from the element type `T`:

- `Map[R any](mapper Function[T, R]) Stream[R]`
- `FlatMap[R any](mapper Function[T, Stream[R]]) Stream[R]`
- `MapMulti[R any](mapper func(t T, emit func(R))) Stream[R]`
- `ReduceWith[U any](identity U, accumulator BiFunction[U, T, U], combiner BinaryOperator[U]) U`
- `Collect[R any](supplier Supplier[R], accumulator BiConsumer[R, T], combiner BiConsumer[R, R]) R`

The following top-level functions remain because their type parameters
cannot be expressed purely as a method on `Stream[T]` (they require an
additional constraint such as `comparable`, `cmp.Ordered`, or `Number`, or
they combine multiple `Stream`s):

- `Distinct` (constraint: `comparable`)
- `Sorted` (constraint: `cmp.Ordered`)
- `CollectByCollector`
- `Concat`
- `Sum`, `Min`, `Max` (constraint: `Number`)

## Examples

### Map, Filter, Reduce (Java-style chaining)

```go
// Sum of squares of even numbers in 1..10.
sum := gostream.RangeClosed(1, 10).
    Filter(func(n int) bool { return n%2 == 0 }).
    Map(func(n int) int { return n * n }).
    Reduce(0, func(a, b int) int { return a + b })
// sum == 220
```

### Type-changing pipeline (`Map[R]`)

```go
words := []string{"alpha", "beta", "gamma"}

lengths := gostream.Of(words...).
    Map(func(s string) int { return len(s) }). // Stream[string] -> Stream[int]
    ToSlice()
// lengths == [5 4 5]
```

### FlatMap

```go
runeStream := func(s string) gostream.Stream[rune] {
    return gostream.Of([]rune(s)...)
}

runes := gostream.Of("your", "boat").
    FlatMap(runeStream).
    ToSlice()
// runes == ['y' 'o' 'u' 'r' 'b' 'o' 'a' 't']
```

### MapMulti — emit 0/1/many outputs per input

```go
// Duplicate every element (2-to-many).
got := gostream.Of(1, 2, 3).
    MapMulti(func(v int, emit func(int)) {
        emit(v)
        emit(v * 10)
    }).
    ToSlice()
// got == [1 10 2 20 3 30]

// Filter via 0-or-1 emission (many-to-0-or-1). Cheaper than FlatMap
// because no intermediate Stream[R] is built per input.
odd := gostream.Of(1, 2, 3, 4, 5).
    MapMulti(func(v int, emit func(int)) {
        if v%2 == 1 {
            emit(v)
        }
    }).
    ToSlice()
// odd == [1 3 5]
```

### OnClose — register close handlers (e.g. for FileLines)

```go
// Close handlers run in registration order when Close() is called.
s := gostream.Of(1, 2, 3).
    OnClose(func() { log.Println("first") }).
    OnClose(func() { log.Println("second") })
defer s.Close()

_ = s.Filter(func(v int) bool { return v > 1 }).Count()
// After the terminal op, s.Close() (via defer) prints "first" then "second".
```

Handlers registered on a source stream are propagated to every derived
stream, so `defer s.Close()` on any point in the pipeline fires the
same handler set exactly once.

### TakeWhile / DropWhile

```go
// TakeWhile: prefix of elements matching the predicate.
prefix := gostream.Of(1, 2, 3, 4, 5, 1, 2).
    TakeWhile(func(v int) bool { return v < 4 }).
    ToSlice()
// prefix == [1 2 3]

// DropWhile: everything after the initial matching prefix.
rest := gostream.Of(1, 2, 3, 4, 5, 1, 2).
    DropWhile(func(v int) bool { return v < 4 }).
    ToSlice()
// rest == [4 5 1 2]
```

### Parallel execution

```go
// Count primes up to 1e6 in parallel.
primes := gostream.RangeClosed[int64](2, 1e6).
    Parallel().
    Map(func(i int64) *big.Int { return big.NewInt(i) }).
    Filter(func(i *big.Int) bool { return i.ProbablyPrime(1) }).
    Count()
```

### Sequential / IsParallel

```go
s := gostream.Of(1, 2, 3).Parallel()
_ = s.IsParallel() // true

// Switch back to sequential when a downstream operation must run in
// encounter order.
seq := s.Sequential()
_ = seq.IsParallel() // false
```

### `ReduceWith[U]` — accumulating into a different type

```go
// Total length of all words, computed in parallel.
totalLen := gostream.Of(words...).
    Parallel().
    ReduceWith(
        0, // identity of type int
        func(u int, s string) int { return u + len(s) }, // accumulator
        func(a, b int) int { return a + b },             // combiner
    )
```

### `Collect[R]` — mutable reduction into a container

```go
// Collect elements into a slice (via a *[]int container).
slice := gostream.Of(1, 2, 3, 4, 5).
    Collect(
        func() *[]int { return &[]int{} },
        func(r *[]int, v int) { *r = append(*r, v) },
        func(a, b *[]int) { *a = append(*a, *b...) },
    )
// *slice == [1 2 3 4 5]
```

### `CollectByCollector` — reusable collection strategies

```go
// Join integers as space-separated strings.
joined := gostream.CollectByCollector(
    gostream.Of(1, 2, 3).Map(strconv.Itoa),
    gostream.JoiningCollector(" "),
)
// joined == "1 2 3"
```

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
- `AveragingInt64Collector`
- `AveragingFloat64Collector`
