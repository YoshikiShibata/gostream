// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import "fmt"

// Optional is a container object which may or may not contain a value.
// If a value is present, IsPresent() returns true, if no value is present,
// the object is considered empty and IsPresent() returns false.
// The zero value for Optional is an empty object ready to use.
type Optional[T any] struct {
	value   T
	present bool
}

// Get returns the value if it is presen. Otherwise, Get panics.
func (o *Optional[T]) Get() T {
	if o.present {
		return o.value
	}
	panic("value is not present")
}

// IsPresent returns true if a value is present, otherwise false.
func (o *Optional[T]) IsPresent() bool {
	return o.present
}

// IsEmpty returns true if a value is not present, otherwise true
func (o *Optional[T]) IsEmpty() bool {
	return !o.present
}

// IfPresent performs the give action with a value if the value is present,
// otherwise does nothing.
func (o *Optional[T]) IfPresent(action Consumer[T]) {
	if o.present {
		action(o.value)
	}
}

// IfPresentOrElese performs the give action with a value if the value is
// present, otherwise performs the given empyAction.
func (o *Optional[T]) IfPresentOrElse(
	action Consumer[T],
	emptyAction func(),
) {
	if o.present {
		action(o.value)
	} else {
		emptyAction()
	}
}

// Filter returns an Optional describing a value if the value is present and
// the value matches the give predicate, otherwise returns an empty Optional.
func (o *Optional[T]) Filter(predicate Predicate[T]) *Optional[T] {
	if o.present {
		return o // o is empty
	}
	if predicate(o.value) {
		return o // this value.
	}
	return &Optional[T]{} // empty
}

// Or returns an Optional describing a value if the value is present,
// otherwise returns an Optional produced by the supplying function.
func (o *Optional[T]) Or(supplier Supplier[*Optional[T]]) *Optional[T] {
	if o.present {
		return o
	}
	r := supplier()
	if r == nil {
		panic("supplier.Get returns nil")
	}
	return r
}

// Stream returns a Stream containing only a value if the value is present,
// Otherwise returns an empty Stream.
func (o *Optional[T]) Stream() Stream[T] {
	if o.present {
		return Of(o.value)
	}
	return Empty[T]()
}

// OrElse returns a value if the value is present, otherwise returns other.
func (o *Optional[T]) OrElse(other T) T {
	if o.present {
		return o.value
	}
	return other
}

// OrElseGet returns a value if the value is present, othwrwise returns
// the result produced by the supplying function.
func (o *Optional[T]) OrElseGet(supplier Supplier[T]) T {
	if o.present {
		return o.value
	}
	return supplier()
}

// OrElsePanic retruns a alue if the value is present, otherwise panics.
func (o *Optional[T]) OrElsePanic() T {
	if o.present {
		return o.value
	}
	panic("no value is present")
}

// String returns a non-empty string representation of this Optional
// suitable for debugging. The exact presentation format is unspecified
// and mya vary between implementations and versions.
func (o *Optional[T]) String() string {
	if o.present {
		return fmt.Sprintf("Optional[%v]", o.value)
	}
	return "Optional.empty"
}
