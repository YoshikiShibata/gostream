// Copyright © 2026 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"slices"
	"sort"
	"testing"
)

func TestStream_MapMulti_ExpandsElements(t *testing.T) {
	// Emit v and v*10 for each element (2-to-many).
	got := Of(1, 2, 3).
		MapMulti(func(v int, emit func(int)) {
			emit(v)
			emit(v * 10)
		}).
		ToSlice()
	want := []int{1, 10, 2, 20, 3, 30}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStream_MapMulti_FiltersElements(t *testing.T) {
	// Emit only when v > 2 (many-to-0-or-1).
	got := Of(1, 2, 3, 4, 5).
		MapMulti(func(v int, emit func(int)) {
			if v > 2 {
				emit(v)
			}
		}).
		ToSlice()
	want := []int{3, 4, 5}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStream_MapMulti_TypeChanging(t *testing.T) {
	// Stream[string] -> Stream[rune] (like a lightweight flat-map).
	got := Of("ab", "cd").
		MapMulti(func(s string, emit func(rune)) {
			for _, r := range s {
				emit(r)
			}
		}).
		ToSlice()
	want := []rune{'a', 'b', 'c', 'd'}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStream_MapMulti_EmptyInput(t *testing.T) {
	got := Of[int]().
		MapMulti(func(v int, emit func(int)) { emit(v) }).
		ToSlice()
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestStream_MapMulti_Parallel(t *testing.T) {
	// Element set is preserved (order may differ under parallel).
	data := []int{1, 2, 3, 4, 5}
	got := Of(data...).Parallel().
		MapMulti(func(v int, emit func(int)) {
			emit(v)
			emit(v * 10)
		}).
		ToSlice()

	sort.Ints(got)
	want := []int{1, 2, 3, 4, 5, 10, 20, 30, 40, 50}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStream_MapMulti_LimitStopsEarly(t *testing.T) {
	// Ensure the seqStream fast-path honours downstream termination.
	got := Iterate(1, func(v int) int { return v + 1 }).
		MapMulti(func(v int, emit func(int)) {
			emit(v)
			emit(-v)
		}).
		Limit(5).
		ToSlice()
	want := []int{1, -1, 2, -2, 3}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
