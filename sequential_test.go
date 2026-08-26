// Copyright © 2026 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"slices"
	"sort"
	"testing"
)

func TestStream_IsParallel(t *testing.T) {
	t.Run("Of is not parallel", func(t *testing.T) {
		if got := Of(1, 2, 3).IsParallel(); got {
			t.Errorf("IsParallel() = true, want false")
		}
	})

	t.Run("Parallel() makes it parallel", func(t *testing.T) {
		s := Of(1, 2, 3).Parallel()
		if !s.IsParallel() {
			t.Errorf("IsParallel() = false, want true")
		}
	})

	t.Run("Sequential() makes it sequential again", func(t *testing.T) {
		s := Of(1, 2, 3).Parallel().Sequential()
		if s.IsParallel() {
			t.Errorf("IsParallel() = true, want false")
		}
	})

	t.Run("Range is not parallel", func(t *testing.T) {
		if Range(0, 10).IsParallel() {
			t.Errorf("IsParallel() = true, want false")
		}
	})
}

func TestStream_Sequential_OnSeqStreamIsNoop(t *testing.T) {
	s := Of(1, 2, 3, 4, 5)
	// Should return the same underlying impl (identity semantics).
	got := s.Sequential().ToSlice()
	want := []int{1, 2, 3, 4, 5}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStream_Sequential_AfterParallelPreservesElements(t *testing.T) {
	data := make([]int, 100)
	for i := range data {
		data[i] = i
	}

	got := Of(data...).Parallel().Sequential().ToSlice()
	// Element set is preserved; order may differ. Sort for comparison.
	sort.Ints(got)
	if !slices.Equal(got, data) {
		t.Errorf("elements differ: got %v", got)
	}
}

func TestStream_Sequential_ChainsWithOtherOps(t *testing.T) {
	// Go parallel then back to sequential, then filter+map+reduce.
	sum := Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).
		Parallel().
		Sequential().
		Filter(func(v int) bool { return v%2 == 0 }).
		Map(func(v int) int { return v * 10 }).
		Reduce(0, func(a, b int) int { return a + b })
	if want := 300; sum != want { // (2+4+6+8+10)*10
		t.Errorf("sum = %d, want %d", sum, want)
	}
}

func TestStream_Parallel_IsIdempotent(t *testing.T) {
	// Parallel() on an already-parallel stream should be a no-op and
	// still produce correct results.
	got := Of(1, 2, 3).Parallel().Parallel().Count()
	if got != 3 {
		t.Errorf("Count = %d, want 3", got)
	}
}

func TestStream_Sequential_IsIdempotent(t *testing.T) {
	// Sequential() on an already-sequential stream should be a no-op.
	got := Of(1, 2, 3).Sequential().Sequential().Count()
	if got != 3 {
		t.Errorf("Count = %d, want 3", got)
	}
}
