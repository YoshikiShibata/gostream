// Copyright © 2020, 2022 Yoshiki Shibata. All rights reserved.

package function

// BinaryOperator represents an operation upon two operands of the same type,
// producing a result of the same type as the operands.
type BinaryOperator[T any] BiFunction[T, T, T]
