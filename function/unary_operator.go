// Copyright © 2020, 2022 Yoshiki Shibata. All rights reserved.

package function

// UnaryOperator represents an operation on a single operand that produces a
// result of the same type as its operand
type UnaryOperator[T any] Function[T, T]
