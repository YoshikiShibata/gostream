// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"cmp"
	"slices"
	"sync"

	"github.com/YoshikiShibata/gostream/function"
)

// newSeqStream constructs an internal *seqStream[T] from data.
// It is used to build the sequential implementation without wrapping
// it in the exported Stream[T] struct.
func newSeqStream[T any](data ...T) *seqStream[T] {
	return &seqStream[T]{
		seq: func(yield func(T) bool) {
			for _, v := range data {
				if !yield(v) {
					return
				}
			}
		},
	}
}

// mapImpl applies mapper to each element of impl and returns a new
// streamImpl[R]. It is used by the Stream[T].Map generic method and by
// the package-level Map function.
func mapImpl[T, R any](impl streamImpl[T], mapper function.Function[T, R]) streamImpl[R] {
	if ss, ok := impl.(*seqStream[T]); ok {
		upstream := ss.seq
		return &seqStream[R]{
			seq: func(yield func(R) bool) {
				upstream(func(v T) bool {
					return yield(mapper(v))
				})
			},
		}
	}

	gs := impl.(*genericStream[T])
	gs.validateState()

	nextReq := make(chan struct{}, gs.parallelCount)
	nextData := make(chan orderedData[R], gs.parallelCount*2)

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
	for range parallelCount {
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

// mapMultiImpl applies mapper to each element of impl. The mapper emits
// zero or more R values via the provided emit callback, all of which
// appear in the resulting streamImpl[R]. It is used by the
// Stream[T].MapMulti generic method.
func mapMultiImpl[T, R any](
	impl streamImpl[T],
	mapper func(t T, emit func(R)),
) streamImpl[R] {
	if ss, ok := impl.(*seqStream[T]); ok {
		upstream := ss.seq
		return &seqStream[R]{
			seq: func(yield func(R) bool) {
				stopped := false
				upstream(func(v T) bool {
					mapper(v, func(r R) {
						if stopped {
							return
						}
						if !yield(r) {
							stopped = true
						}
					})
					return !stopped
				})
			},
		}
	}

	gs := impl.(*genericStream[T])
	gs.validateState()

	nextReq := make(chan struct{})
	nextData := make(chan orderedData[R])

	go func() {
		var buffered []R
		bufIdx := 0
		order := uint64(0)

		for range nextReq {
			// If we still have buffered outputs from the previous input,
			// emit one and wait for the next request.
			if bufIdx < len(buffered) {
				nextData <- orderedData[R]{order: order, data: buffered[bufIdx]}
				order++
				bufIdx++
				continue
			}

			// Otherwise pull from upstream until the mapper produces at
			// least one output, or upstream is exhausted.
			for {
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
				buffered = buffered[:0]
				bufIdx = 0
				mapper(od.data, func(r R) {
					buffered = append(buffered, r)
				})
				if len(buffered) > 0 {
					nextData <- orderedData[R]{order: order, data: buffered[bufIdx]}
					order++
					bufIdx++
					break
				}
			}
		}
	}()

	return &genericStream[R]{
		parallelCount: 1,
		nextReq:       nextReq,
		nextData:      nextData,
	}
}

// flatMapImpl applies mapper to each element of impl and flattens the resulting
// streams into a single streamImpl[R]. It is used by the Stream[T].FlatMap
// generic method and by the package-level FlatMap function.
func flatMapImpl[T, R any](
	impl streamImpl[T],
	mapper function.Function[T, Stream[R]],
) streamImpl[R] {
	if ss, ok := impl.(*seqStream[T]); ok {
		upstream := ss.seq
		return &seqStream[R]{
			seq: func(yield func(R) bool) {
				upstream(func(v T) bool {
					rStream := mapper(v)
					stopped := false
					if rss, ok := rStream.impl.(*seqStream[R]); ok {
						rss.seq(func(r R) bool {
							if !yield(r) {
								stopped = true
								return false
							}
							return true
						})
					} else {
						for _, r := range rStream.ToSlice() {
							if !yield(r) {
								stopped = true
								break
							}
						}
					}
					return !stopped
				})
			},
		}
	}

	gs := impl.(*genericStream[T])
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
					rgs = asGenericStream(r.impl)
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

// Of returns a sequential ordered stream whose elements are the specified
// values.
func Of[T any](data ...T) Stream[T] {
	return Stream[T]{impl: newSeqStream(data...)}
}

// Distinct returns a stream consisting of the distinct elements
// (according to ==) of this stream.
func Distinct[T comparable](stream Stream[T]) Stream[T] {
	if ss, ok := stream.impl.(*seqStream[T]); ok {
		upstream := ss.seq
		return Stream[T]{impl: &seqStream[T]{
			seq: func(yield func(T) bool) {
				seen := make(map[T]struct{})
				upstream(func(v T) bool {
					if _, exists := seen[v]; exists {
						return true
					}
					seen[v] = struct{}{}
					return yield(v)
				})
			},
		}}
	}

	s := stream.impl.(*genericStream[T])
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

	return Stream[T]{impl: gs}
}

// Sorted returns a stream consisting of the elements of stream, sorted
// according to natural order.
func Sorted[T cmp.Ordered](stream Stream[T]) Stream[T] {
	if ss, ok := stream.impl.(*seqStream[T]); ok {
		upstream := ss.seq
		return Stream[T]{impl: &seqStream[T]{
			seq: func(yield func(T) bool) {
				var data []T
				upstream(func(v T) bool {
					data = append(data, v)
					return true
				})
				slices.Sort(data)
				for _, v := range data {
					if !yield(v) {
						return
					}
				}
			},
		}}
	}

	s := stream.impl.(*genericStream[T])
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

	slices.SortFunc(dataSlice, func(a, b T) int {
		if a == b {
			return 0
		}
		if a < b {
			return -1
		}
		return 1
	})
	return Of(dataSlice...)
}

// reduceWithImpl performs a reduction on impl into a value of type U, using
// the provided identity, accumulator and combiner. It is used by the
// Stream[T].ReduceWith generic method and by the package-level Reduce function.
func reduceWithImpl[T, U any](
	impl streamImpl[T],
	identity U,
	accumulator function.BiFunction[U, T, U],
	combiner function.BinaryOperator[U],
) U {
	if ss, ok := impl.(*seqStream[T]); ok {
		result := identity
		ss.seq(func(v T) bool {
			result = accumulator(result, v)
			return true
		})
		return result
	}

	s := impl.(*genericStream[T])
	s.validateState()

	prevReq := s.nextReq
	prevData := s.nextData

	results := make(chan U)

	parallelCount := s.parallelCount
	for range parallelCount {
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
	for range parallelCount {
		result = combiner(result, <-results)
	}

	close(prevReq)
	close(results)

	return result
}

// collectImpl performs mutable reduction on impl into a value of type R.
// It is used by the Stream[T].Collect generic method and by the
// package-level Collect function.
func collectImpl[T, R any](
	impl streamImpl[T],
	supplier function.Supplier[R],
	accumulator function.BiConsumer[R, T],
	combiner function.BiConsumer[R, R],
) R {
	if ss, ok := impl.(*seqStream[T]); ok {
		result := supplier()
		ss.seq(func(v T) bool {
			accumulator(result, v)
			return true
		})
		return result
	}

	s := impl.(*genericStream[T])
	s.validateState()

	prevReq := s.nextReq
	prevData := s.nextData

	results := make(chan R)

	parallelCount := s.parallelCount
	for range parallelCount {
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
	for range parallelCount {
		combiner(result, <-results)
	}

	close(prevReq)
	close(results)

	return result
}

// CollectByCollector performs mutable reduction operation on the elements of
// stream using a Collector. A Collector encapsulates the functions used as
// arguments to Stream[T].Collect(Supplier, BiConsumer, BiConsumer), allowing
// for reuse of collection strategies and composition of collect operations
// such as multiple-level grouping or partitioning.
func CollectByCollector[T, R, A any](
	stream Stream[T],
	collector *Collector[T, A, R],
) R {
	supplier := collector.Supplier()
	accumulator := collector.Accumulator()
	combiner := func(r, t A) {
		_ = collector.Combiner()(r, t)
	}

	a := stream.Collect(supplier, accumulator, combiner)
	return collector.Finisher()(a)
}

// Empty returns an empty Stream
func Empty[T any]() Stream[T] {
	return Stream[T]{impl: &seqStream[T]{
		seq: func(yield func(T) bool) {},
	}}
}

// Iterate returns an infinite sequential ordered Stream produces by iterative
// appliation of a function f to an initial element seed, producing a Stream
// consisiting of seed, f(seed), f(f(seed)), etc.
func Iterate[T any](seed T, f function.UnaryOperator[T]) Stream[T] {
	return Stream[T]{impl: &seqStream[T]{
		seq: func(yield func(T) bool) {
			v := seed
			if !yield(v) {
				return
			}
			for {
				v = f(v)
				if !yield(v) {
					return
				}
			}
		},
	}}
}

// IterateN returns a sequential ordered Stream produced by iterative
// application of the given next function to an initial element,
// conditioned on satisfying the given code hasNext predicate.
// stream terminates as soon as the code hasNext predicate returns false.
func IterateN[T any](
	seed T,
	hasNext function.Predicate[T],
	next function.UnaryOperator[T]) Stream[T] {

	return Stream[T]{impl: &seqStream[T]{
		seq: func(yield func(T) bool) {
			v := seed
			for hasNext(v) {
				if !yield(v) {
					return
				}
				v = next(v)
			}
		},
	}}
}

// Generate returns an infinite sequential unordered stream where each element
// is generated by the provided Supplier.  This is suitable for generating
// constant streams, streams of random elements, etc.
func Generate[T any](s function.Supplier[T]) Stream[T] {
	return Stream[T]{impl: &seqStream[T]{
		seq: func(yield func(T) bool) {
			for {
				if !yield(s()) {
					return
				}
			}
		},
	}}
}

// Concat a lazily concatenated stream whose elements are all the elements of
// the first stream followed by all the elements of the second stream.
func Concat[T any](a, b Stream[T]) Stream[T] {
	if ass, aOk := a.impl.(*seqStream[T]); aOk {
		if bss, bOk := b.impl.(*seqStream[T]); bOk {
			aSeq := ass.seq
			bSeq := bss.seq
			return Stream[T]{impl: &seqStream[T]{
				seq: func(yield func(T) bool) {
					stopped := false
					aSeq(func(v T) bool {
						if !yield(v) {
							stopped = true
							return false
						}
						return true
					})
					if !stopped {
						bSeq(yield)
					}
				},
			}}
		}
	}

	ags := asGenericStream(a.impl)
	bgs := asGenericStream(b.impl)
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

	return Stream[T]{impl: gs}
}

// Returns the sum of elements in this stream.
func Sum[T Number](stream Stream[T]) T {
	if ss, ok := stream.impl.(*seqStream[T]); ok {
		var sum T
		ss.seq(func(v T) bool {
			sum += v
			return true
		})
		return sum
	}

	gs := stream.impl.(*genericStream[T])
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
	for range parallelCount {
		go func() {
			var sum T
			gs.terminalOp(func(t T) {
				sum += t
			})
			sums <- sum
		}()
	}

	var sum T
	for range parallelCount {
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
