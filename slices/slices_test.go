// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slices

import (
	"math"
	"strings"
	"testing"

	"constraints"
)

func TestEqual(t *testing.T) {
	s1 := []int{1, 2, 3}
	if !Equal(s1, s1) {
		t.Errorf("Equal(%v, %v) = false, want true", s1, s1)
	}
	s2 := []int{1, 2, 3}
	if !Equal(s1, s2) {
		t.Errorf("Equal(%v, %v) = false, want true", s1, s2)
	}
	s2 = append(s2, 4)
	if Equal(s1, s2) {
		t.Errorf("Equal(%v, %v) = true, want false", s1, s2)
	}

	s3 := []float64{1, 2, math.NaN()}
	if !Equal(s3, s3) {
		t.Errorf("Equal(%v, %v) = false, want true", s3, s3)
	}

	if Equal(s1, nil) {
		t.Errorf("Equal(%v, nil) = true, want false", s1)
	}
	if Equal(nil, s1) {
		t.Errorf("Equal(nil, %v) = true, want false", s1)
	}
	if !Equal(s1[:0], nil) {
		t.Errorf("Equal(%v, nil = false, want true", s1[:0])
	}
}

func offByOne[Elem constraints.Integer](a, b Elem) bool {
	return a == b + 1 || a == b - 1
}

func TestEqualFn(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := []int{2, 3, 4}
	if EqualFn(s1, s1, offByOne[int]) {
		t.Errorf("EqualFn(%v, %v, offByOne) = true, want false", s1, s1)
	}
	if !EqualFn(s1, s2, offByOne[int]) {
		t.Errorf("EqualFn(%v, %v, offByOne) = false, want true", s1, s2)
	}

	s3 := []string{"a", "b", "c"}
	s4 := []string{"A", "B", "C"}
	if !EqualFn(s3, s4, strings.EqualFold) {
		t.Errorf("EqualFn(%v, %v, strings.EqualFold) = false, want true", s3, s4)
	}

	if !EqualFn(s1[:0], nil, offByOne[int]) {
		t.Errorf("EqualFn(%v, nil, offByOne) = false, want true", s1[:0])
	}
}

func TestMap(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := Map(s1, func(i int) float64 { return float64(i) * 2.5 })
	if want := []float64{2.5, 5, 7.5}; !Equal(s2, want) {
		t.Errorf("Map(%v, ...) = %v, want %v", s1, s2, want)
	}

	s3 := []string{"Hello", "World"}
	s4 := Map(s3, strings.ToLower)
	if want := []string{"hello", "world"}; !Equal(s4, want) {
		t.Errorf("Map(%v, strings.ToLower) = %v, want %v", s3, s4, want)
	}

	s5 := Map(nil, func(i int) int { return i })
	if len(s5) != 0 {
		t.Errorf("Map(nil, identity) = %v, want empty slice", s5)
	}
}

func TestReduce(t *testing.T) {
	s1 := []int{1, 2, 3}
	r := Reduce(s1, 0, func(f float64, i int) float64 { return float64(i) * 2.5 + f })
	if want := 15.0; r != want {
		t.Errorf("Reduce(%v, 0, ...) = %v, want %v", s1, r, want)
	}

	if got := Reduce(nil, 0, func(i, j int) int { return i + j}); got != 0 {
		t.Errorf("Reduce(nil, 0, add) = %v, want 0", got)
	}
}

func TestFilter(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := Filter(s1, func(i int) bool { return i%2 == 0 })
	if want := []int{2}; !Equal(s2, want) {
		t.Errorf("Filter(%v, even) = %v, want %v", s1, s2, want)
	}

	if s3 := Filter(s1[:0], func(i int) bool { return true }); len(s3) > 0 {
		t.Errorf("Filter(%v, identity) = %v, want empty slice", s1[:0], s3)
	}
}

func TestMax(t *testing.T) {
	s1 := []int{1, 2, 3, -5}
	if got, want := Max(s1), 3; got != want {
		t.Errorf("Max(%v) = %d, want %d", s1, got, want)
	}

	s2 := []string{"aaa", "a", "aa", "aaaa"}
	if got, want := Max(s2), "aaaa"; got != want {
		t.Errorf("Max(%v) = %q, want %q", s2, got, want)
	}

	if got, want := Max(s2[:0]), ""; got != want {
		t.Errorf("Max(%v) = %q, want %q", s2[:0], got, want)
	}
}

func TestMin(t *testing.T) {
	s1 := []int{1, 2, 3, -5}
	if got, want := Min(s1), -5; got != want {
		t.Errorf("Min(%v) = %d, want %d", s1, got, want)
	}

	s2 := []string{"aaa", "a", "aa", "aaaa"}
	if got, want := Min(s2), "a"; got != want {
		t.Errorf("Min(%v) = %q, want %q", s2, got, want)
	}

	if got, want := Min(s2[:0]), ""; got != want {
		t.Errorf("Min(%v) = %q, want %q", s2[:0], got, want)
	}
}

func TestAppend(t *testing.T) {
	s := []int{1, 2, 3}
	s = Append(s, 4, 5, 6)
	want := []int{1, 2, 3, 4, 5, 6}
	if !Equal(s, want) {
		t.Errorf("after Append got %v, want %v", s, want)
	}
}

func TestCopy(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := []int{4, 5}
	if got := Copy(s1, s2); got != 2 {
		t.Errorf("Copy returned %d, want 2", got)
	}
	want := []int{4, 5, 3}
	if !Equal(s1, want) {
		t.Errorf("after Copy got %v, want %v", s1, want)
	}
}
