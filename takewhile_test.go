// Copyright © 2026 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"slices"
	"testing"
)

func TestStream_TakeWhile(t *testing.T) {
	type testCase struct {
		name  string
		input []int
		pred  func(int) bool
		want  []int
	}

	cases := []testCase{
		{
			name:  "prefix matches",
			input: []int{1, 2, 3, 4, 5, 1, 2},
			pred:  func(v int) bool { return v < 4 },
			want:  []int{1, 2, 3},
		},
		{
			name:  "predicate always true",
			input: []int{1, 2, 3},
			pred:  func(int) bool { return true },
			want:  []int{1, 2, 3},
		},
		{
			name:  "predicate false at first element",
			input: []int{5, 1, 2, 3},
			pred:  func(v int) bool { return v < 4 },
			want:  nil,
		},
		{
			name:  "empty stream",
			input: nil,
			pred:  func(int) bool { return true },
			want:  nil,
		},
	}

	for _, tc := range cases {
		for _, parallel := range []bool{false, true} {
			name := tc.name
			if parallel {
				name += "/parallel"
			} else {
				name += "/serial"
			}
			t.Run(name, func(t *testing.T) {
				s := Of(tc.input...)
				if parallel {
					s = s.Parallel()
				}
				got := s.TakeWhile(tc.pred).ToSlice()
				if !slices.Equal(got, tc.want) {
					t.Errorf("got %v, want %v", got, tc.want)
				}
			})
		}
	}
}

func TestStream_DropWhile(t *testing.T) {
	type testCase struct {
		name  string
		input []int
		pred  func(int) bool
		want  []int
	}

	cases := []testCase{
		{
			name:  "drop prefix",
			input: []int{1, 2, 3, 4, 5, 1, 2},
			pred:  func(v int) bool { return v < 4 },
			want:  []int{4, 5, 1, 2},
		},
		{
			name:  "predicate always true drops all",
			input: []int{1, 2, 3},
			pred:  func(int) bool { return true },
			want:  nil,
		},
		{
			name:  "predicate false at first element",
			input: []int{5, 1, 2, 3},
			pred:  func(v int) bool { return v < 4 },
			want:  []int{5, 1, 2, 3},
		},
		{
			name:  "empty stream",
			input: nil,
			pred:  func(int) bool { return true },
			want:  nil,
		},
	}

	for _, tc := range cases {
		for _, parallel := range []bool{false, true} {
			name := tc.name
			if parallel {
				name += "/parallel"
			} else {
				name += "/serial"
			}
			t.Run(name, func(t *testing.T) {
				s := Of(tc.input...)
				if parallel {
					s = s.Parallel()
				}
				got := s.DropWhile(tc.pred).ToSlice()
				if !slices.Equal(got, tc.want) {
					t.Errorf("got %v, want %v", got, tc.want)
				}
			})
		}
	}
}

func TestStream_TakeWhile_ChainsWithOtherOps(t *testing.T) {
	// takeWhile then filter then reduce
	sum := Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).
		TakeWhile(func(v int) bool { return v < 8 }).
		Filter(func(v int) bool { return v%2 == 0 }).
		Reduce(0, func(a, b int) int { return a + b })
	if want := 12; sum != want { // 2 + 4 + 6
		t.Errorf("sum = %d, want %d", sum, want)
	}
}

func TestStream_DropWhile_ChainsWithOtherOps(t *testing.T) {
	// dropWhile then map then collect via slice
	got := Of(1, 2, 3, 4, 5, 6).
		DropWhile(func(v int) bool { return v < 4 }).
		Map(func(v int) int { return v * 10 }).
		ToSlice()
	want := []int{40, 50, 60}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
