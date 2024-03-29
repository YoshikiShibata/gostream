// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import "github.com/YoshikiShibata/gostream/function"

type BaseStream[T any] interface {
	// Close closes this stream, causing all close handlers for this
	// stream pipeline to be called.
	Close()
}

type Stream[T any] interface {
	BaseStream[T]

	// Parallel returns an equivalent stream that is parallel. May return
	// itself, either because the stream was alreay parallel, or because
	// the underlying stream state was modified to be parallel.
	Parallel() Stream[T]

	// Filter returns a stream consisting of the elements of this stream
	// that match given predicate.
	Filter(predicate function.Predicate[T]) Stream[T]

	// Sorted returns a stream consisting of the elements of this stream,
	// according to the provided Less.
	Sorted(cmp func(a, b T) int) Stream[T]

	// Peek returns a stream consisting of the elements of this stream,
	// additionally performing the provided action on each element as elements
	// are consumed from the resulting steam.
	Peek(action function.Consumer[T]) Stream[T]

	// Limit returns a stream consisting of the elements of this stream,
	// truncated to be no logner than maxSize in length.
	Limit(maxSize int) Stream[T]

	// Skip returns a stream consisting of the remaining elements of this
	// stream after discarding the first n elements of the stream.
	// If this stream contians fewer than n elements then an empty stream
	// will be returned.
	Skip(n int) Stream[T]

	// ForEach performs an action for each element of this stream.
	ForEach(action function.Consumer[T])

	// ToSlice returns a slice containing the elements of this stream.
	ToSlice() []T

	// Reduce performs a reduction on the elements of this stream, using
	// the provided identity value and an accumulation function, and returns
	// the reduced value.
	Reduce(identity T, accumulator function.BinaryOperator[T]) T

	// ReduceToOptional performs a reduction on the elements of this strem,
	// using an associative accumulation function, and returns an Optional
	// describing the reduced value, if nay.
	ReduceToOptional(accumulator function.BinaryOperator[T]) *Optional[T]

	// Min returns the minimum element of this stream according to the
	// provided Less.
	Min(Less Less[T]) *Optional[T]

	// Max returns the maximum element of this stream according to the
	// provided Less.
	Max(less Less[T]) *Optional[T]

	// Count returns the count of elements in this stream.
	Count() int

	// AnyMatch returns whether any elements of this stream match the provided
	// predicate. May not evaluate the predicate on all elements if not
	// necesary for determining the resulst. If the stream is empty then false
	// is returned and the predicate is not evaluated.
	AnyMatch(predicate function.Predicate[T]) bool

	// AllMatch returns whether all elements of this stream match the provided
	// predicated. May not evaluate the predicate on all elements if not
	// necessary. If the stream is empty then true is returned and the
	// predicate is not evaluated.
	AllMatch(predicate function.Predicate[T]) bool

	// NoneMatch returns whether no elements of this stream match the provide
	// predicate. May not evaluate the predicate on all elements if not
	// necessary for determing the result. If the stream is empty then true is
	// returned and predicate is not evaluated.
	NoneMatch(predicate function.Predicate[T]) bool

	// FindFirst returns an Optional describing the first element of this
	// stream or an empty Optional if the stream is empty.
	FindFirst() *Optional[T]

	// FindAny returns an Optional describing some element of the stream, or
	// an empty Optional if the steam is empty.
	//
	// The behavior of this operation is explicitly nondeterministic; it is
	// free to select any element in the stream. This is to allow for maximal
	// performance in parallel operations; the cost is that multiple
	// invocations on the same source many not return the same result. (If a
	// stable result is desired, use FindFirst instead.
	FindAny() *Optional[T]
}
