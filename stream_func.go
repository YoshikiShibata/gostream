// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"cmp"
	"sort"
	"sync"

	"github.com/YoshikiShibata/gostream/function"
)

// Map returns a stream consisting of the results of applying the given
// function to the elements of the given stream.
func Map[T, R any](stream Stream[T], mapper function.Function[T, R]) Stream[R] {
	gs := stream.(*genericStream[T])
	gs.validateState()

	nextReq := make(chan struct{})
	nextData := make(chan orderedData[R])

	closeCounter := gs.parallelCount
	var lock sync.Mutex

	closeChans := func() {
		lock.Lock()
		defer lock.Unlock()

		if closeCounter > 1 {
			closeCounter--
			return
		}

		close(nextData)
		close(gs.nextReq)
		go func() {
			for range nextReq {
			}
		}()
	}

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {
			for range nextReq {
				gs.nextReq <- struct{}{}
				od, ok := <-gs.nextData
				if !ok {
					closeChans()
					return
				}
				r := mapper(od.data)
				nextData <- orderedData[R]{
					order: od.order,
					data:  r,
				}
			}
		}()
	}

	return &genericStream[R]{
		parallel:      gs.parallel,
		parallelCount: parallelCount,
		nextReq:       nextReq,
		nextData:      nextData,
	}
}

// FlatMap returns a stream consisting of the results of replacing each
// element of stream with the contents of mapped stream produced by applying
// the provided mapping function to each element.
func FlatMap[T, R any](
	stream Stream[T],
	mapper function.Function[T, Stream[R]],
) Stream[R] {
	gs := stream.(*genericStream[T])
	gs.validateState()

	nextReq := make(chan struct{})
	nextData := make(chan orderedData[R])

	var rgs *genericStream[R]

	offset := uint64(0)
	lastOrder := uint64(0)

	go func() {
		for range nextReq {
			for {
				if rgs == nil {
					gs.nextReq <- struct{}{}
					od, ok := <-gs.nextData
					if !ok {
						close(nextData)
						close(gs.nextReq)
						go func() {
							for range nextReq {
							}
						}()
						return
					}

					r := mapper(od.data)
					rgs = r.(*genericStream[R])
				}

				rgs.nextReq <- struct{}{}
				r, ok := <-rgs.nextData
				if !ok {
					close(rgs.nextReq)
					rgs = nil
					offset += lastOrder + 1
				} else {
					lastOrder = r.order
					nextData <- orderedData[R]{
						order: r.order + offset,
						data:  r.data,
					}
					break
				}
			}
		}
	}()

	// Always return non-parallel stream
	return &genericStream[R]{
		parallelCount: 1,
		nextReq:       nextReq,
		nextData:      nextData,
	}
}

// Returns a sequential ordered stream whose elements are the specified
// values.
func Of[T any](data ...T) Stream[T] {
	nextReq := make(chan struct{})
	nextData := make(chan orderedData[T])
	prevDone := make(chan struct{})

	go func() {
		i := 0
		for range nextReq {
			if i == len(data) {
				close(nextData)
				close(prevDone)
				go func() {
					for range nextReq {
					}
				}()
				return
			}
			if i < len(data) {
				nextData <- orderedData[T]{
					order: uint64(i),
					data:  data[i],
				}
				i++
			}
		}
		close(nextData)
		close(prevDone)
	}()

	return &genericStream[T]{
		parallelCount: 1,
		prevDone:      prevDone,
		nextReq:       nextReq,
		nextData:      nextData,
	}
}

// Distinct returns a stream consisting of the distinct elements
// (according to ==) of this stream.
func Distinct[T comparable](stream Stream[T]) Stream[T] {
	s := stream.(*genericStream[T])
	s.validateState()

	gs := &genericStream[T]{
		parallelCount: 1,
		prevReq:       s.nextReq,
		prevData:      s.nextData,
		nextReq:       make(chan struct{}),
		nextData:      make(chan orderedData[T]),
	}

	go func() {
		seen := make(map[T]bool)

		for range gs.nextReq {
			od, ok := gs.getPrevData()
			if !ok {
				gs.close()
				return
			}

			for seen[od.data] {
				od, ok = gs.getPrevData()
				if !ok {
					gs.close()
					return
				}
			}
			gs.nextData <- od
			seen[od.data] = true
		}
		gs.close()
	}()

	return gs
}

// Sorted returns a stream consisting of the elements of stream, sorted
// according to natural order.
func Sorted[T cmp.Ordered](stream Stream[T]) Stream[T] {
	s := stream.(*genericStream[T])
	s.validateState()

	prevReq := s.nextReq
	prevData := s.nextData

	var dataSlice []T
	for {
		prevReq <- struct{}{}
		od, ok := <-prevData
		if !ok {
			break
		}
		dataSlice = append(dataSlice, od.data)
	}
	close(prevReq)

	sort.Slice(dataSlice, func(i, j int) bool {
		return dataSlice[i] < dataSlice[j]
	})

	return Of(dataSlice...)
}

// Reduce performs a reduction on the elements of stream, using the provided
// identity, accumulation and combining functions.
func Reduce[U, T any](
	stream Stream[T],
	identity U,
	accumulator function.BiFunction[U, T, U],
	combiner function.BinaryOperator[U],
) U {
	s := stream.(*genericStream[T])
	s.validateState()

	prevReq := s.nextReq
	prevData := s.nextData

	results := make(chan U)

	parallelCount := s.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {
			result := identity
			for {
				prevReq <- struct{}{}
				od, ok := <-prevData
				if !ok {
					break
				}
				result = accumulator(result, od.data)
			}
			results <- result
		}()
	}

	result := identity
	for i := 0; i < parallelCount; i++ {
		result = combiner(result, <-results)
	}

	close(prevReq)
	close(results)

	return result
}

// Collect performs mutable reduction opertion on the elements of stream. A
// mutable result is one in which reduced value is a mutable result container
// such as a slice.
func Collect[R, T any](
	stream Stream[T],
	supplier function.Supplier[R],
	accumulator function.BiConsumer[R, T],
	combiner function.BiConsumer[R, R],
) R {
	s := stream.(*genericStream[T])
	s.validateState()

	prevReq := s.nextReq
	prevData := s.nextData

	results := make(chan R)

	parallelCount := s.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {

			result := supplier()
			for {
				prevReq <- struct{}{}
				od, ok := <-prevData
				if !ok {
					break
				}
				accumulator(result, od.data)
			}
			results <- result
		}()
	}

	result := supplier()
	for i := 0; i < parallelCount; i++ {
		combiner(result, <-results)
	}

	close(prevReq)
	close(results)

	return result
}

// CollectByCollector performs mutable reduction operation on the elements of
// stream using a Collector. A Collector encapsulates the functions used as
// arguments to Collect(Supplier, BiConsumer, BiConsumer), allowing for
// resuse of collection strategies and composition of collect operations such
// as multiple-level grouping or partitioning.
func CollectByCollector[T, R, A any](
	stream Stream[T],
	collector *Collector[T, A, R],
) R {
	supplier := collector.Supplier()
	accumulator := collector.Accumulator()
	combiner := func(r, t A) {
		_ = collector.Combiner()(r, t)
	}

	a := Collect(stream, supplier, accumulator, combiner)
	return collector.Finisher()(a)
}

// Empty returns an empty Stream
func Empty[T any]() Stream[T] {
	gs := &genericStream[T]{
		parallelCount: 1,
		nextReq:       make(chan struct{}),
		nextData:      make(chan orderedData[T]),
	}

	go func() {
		for range gs.nextReq {
			// discard all requests
		}
	}()

	close(gs.nextData)
	return gs
}

// Iterate returns an infinite sequential ordered Stream produces by iterative
// appliation of a function f to an initial element seed, producing a Stream
// consisiting of seed, f(seed), f(f(seed)), etc.
func Iterate[T any](seed T, f function.UnaryOperator[T]) Stream[T] {
	gs := &genericStream[T]{
		parallelCount: 1,
		nextReq:       make(chan struct{}, goMaxProcs),
		nextData:      make(chan orderedData[T], goMaxProcs),
	}

	go func() {
		useSeed := true
		nextValue := seed

		order := uint64(0)
		for range gs.nextReq {
			if useSeed {
				gs.nextData <- orderedData[T]{
					order: order,
					data:  seed,
				}
				useSeed = false
			} else {
				nextValue = f(nextValue)
				gs.nextData <- orderedData[T]{
					order: order,
					data:  nextValue,
				}
			}
			order++
		}
		close(gs.nextData)
	}()

	return gs
}

// IterateN returns a sequential ordered Stream produced by iterative
// application of the given next function to an initial element,
// conditioned on satisfying the given code hasNext predicate.
// stream terminates as soon as the code hasNext predicate returns false.
func IterateN[T any](
	seed T,
	hasNext function.Predicate[T],
	next function.UnaryOperator[T]) Stream[T] {

	gs := &genericStream[T]{
		parallelCount: 1,
		nextReq:       make(chan struct{}),
		nextData:      make(chan orderedData[T]),
	}

	go func() {
		nextValue := seed
		applyNext := false

		order := uint64(0)
		for range gs.nextReq {
			if applyNext {
				nextValue = next(nextValue)
			}

			if !hasNext(nextValue) {
				break
			}

			gs.nextData <- orderedData[T]{
				order: order,
				data:  nextValue,
			}
			order++
			applyNext = true
		}
		close(gs.nextData)
	}()

	return gs
}

// Generate returns an infinite sequential unordered stream where each element
// is generated by the provided Supplier.  This is suitable for generating
// constant streams, streams of random elements, etc.
func Generate[T any](s function.Supplier[T]) Stream[T] {
	gs := &genericStream[T]{
		parallelCount: 1,
		nextReq:       make(chan struct{}),
		nextData:      make(chan orderedData[T]),
	}

	go func() {
		order := uint64(0)
		for range gs.nextReq {
			gs.nextData <- orderedData[T]{
				order: order,
				data:  s(),
			}
			order++
		}
		close(gs.nextData)
	}()

	return gs
}

// Concat a lazily concatenated stream whose elements are all the elements of
// the first stream followed by all the elements of the second stream.
func Concat[T any](a, b Stream[T]) Stream[T] {
	ags := a.(*genericStream[T])
	bgs := b.(*genericStream[T])
	ags.validateState()
	bgs.validateState()

	// The concatenated stream is always not parallel.
	gs := &genericStream[T]{
		parallelCount: 1,
		nextReq:       make(chan struct{}),
		nextData:      make(chan orderedData[T]),
	}

	go func() {
		gs.prevReq = ags.nextReq
		gs.prevData = ags.nextData
		switchedToB := false

		offset := uint64(0)
		lastOrder := uint64(0)

		for range gs.nextReq {
			data, ok := gs.getPrevData()
			if !ok {
				if switchedToB {
					gs.close()
					return
				}

				gs.prevReq = bgs.nextReq
				gs.prevData = bgs.nextData
				switchedToB = true
				offset = lastOrder + 1

				data, ok = gs.getPrevData()
				if !ok {
					gs.close()
					return
				}
			}
			lastOrder = data.order
			gs.nextData <- orderedData[T]{
				order: data.order + offset,
				data:  data.data,
			}
		}
		gs.close()
	}()

	return gs
}

// Returns the sum of elements in this stream.
func Sum[T Number](stream Stream[T]) T {
	gs := stream.(*genericStream[T])
	gs.validateState()

	if !gs.parallel {
		var sum T
		gs.terminalOp(func(t T) {
			sum += t
		})
		return sum
	}

	sums := make(chan T)
	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {
			var sum T
			gs.terminalOp(func(t T) {
				sum += t
			})
			sums <- sum
		}()
	}

	var sum T
	for i := 0; i < parallelCount; i++ {
		sum += <-sums
	}
	close(sums)
	return sum
}

// Range returns a sequential ordered Stream from startInclusive to
// endExclusive (exclusive) by an incremental step of 1.
func Range[T Number](
	startInclusive T,
	endExclusive T,
) Stream[T] {
	return Iterate(
		startInclusive,
		func(t T) T {
			return t + 1
		},
	).Limit(int(endExclusive - startInclusive))
}

// RangeClosed returns a sequential ordered Stream from staticInclusive to
// endInclusive (inclusive) by an incremental step of 1.
func RangeClosed[T Number](
	startInclusive T,
	endInclusive T,
) Stream[T] {
	return Iterate(
		startInclusive,
		func(t T) T {
			return t + 1
		},
	).Limit(int(endInclusive - startInclusive + 1))
}

// Max returns the maximum element of a stream.
func Max[T Number](
	stream Stream[T],
) *Optional[T] {
	return stream.Max(func(x, y T) bool {
		return x < y
	})
}

// Min returns the minimum element of a stream.
func Min[T Number](
	stream Stream[T],
) *Optional[T] {
	return stream.Min(func(x, y T) bool {
		return x < y
	})
}
