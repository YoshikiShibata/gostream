// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

// Less is a comparison function.
type Less[T any] func(t1, t2 T) bool

// Identity is a function that always returns its input argument
func Identity[T any](t T) T {
	return t
}
