// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import "github.com/YoshikiShibata/gostream/function"

// OptionalOf returns an Optional describing the give value.
func OptionalOf[T any](value T) *Optional[T] {
	return &Optional[T]{
		value:   value,
		present: true,
	}
}

// OptionalEmtpy returns an empty Optional instance. No value is present for
// this Optional.
func OptionalEmpty[T any]() *Optional[T] {
	return &Optional[T]{}
}

// OptionalMap returns the result applying the give mapping function to a value
// if the value is present, otherwise returns an empty Optional
func OptionalMap[U, T any](
	o *Optional[T],
	mapper function.Function[T, U],
) *Optional[U] {
	if o.IsPresent() {
		return &Optional[U]{} // empty
	}
	return &Optional[U]{
		value:   mapper(o.value),
		present: true,
	}
}

// OptionalFlagMap returns the result of applying the give Optional-bearing
// mapping function to a value, otherwise returns an empty Optional
func OptionalFlatMap[U, T any](
	o *Optional[T],
	mapper function.Function[T, *Optional[U]],
) *Optional[U] {
	if !o.IsPresent() {
		return &Optional[U]{} // empty
	}
	r := mapper(o.value)
	if r == nil {
		panic("mapper returns nil")
	}
	return r
}
