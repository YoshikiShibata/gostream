// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import "github.com/YoshikiShibata/gostream/function"

// Collector is a mutable reduction operation that accumulates input elements
// into a mutable result container.
type Collector[T, A, R any] struct {
	supplier    function.Supplier[A]
	accumulator function.BiConsumer[A, T]
	combiner    function.BinaryOperator[A]
	finisher    function.Function[A, R]
}

// Supplier is a function that creates and returns a new mutable result
// container.
func (c *Collector[T, A, R]) Supplier() function.Supplier[A] {
	return c.supplier
}

// Accumulator is a function that folds a value into a mutable result
// container.
func (c *Collector[T, A, R]) Accumulator() function.BiConsumer[A, T] {
	return c.accumulator
}

// Combiner is a function that accepts two partial results and merges them.
// The combiner function may fold state from one argument into the other and
// return that, or may return a new result container.
func (c *Collector[T, A, R]) Combiner() function.BinaryOperator[A] {
	return c.combiner
}

// Finisher performs the final transformation from the intermediate
// accumulation type A to the final result Type R
func (c *Collector[T, A, R]) Finisher() function.Function[A, R] {
	return c.finisher
}
