// Copyright © 2020, 2022 Yoshiki Shibata. All rights reserved.

package function

// Predicate represents a predicate (bool-valued function) of one argument.
type Predicate[T any] func(t T) bool
