// Copyright © 2020, 2022 Yoshiki Shibata. All rights reserved.

package function

// Function represents a function that accepts one argument and produces a
// result
type Function[T, R any] func(t T) R
