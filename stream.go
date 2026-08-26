// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import "github.com/YoshikiShibata/gostream/function"

// streamImpl is the unexported implementation interface backing Stream[T].
// Concrete implementations (seqStream, genericStream) satisfy this interface.
// Callers use the exported Stream[T] struct instead of this interface so that
// generic methods (which cannot be declared on interfaces) become available.
type streamImpl[T any] interface {
	Close()
	Parallel() streamImpl[T]
	Sequential() streamImpl[T]
	IsParallel() bool
	Filter(predicate function.Predicate[T]) streamImpl[T]
	Sorted(cmp func(a, b T) int) streamImpl[T]
	Peek(action function.Consumer[T]) streamImpl[T]
	Limit(maxSize int) streamImpl[T]
	Skip(n int) streamImpl[T]
	TakeWhile(predicate function.Predicate[T]) streamImpl[T]
	DropWhile(predicate function.Predicate[T]) streamImpl[T]
	ForEach(action function.Consumer[T])
	ToSlice() []T
	Reduce(identity T, accumulator function.BinaryOperator[T]) T
	ReduceToOptional(accumulator function.BinaryOperator[T]) *Optional[T]
	Min(less Less[T]) *Optional[T]
	Max(less Less[T]) *Optional[T]
	Count() int
	AnyMatch(predicate function.Predicate[T]) bool
	AllMatch(predicate function.Predicate[T]) bool
	NoneMatch(predicate function.Predicate[T]) bool
	FindFirst() *Optional[T]
	FindAny() *Optional[T]
}

// closeState carries the list of registered close handlers along the
// intermediate operations of a Stream pipeline. All Stream values that
// derive from the same source share the same *closeState, so a handler
// registered on one branch is observed by Close() on any branch.
type closeState struct {
	handlers []func()
	closed   bool
}

// Stream is a sequence of elements supporting sequential and parallel
// aggregate operations, patterned after java.util.stream.Stream.
type Stream[T any] struct {
	impl  streamImpl[T]
	close *closeState
}

// with returns a Stream[T] wrapping the given impl and preserving the
// close-handler state of s.
func (s Stream[T]) with(impl streamImpl[T]) Stream[T] {
	return Stream[T]{impl: impl, close: s.close}
}

// Close runs every close handler registered via OnClose (once), and
// then closes the underlying implementation.
func (s Stream[T]) Close() {
	if s.close != nil && !s.close.closed {
		s.close.closed = true
		for _, h := range s.close.handlers {
			h()
		}
	}
	s.impl.Close()
}

// OnClose returns a stream that runs handler when Close is called on
// this stream (or on any Stream derived from it). Handlers are executed
// in registration order.
func (s Stream[T]) OnClose(handler func()) Stream[T] {
	if s.close == nil {
		return Stream[T]{
			impl:  s.impl,
			close: &closeState{handlers: []func(){handler}},
		}
	}
	s.close.handlers = append(s.close.handlers, handler)
	return s
}

// Parallel returns an equivalent stream that is parallel. May return
// itself, either because the stream was already parallel, or because
// the underlying stream state was modified to be parallel.
func (s Stream[T]) Parallel() Stream[T] {
	return s.with(s.impl.Parallel())
}

// Sequential returns an equivalent stream that is sequential. May return
// itself, either because the stream was already sequential, or because
// the underlying stream state was modified to be sequential.
func (s Stream[T]) Sequential() Stream[T] {
	return s.with(s.impl.Sequential())
}

// IsParallel reports whether this stream would execute in parallel if
// a terminal operation were executed.
func (s Stream[T]) IsParallel() bool {
	return s.impl.IsParallel()
}

// Filter returns a stream consisting of the elements of this stream
// that match given predicate.
func (s Stream[T]) Filter(predicate function.Predicate[T]) Stream[T] {
	return s.with(s.impl.Filter(predicate))
}

// Sorted returns a stream consisting of the elements of this stream,
// according to the provided cmp.
func (s Stream[T]) Sorted(cmp func(a, b T) int) Stream[T] {
	return s.with(s.impl.Sorted(cmp))
}

// Peek returns a stream consisting of the elements of this stream,
// additionally performing the provided action on each element as elements
// are consumed from the resulting stream.
func (s Stream[T]) Peek(action function.Consumer[T]) Stream[T] {
	return s.with(s.impl.Peek(action))
}

// Limit returns a stream consisting of the elements of this stream,
// truncated to be no longer than maxSize in length.
func (s Stream[T]) Limit(maxSize int) Stream[T] {
	return s.with(s.impl.Limit(maxSize))
}

// Skip returns a stream consisting of the remaining elements of this
// stream after discarding the first n elements of the stream.
func (s Stream[T]) Skip(n int) Stream[T] {
	return s.with(s.impl.Skip(n))
}

// TakeWhile returns a stream consisting of the longest prefix of this
// stream whose elements match the given predicate. Iteration of the
// upstream stops as soon as the predicate returns false.
func (s Stream[T]) TakeWhile(predicate function.Predicate[T]) Stream[T] {
	return s.with(s.impl.TakeWhile(predicate))
}

// DropWhile returns a stream consisting of the remaining elements of
// this stream after dropping the longest prefix of elements that match
// the given predicate.
func (s Stream[T]) DropWhile(predicate function.Predicate[T]) Stream[T] {
	return s.with(s.impl.DropWhile(predicate))
}

// Map returns a stream consisting of the results of applying the given
// function to the elements of this stream.
func (s Stream[T]) Map[R any](mapper function.Function[T, R]) Stream[R] {
	return Stream[R]{impl: mapImpl(s.impl, mapper), close: s.close}
}

// FlatMap returns a stream consisting of the results of replacing each
// element of this stream with the contents of a mapped stream produced by
// applying the provided mapping function to each element.
func (s Stream[T]) FlatMap[R any](mapper function.Function[T, Stream[R]]) Stream[R] {
	return Stream[R]{impl: flatMapImpl(s.impl, mapper), close: s.close}
}

// MapMulti applies the given mapper to each element of this stream and
// emits zero or more results per element via the provided emit callback.
// It is a lightweight alternative to FlatMap that does not require
// constructing an intermediate Stream[R] for each input element.
func (s Stream[T]) MapMulti[R any](mapper func(t T, emit func(R))) Stream[R] {
	return Stream[R]{impl: mapMultiImpl(s.impl, mapper), close: s.close}
}

// ReduceWith performs a reduction on the elements of this stream, using
// the provided identity value of type U, an accumulation function and a
// combining function. It is the U-typed counterpart of Reduce.
func (s Stream[T]) ReduceWith[U any](
	identity U,
	accumulator function.BiFunction[U, T, U],
	combiner function.BinaryOperator[U],
) U {
	return reduceWithImpl(s.impl, identity, accumulator, combiner)
}

// Collect performs a mutable reduction on the elements of this stream,
// returning a container of type R.
func (s Stream[T]) Collect[R any](
	supplier function.Supplier[R],
	accumulator function.BiConsumer[R, T],
	combiner function.BiConsumer[R, R],
) R {
	return collectImpl(s.impl, supplier, accumulator, combiner)
}

// ForEach performs an action for each element of this stream.
func (s Stream[T]) ForEach(action function.Consumer[T]) { s.impl.ForEach(action) }

// ToSlice returns a slice containing the elements of this stream.
func (s Stream[T]) ToSlice() []T { return s.impl.ToSlice() }

// Reduce performs a reduction on the elements of this stream, using
// the provided identity value and an accumulation function, and returns
// the reduced value.
func (s Stream[T]) Reduce(identity T, accumulator function.BinaryOperator[T]) T {
	return s.impl.Reduce(identity, accumulator)
}

// ReduceToOptional performs a reduction on the elements of this stream,
// using an associative accumulation function, and returns an Optional
// describing the reduced value, if any.
func (s Stream[T]) ReduceToOptional(accumulator function.BinaryOperator[T]) *Optional[T] {
	return s.impl.ReduceToOptional(accumulator)
}

// Min returns the minimum element of this stream according to the
// provided Less.
func (s Stream[T]) Min(less Less[T]) *Optional[T] { return s.impl.Min(less) }

// Max returns the maximum element of this stream according to the
// provided Less.
func (s Stream[T]) Max(less Less[T]) *Optional[T] { return s.impl.Max(less) }

// Count returns the count of elements in this stream.
func (s Stream[T]) Count() int { return s.impl.Count() }

// AnyMatch returns whether any elements of this stream match the provided
// predicate.
func (s Stream[T]) AnyMatch(predicate function.Predicate[T]) bool {
	return s.impl.AnyMatch(predicate)
}

// AllMatch returns whether all elements of this stream match the provided
// predicate.
func (s Stream[T]) AllMatch(predicate function.Predicate[T]) bool {
	return s.impl.AllMatch(predicate)
}

// NoneMatch returns whether no elements of this stream match the provided
// predicate.
func (s Stream[T]) NoneMatch(predicate function.Predicate[T]) bool {
	return s.impl.NoneMatch(predicate)
}

// FindFirst returns an Optional describing the first element of this
// stream or an empty Optional if the stream is empty.
func (s Stream[T]) FindFirst() *Optional[T] { return s.impl.FindFirst() }

// FindAny returns an Optional describing some element of the stream, or
// an empty Optional if the stream is empty.
func (s Stream[T]) FindAny() *Optional[T] { return s.impl.FindAny() }
