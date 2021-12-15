// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"golang.org/x/exp/slices"
)

func TestStream_ForEach(t *testing.T) {
	data := make([]int, 1000)
	for i := 0; i < len(data); i++ {
		data[i] = i
	}

	t.Run("serial", func(t *testing.T) {
		want := 0
		Of(data...).ForEach(func(v int) {
			if v != want {
				t.Fatalf("t is %d, want %d", v, want)
			}
			want++
		})
	})

	t.Run("parallel", func(t *testing.T) {
		resultChan := make(chan int, 1000)

		Of(data...).Parallel().ForEach(func(v int) {
			resultChan <- v
		})

		close(resultChan)

		var result []int
		for v := range resultChan {
			result = append(result, v)
		}

		if len(result) != len(data) {
			t.Errorf("len(result) is %d, want %d",
				len(result), len(data))
		}

		isSorted := func() bool {
			for i := 1; i < len(result); i++ {
				if result[i-1] > result[i] {
					return false
				}
			}
			return true
		}
		if goMaxProcs != 1 && isSorted() {
			t.Errorf("result(%v) is sorted\n", result)
		}
	})
}

func TestStream_Filter(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1000},
	} {
		var data []int
		var want []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
			if i&1 == 1 {
				want = append(want, i)
			}
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := s.Filter(func(t int) bool {
				return t&1 == 1
			}).ToSlice()

			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}
		}
	}
}

func TestStream_Sorted(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1000},
	} {
		var data []int
		var want []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, tc.dataSize-i-1)
			want = append(want, i)
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := s.Sorted(func(t1, t2 int) bool {
				return t1 < t2
			}).ToSlice()

			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}
		}
	}
}

func TestStream_Peek(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1000},
	} {
		var data []int
		var want []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, tc.dataSize-1-i)
			want = append(want, i)
		}

		t.Run("serial", func(t *testing.T) {
			s := Of(data...)

			peekedChan := make(chan int, tc.dataSize)
			peekFunc := func(t int) {
				peekedChan <- t
			}

			result := s.Peek(peekFunc).Sorted(func(t1, t2 int) bool {
				return t1 < t2
			}).ToSlice()
			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}

			close(peekedChan)
			var peeked []int
			for v := range peekedChan {
				peeked = append(peeked, v)
			}

			if !slices.Equal(peeked, data) {
				t.Errorf("peeked is %v, want %v", peeked, data)
			}
		})

		t.Run("parallel", func(t *testing.T) {
			s := Of(data...).Parallel()

			peekedChan := make(chan int, tc.dataSize)
			peekFunc := func(t int) {
				peekedChan <- t
			}

			result := s.Peek(peekFunc).Sorted(func(t1, t2 int) bool {
				return t1 < t2
			}).ToSlice()

			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}

			close(peekedChan)
			var peeked []int
			for v := range peekedChan {
				peeked = append(peeked, v)
			}

			if tc.dataSize != 0 && tc.dataSize != 1 && goMaxProcs != 1 {
				if slices.Equal(peeked, data) {
					t.Logf("tc.dataSize = %d", tc.dataSize)
					t.Errorf("peeked is identical to data")
				}
			}

			sort.Slice(peeked, func(i, j int) bool {
				return peeked[i] < peeked[j]
			})
			if !slices.Equal(peeked, want) {
				t.Errorf("peeked is %v, want %v", peeked, want)
			}
		})
	}
}

func TestStream_Limit(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
		limit    int
	}{
		{dataSize: 0, limit: 1},
		{dataSize: 1, limit: 1},
		{dataSize: 1000, limit: 100},
	} {
		var data []int
		var want []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
			if i < tc.limit {
				want = append(want, i)
			}
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := s.Limit(tc.limit).ToSlice()
			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}
		}
	}
}

func TestStream_Skip(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
		skip     int
	}{
		{dataSize: 0, skip: 1},
		{dataSize: 1, skip: 1},
		{dataSize: 1000, skip: 100},
	} {
		var data []int
		var want []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
			if i >= tc.skip {
				want = append(want, i)
			}
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := s.Skip(tc.skip).ToSlice()
			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}
		}
	}
}

func TestStream_ToSlice(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1000},
	} {
		var data []int
		var want []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
			want = append(want, i)
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := s.ToSlice()
			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}
		}
	}
}

func TestStream_Reduce(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1000},
	} {
		var data []int
		var want int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
			want += i
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := Of(data...).Reduce(0, sum[int])

			if result != want {
				t.Errorf("result is %d, want %d", result, want)
			}
		}
	}
}

func TestStream_ReduceToOptional(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
		present  bool
	}{
		{dataSize: 0, present: false},
		{dataSize: 1, present: true},
		{dataSize: 1000, present: true},
	} {
		var data []int
		var want int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
			want += i
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := s.ReduceToOptional(sum[int])

			if result.IsPresent() != tc.present {
				t.Errorf("result.IsPresent() is %t, want %t",
					result.IsPresent(), tc.present)
			}

			if tc.present && result.Get() != want {
				t.Errorf("result.Get() is %d, want %d", result.Get(), want)
			}
		}

	}
}

func TestStream_Min(t *testing.T) {
	rand.Seed(time.Now().Unix())

	for _, tc := range [...]struct {
		dataSize int
		present  bool
	}{
		{dataSize: 0, present: false},
		{dataSize: 1, present: true},
		{dataSize: 1000, present: true},
	} {
		var data []int
		var want int
		wantSet := false

		for i := 0; i < tc.dataSize; i++ {
			r := rand.Int()
			data = append(data, r)
			if !wantSet {
				want = r
				wantSet = true
				continue
			}
			if r < want {
				want = r
			}
		}

		less := func(t1, t2 int) bool { return t1 < t2 }

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			min := s.Min(less)
			if min.IsPresent() != tc.present {
				t.Errorf("min.IsPresent() is %t, want %t",
					min.IsPresent(), tc.present)
			}

			if tc.present && min.Get() != want {
				t.Errorf("min.Get() is %d, want %d", min.Get(), want)
			}
		}
	}
}

func TestStream_Max(t *testing.T) {
	rand.Seed(time.Now().Unix())

	for _, tc := range [...]struct {
		dataSize int
		present  bool
	}{
		{dataSize: 0, present: false},
		{dataSize: 1, present: true},
		{dataSize: 1000, present: true},
	} {
		var data []int
		var want int
		wantSet := false

		for i := 0; i < tc.dataSize; i++ {
			r := rand.Int()
			data = append(data, r)
			if !wantSet {
				want = r
				wantSet = true
				continue
			}
			if r > want {
				want = r
			}
		}

		less := func(t1, t2 int) bool { return t1 < t2 }

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			max := s.Max(less)
			if max.IsPresent() != tc.present {
				t.Errorf("max.IsPresent() is %t, want %t",
					max.IsPresent(), tc.present)
			}

			if tc.present && max.Get() != want {
				t.Errorf("max.Get() is %d, want %d", max.Get(), want)
			}
		}
	}
}

func TestStream_Count(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1000},
	} {
		var data []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			count := s.Count()
			if count != tc.dataSize {
				t.Errorf("count is %d, want %d", count, tc.dataSize)
			}
		}
	}
}

func TestStream_AnyMatch(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize  int
		matchData int
		match     bool
	}{
		{dataSize: 0, matchData: 0, match: false},
		{dataSize: 1, matchData: 0, match: true},
		{dataSize: 1, matchData: 1, match: false},
		{dataSize: 1000, matchData: 100, match: true},
		{dataSize: 1000, matchData: 1000, match: false},
	} {
		var data []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			match := s.AnyMatch(func(t int) bool {
				return t == tc.matchData
			})

			if match != tc.match {
				t.Errorf("match is %t, want %t", match, tc.match)
				t.Errorf("tc: %v", tc)
			}
		}
	}
}

func TestStream_AllMatch(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
		match    bool
	}{
		{dataSize: 0, match: true},
		{dataSize: 1, match: true},
		{dataSize: 1, match: false},
		{dataSize: 1000, match: true},
		{dataSize: 1000, match: false},
	} {
		var data []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			match := s.AllMatch(func(t int) bool {
				return tc.match
			})

			if match != tc.match {
				t.Errorf("match is %t, want %t", match, tc.match)
			}
		}
	}
}

func TestStream_NoneMatch(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
		match    bool
	}{
		{dataSize: 0, match: true},
		{dataSize: 1, match: true},
		{dataSize: 1, match: false},
		{dataSize: 1000, match: true},
		{dataSize: 1000, match: false},
	} {
		var data []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			noneMatch := s.NoneMatch(func(t int) bool {
				return tc.match
			})

			if tc.dataSize == 0 {
				if !noneMatch {
					t.Errorf("noneMatch is false, want true")
				}
				continue
			}

			if noneMatch == tc.match {
				t.Errorf("noneMatch is %t, want %t", noneMatch, !tc.match)
			}
		}
	}
}

func TestStream_FindFirst(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1000},
	} {
		var data []int

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, 777-i)
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			min := s.FindFirst()
			if tc.dataSize == 0 {
				if min.IsPresent() {
					t.Errorf("min.IsPresent() is true, want false")
				}
				continue
			}

			if min.Get() != 777 {
				t.Errorf("min.Get() is %d, want 777", min.Get())
			}
		}
	}
}
