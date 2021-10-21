// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"constraints"
)

// sum returns the sum of two values of some integer type.
func sum[T constraints.Integer](x, y T) T {
	return x + y
}
