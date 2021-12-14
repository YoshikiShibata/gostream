// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package alg provides basic algorithms.
package alg

import "constraints"

// Max returns the maximum of two values of some ordered type.
func Max[T constraints.Ordered](a, b T) T {
	if a < b {
		return b
	}
	return a
}

// Min returns the minimum of two values of some ordered type.
func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
