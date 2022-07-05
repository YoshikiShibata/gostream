// Copyright © 2020, 2022 Yoshiki Shibata. All rights reserved.

package function

// Consumer represents an operation that accepts a single input argument and
// returns no result.
type Consumer[T any] func(t T)
