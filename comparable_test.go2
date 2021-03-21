// Copyright © 2021 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"testing"
)

func index[T Comparable[T]](s []T, e T) int {
	for i, v := range s {
		if e.CompareTo(v) == 0 {
			return i
		}
	}
	return -1
}

type equalInt int

func (a equalInt) CompareTo(b equalInt) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	case a == b:
		return 0
	}
	panic("Not Reachable")
}

func indexEqualInt(s []equalInt, e equalInt) int {
	return index(s, e)
	// return index[equalInt](s, e)
}

func TestComparable(t *testing.T) {

}
