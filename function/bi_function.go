// Copyright © 2020, 2022 Yoshiki Shibata. All rights reserved.

package function

// BiFunction represents a function that accepts two arguments and produces
// a result.
type BiFunction[T, U, R any] func(t T, u U) R
