// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"fmt"
	"math"
	"strings"

	"github.com/YoshikiShibata/gostream/function"
)

// ToSliceCollector returns a Collector that accumulates the input elements
// into a new slice.
func ToSliceCollector[T any]() *Collector[T, *[]T, []T] {
	return &Collector[T, *[]T, []T]{
		supplier: func() *[]T {
			var t []T
			return &t
		},
		accumulator: func(t1 *[]T, t2 T) {
			*t1 = append(*t1, t2)
		},
		combiner: func(left, right *[]T) *[]T {
			*left = append(*left, *right...)
			return left
		},
		finisher: func(t *[]T) []T {
			return *t
		},
	}
}

// ToSetCollector returns a Collector that accumulates the input elements
// into a new map[T]bool.
func ToSetCollector[T comparable]() *Collector[T, map[T]bool, map[T]bool] {
	return &Collector[T, map[T]bool, map[T]bool]{
		supplier: func() map[T]bool {
			return make(map[T]bool)
		},
		accumulator: func(m map[T]bool, k T) {
			m[k] = true
		},
		combiner: func(left, right map[T]bool) map[T]bool {
			for k := range right {
				left[k] = true
			}
			return left
		},
		finisher: func(t map[T]bool) map[T]bool {
			return t
		},
	}
}

// JoiningCollector returns a Collector that concatenates the input elements
// into a string, in encounter order.
func JoiningCollector(
	sep string,
) *Collector[string, *[]string, string] {
	return &Collector[string, *[]string, string]{
		supplier: func() *[]string {
			var s []string
			return &s
		},
		accumulator: func(b *[]string, s string) {
			*b = append(*b, s)
		},
		combiner: func(left, right *[]string) *[]string {
			*left = append(*left, *right...)
			return left
		},
		finisher: func(b *[]string) string {
			return strings.Join(*b, sep)
		},
	}
}

// MappingCollector adapts a Collector accepting elements of type U
// to one accepting elements of type T by appling a mapping function
// to each input element before accumulation.
func MappingCollector[T, U, A, R any](
	mapper function.Function[T, U],
	downstream *Collector[U, A, R]) *Collector[T, A, R] {
	downstreamAccumulator := downstream.Accumulator()

	return &Collector[T, A, R]{
		supplier: downstream.Supplier(),
		accumulator: func(r A, t T) {
			downstreamAccumulator(r, mapper(t))
		},
		combiner: downstream.Combiner(),
		finisher: downstream.Finisher(),
	}
}

// FlatMappingCollector adapts Collector accepting elements of type U to one
// accepting elements of type T by applying a flat mapping function to each
// input element before accumulation. The flat mapping function maps an input
// elements to a Stream covering zero or more output elements that are then
// accumulated downstream.
func FlatMappingCollector[T, U, A, R any](
	mapper function.Function[T, Stream[U]],
	downstream *Collector[U, A, R]) *Collector[T, A, R] {
	downstreamAccumulator := downstream.Accumulator()

	return &Collector[T, A, R]{
		supplier: downstream.Supplier(),
		accumulator: func(r A, t T) {
			mapper(t).ForEach(func(u U) {
				downstreamAccumulator(r, u)
			})
		},
		combiner: downstream.Combiner(),
		finisher: downstream.Finisher(),
	}
}

// FilteringCollector adapts a Collector to one accepting elements of the same
// type T by applying the predicate to each input element and only accumulating
// if the predicate returns true
func FilteringCollector[T, A, R any](
	predicate function.Predicate[T],
	downstream *Collector[T, A, R],
) *Collector[T, A, R] {
	downstreamAccumulator := downstream.Accumulator()

	return &Collector[T, A, R]{
		supplier: downstream.Supplier(),
		accumulator: func(r A, t T) {
			if predicate(t) {
				downstreamAccumulator(r, t)
			}
		},
		combiner: downstream.Combiner(),
		finisher: downstream.Finisher(),
	}
}

// GroupingByToSliceCollector returns a Collector implementing a "group by"
// operation on input elements of type T, grouping elements according to a
// classification function, and returning the results in a map.
//
// The classification function maps elements to some key type K.
// The collector produces a map[K][]T whoses keys are value resulting from
// applying the classification function to the input elements which map to
// the associated key under the classification function.
func GroupingByToSliceCollector[T any, K comparable](
	classifer function.Function[T, K],
) *Collector[T, map[K]*[]T, map[K][]T] {
	return GroupingByCollector(classifer, ToSliceCollector[T]())
}

// GroupingByCollector returns a Collector implementing a cascaded "group by"
// opertion on input elements of type T, grouping elements according to a
// classifier function, and then performing a reduction operation on the values
// associated with a give key using the specified downstream Collector.
func GroupingByCollector[T any, K comparable, A, D any](
	classifier function.Function[T, K],
	downstream *Collector[T, A, D],
) *Collector[T, map[K]A, map[K]D] {

	downstreamSupplier := downstream.Supplier()
	downstreamAccumulator := downstream.Accumulator()

	return &Collector[T, map[K]A, map[K]D]{
		supplier: func() map[K]A {
			return make(map[K]A)
		},
		accumulator: func(m map[K]A, t T) {
			key := classifier(t)
			a, ok := m[key]
			if !ok {
				a = downstreamSupplier()
				m[key] = a
			}
			downstreamAccumulator(a, t)
		},
		combiner: func(m1, m2 map[K]A) map[K]A {
			for k, a := range m2 {
				_, ok := m1[k]
				if ok {
					a = downstream.Combiner()(m1[k], a)
				}
				m1[k] = a
			}
			return m1
		},
		finisher: func(a map[K]A) map[K]D {
			result := make(map[K]D)
			for k, v := range a {
				result[k] = downstream.Finisher()(v)
			}
			return result
		},
	}
}

// PartitioningByToSliceCollector returns a Collector which partitions the
// input elements according to a Predicated, and organizes them into
// a map[bool][]T.
func PartitioningByToSliceCollector[T any](
	predicate function.Predicate[T],
) *Collector[T, *[2]*[]T, map[bool][]T] {
	return PartitioningByCollector(predicate, ToSliceCollector[T]())
}

// PartitioningByCollector returs a Collector which partitions the input
// elements according to a Predicate, reduces the values in each partition
// according to antoher Collector, and organizes them into map[bool]D whose
// values are the result of the downstream reducation.
func PartitioningByCollector[T, D, A any](
	predicate function.Predicate[T],
	downstream *Collector[T, A, D],
) *Collector[T, *[2]A, map[bool]D] {
	downstreamAccumulator := downstream.Accumulator()

	return &Collector[T, *[2]A, map[bool]D]{
		supplier: func() *[2]A {
			var partition [2]A
			partition[0] = downstream.Supplier()()
			partition[1] = downstream.Supplier()()
			return &partition
		},
		accumulator: func(partition *[2]A, t T) {
			if predicate(t) {
				downstreamAccumulator((*partition)[1], t) // true
			} else {
				downstreamAccumulator((*partition)[0], t) // false
			}
		},
		combiner: func(p1, p2 *[2]A) *[2]A {
			var partition [2]A

			partition[0] = downstream.Combiner()((*p1)[0], (*p2)[0])
			partition[1] = downstream.Combiner()((*p1)[1], (*p2)[1])
			return &partition
		},
		finisher: func(partition *[2]A) map[bool]D {
			result := make(map[bool]D)
			result[false] = downstream.Finisher()((*partition)[0])
			result[true] = downstream.Finisher()((*partition)[1])
			return result
		},
	}
}

// ToUniqueKeysMapCollector returns a Collector that accumulate elements into
// a map[K]U whose keys and values are the result of applying the provided
// mapping functions to the input elements.
//
// If the mapped keys contains duplicates, this function panics.
func ToUniqueKeysMapCollector[T any, K comparable, U any](
	keyMapper function.Function[T, K],
	valueMapper function.Function[T, U],
) *Collector[T, map[K]U, map[K]U] {
	return &Collector[T, map[K]U, map[K]U]{
		supplier: func() map[K]U {
			return make(map[K]U)
		},
		accumulator: func(m map[K]U, t T) {
			key := keyMapper(t)
			value := valueMapper(t)

			if _, ok := m[key]; ok {
				panic(fmt.Sprintf("duplicated key: %v", key))
			}
			m[key] = value
		},
		combiner: func(m1, m2 map[K]U) map[K]U {
			for key, v2 := range m2 {
				if _, ok := m1[key]; ok {
					panic(fmt.Sprintf("duplicated key: %v", key))
				}

				m1[key] = v2
			}
			return m1
		},
		finisher: func(m map[K]U) map[K]U {
			return m
		},
	}
}

// ToMapCollector returns a Collector that accumulates elements into a map[K]U
// whose keys and values are the result of applying the provided mapping
// functions to the input elements.
//
// If mapped keys contains duplicates, the value mapping function is applied
// to each equal element, and the results are merged using the provided
// merging function.
func ToMapCollector[T any, K comparable, U any](
	keyMapper function.Function[T, K],
	valueMapper function.Function[T, U],
	mergeFunction function.BinaryOperator[U],
) *Collector[T, map[K]U, map[K]U] {
	return &Collector[T, map[K]U, map[K]U]{
		supplier: func() map[K]U {
			return make(map[K]U)
		},
		accumulator: func(m map[K]U, t T) {
			key := keyMapper(t)
			value := valueMapper(t)

			v, ok := m[key]
			if ok {
				value = mergeFunction(v, value)
			}
			m[key] = value
		},
		combiner: func(m1, m2 map[K]U) map[K]U {
			for key, v2 := range m2 {
				var value U

				v1, ok := m1[key]
				if ok {
					value = mergeFunction(v1, v2)
				} else {
					value = v2
				}
				m1[key] = value
			}
			return m1
		},
		finisher: func(m map[K]U) map[K]U {
			return m
		},
	}
}

// SummarizingCollector returns a Collector which applies an
// number-producing mapping function to each input element, and returns summary
// statistics for the resulting values.
func SummarizingCollector[T any, R Number](
	mapper function.Function[T, R],
) *Collector[T, *SummaryStatistics[R], *SummaryStatistics[R]] {
	return &Collector[T, *SummaryStatistics[R], *SummaryStatistics[R]]{
		supplier: NewSummaryStatistics[R],
		accumulator: func(i *SummaryStatistics[R], t T) {
			i.accept(mapper(t))
		},
		combiner: func(l *SummaryStatistics[R],
			r *SummaryStatistics[R],
		) *SummaryStatistics[R] {
			l.combine(r)
			return l
		},
		finisher: func(i *SummaryStatistics[R]) *SummaryStatistics[R] {
			return i
		},
	}
}

// SummingCollector returns a Collector that produces the sum of a
// number-valued function applied to the input elements. If no elements are
// present, the result is 0.
func SummingCollector[T any, R Number](
	mapper function.Function[T, R],
) *Collector[T, *R, R] {
	return &Collector[T, *R, R]{
		supplier: func() *R {
			return new(R)
		},
		accumulator: func(a *R, t T) {
			*a += mapper(t)
		},
		combiner: func(a, b *R) *R {
			*a += *b
			return a
		},
		finisher: func(a *R) R {
			return *a
		},
	}
}

// CountingCollector returns a Collector accepting elements of type T that
// counts the number of input elements. If no elements are present, the result
// is 0.
func CountingCollector[T any]() *Collector[T, *int64, int64] {
	return SummingCollector[T, int64](
		func(t T) int64 { return 1 },
	)
}

// ReducingCollector returns a Collector which performs a reduction of its
// input elements under a specified BinaryOperator using the provided
// identity.
func ReducingCollector[T any](
	identity T,
	op function.BinaryOperator[T],
) *Collector[T, *T, T] {
	return &Collector[T, *T, T]{
		supplier: func() *T {
			var t = identity
			return &t
		},
		accumulator: func(a *T, t T) {
			*a = op(*a, t)
		},
		combiner: func(a, b *T) *T {
			*a = op(*a, *b)
			return a
		},
		finisher: func(a *T) T {
			return *a
		},
	}
}

func ReducingToOptionalCollector[T any](
	op function.BinaryOperator[T],
) *Collector[T, *Optional[T], *Optional[T]] {
	accept := func(o *Optional[T], t T) {
		if o.present {
			o.value = op(o.value, t)
		} else {
			o.value = t
			o.present = true
		}
	}

	return &Collector[T, *Optional[T], *Optional[T]]{
		supplier: func() *Optional[T] {
			return OptionalEmpty[T]()
		},
		accumulator: func(a *Optional[T], t T) {
			accept(a, t)
		},
		combiner: func(a, b *Optional[T]) *Optional[T] {
			if b.present {
				accept(a, b.value)
			}
			return a
		},
		finisher: Identity[*Optional[T]],
	}
}

// MaxByCollector returns a Collector that produces the maximal element
// according to a given Less, described as an *Optional[T].
func MaxByCollector[T any](
	less Less[T],
) *Collector[T, *Optional[T], *Optional[T]] {
	return ReducingToOptionalCollector[T](
		func(a, b T) T {
			if less(a, b) {
				return b
			} else {
				return a
			}
		},
	)
}

// MinByCollector returns a Collector that produces the minimal element
// according to a given Less, described as an *Optional[T].
func MinByCollector[T any](
	less Less[T],
) *Collector[T, *Optional[T], *Optional[T]] {
	return ReducingToOptionalCollector[T](
		func(a, b T) T {
			if less(a, b) {
				return a
			} else {
				return b
			}
		},
	)
}

// AveragingInt64 returns a Collector that produces the arithmetic mean of an
// an int64-valued function applied to the input elements. If no elements are
// present, the result is 0.
func AveragingInt64Collector[T any](
	mapper function.Function[T, int64],
) *Collector[T, *[2]int64, float64] {
	return &Collector[T, *[2]int64, float64]{
		supplier: func() *[2]int64 {
			return new([2]int64)
		},
		accumulator: func(a *[2]int64, t T) {
			(*a)[0] += mapper(t)
			(*a)[1] += 1
		},
		combiner: func(a, b *[2]int64) *[2]int64 {
			(*a)[0] += (*b)[0]
			(*a)[1] += (*b)[1]
			return a
		},
		finisher: func(a *[2]int64) float64 {
			if (*a)[1] == 0 {
				return 0
			}
			return float64((*a)[0]) / float64((*a)[1])
		},
	}
}

func AveragingFloat64Collector[T any](
	mapper function.Function[T, float64],
) *Collector[T, *[4]float64, float64] {
	// sumWithCompensation incorporates a new float64 value using Kahan
	// summation/compensation summation.
	//
	// High-order bits of the sum are in (*intermediateSum)[0], low-order bits
	// of the sum are in (*intermediateSum)[1], any additional elements are
	// application-specific.
	sumWithCompensation := func(intermediateSum *[4]float64, value float64) {
		tmp := value - (*intermediateSum)[1]
		sum := (*intermediateSum)[0]
		velvel := sum + tmp
		(*intermediateSum)[1] = (velvel - sum) - tmp
		(*intermediateSum)[0] = velvel
	}

	// If the compensated sum is spuriously NaN from accumulating one or
	// more same-signed infinite values, return the correctly-signed infinity
	// stored in the simple sum.
	computeFinalSum := func(summands *[4]float64) float64 {
		tmp := (*summands)[0] + (*summands)[1]
		simpleSum := (*summands)[3]
		if math.IsNaN(tmp) &&
			(math.IsInf(simpleSum, 1) || math.IsInf(simpleSum, -1)) {
			return simpleSum
		}
		return tmp
	}

	return &Collector[T, *[4]float64, float64]{
		supplier: func() *[4]float64 {
			return new([4]float64)
		},
		accumulator: func(a *[4]float64, t T) {
			val := mapper(t)

			sumWithCompensation(a, val)
			(*a)[2]++
			(*a)[3] += val
		},
		combiner: func(a, b *[4]float64) *[4]float64 {
			sumWithCompensation(a, (*b)[0])
			sumWithCompensation(a, (*b)[1])
			(*a)[2] += (*b)[2]
			(*a)[3] += (*b)[3]
			return a
		},
		finisher: func(a *[4]float64) float64 {
			if (*a)[2] == 0 {
				return 0
			}
			return computeFinalSum(a) / (*a)[2]
		},
	}
}
