// Copyright © 2020, 2022 Yoshiki Shibata. All rights reserved.

package function

// BiConsumer represents an operation that accepts two input arguments and
// returns no result.
type BiConsumer[T, U any] func(t T, u U)
