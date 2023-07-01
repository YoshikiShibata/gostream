// Copyright © 2020, 2021 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"fmt"
	"math/rand"
	"slices"
	"sort"
	"strconv"
	"testing"
	"time"
)

func TestStream_MapFunc(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1000},
	} {
		var data []int
		var want []string

		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i)
			want = append(want, strconv.Itoa(i))
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := Map(s, strconv.Itoa).ToSlice()

			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}
		}
	}
}

func TestStream_DistinctFunc(t *testing.T) {
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
			data = append(data, i, i, i)
			want = append(want, i)
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := Distinct(s).ToSlice()

			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}
		}
	}
}

func TestStream_SortedFunc(t *testing.T) {
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

			result := Sorted(s).ToSlice()
			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}
		}
	}
}

func TestStream_ReduceFunc(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		for _, tc := range [...]struct {
			dataSize int
		}{
			{dataSize: 0},
			{dataSize: 1},
			{dataSize: 1000},
		} {
			var data []int
			wantSum := 0
			for i := 0; i < tc.dataSize; i++ {
				data = append(data, i)
				wantSum += i
			}

			for _, parallel := range [...]bool{false, true} {
				s := Of(data...)
				if parallel {
					s = s.Parallel()
				}

				sum := Reduce(s,
					0,                                   // identity
					func(u, t int) int { return u + t }, // accumulator
					func(u, t int) int { return u + t }, // combiner
				)

				if sum != wantSum {
					t.Errorf("sum is %d, want %d", sum, wantSum)
				}
			}
		}
	})

	t.Run("string", func(t *testing.T) {
		for _, tc := range [...]struct {
			dataSize int
		}{
			{dataSize: 0},
			{dataSize: 1},
			{dataSize: 1000},
		} {
			var data []string
			wantSum := 0
			for i := 0; i < tc.dataSize; i++ {
				data = append(data, strconv.Itoa(i))
				wantSum += i
			}

			for _, parallel := range [...]bool{false, true} {
				s := Of(data...)
				if parallel {
					s = s.Parallel()
				}

				sum := Reduce(s,
					0, // identity
					func(u int, t string) int {
						tVal, err := strconv.Atoi(t)
						if err != nil {
							panic(fmt.Sprintf("strconv.Atoi(%q) failed: %v", t, err))
						}
						return u + tVal
					}, // accumulator
					func(u, t int) int { return u + t }, // combiner
				)

				if sum != wantSum {
					t.Errorf("sum is %d, want %d", sum, wantSum)
				}
			}
		}
	})
}

func TestStream_CollectFunc(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1},
		{dataSize: 1000},
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

			// Action
			result := Collect(s,
				func() *[]int { // supplier
					return &[]int{}
				},
				func(r *[]int, t int) { // accumulator
					*r = append(*r, t)
				},
				func(r1, r2 *[]int) { // combiner
					*r1 = append(*r1, *r2...)
				},
			)

			// Check
			if len(*result) != len(data) {
				t.Fatalf("len(*result) is %d, want %d", len(*result), len(data))
			}

			sort.Slice(*result, func(i, j int) bool {
				return (*result)[i] < (*result)[j]
			})

			if !slices.Equal(*result, data) {
				t.Errorf("*result is  %v\n, want %v", *result, data)
			}
		}
	}
}

func TestStream_EmptyFunc(t *testing.T) {
	for _, parallel := range [...]bool{false, true} {
		count := 0
		s := Empty[int]()
		if parallel {
			s = s.Parallel()
		}

		s.ForEach(func(t int) {
			count++
		})

		if count > 0 {
			t.Errorf("count is %d, want 0", count)
		}
	}
}

func TestStream_IterateFunc(t *testing.T) {
	lastValue := 0

	Iterate[int](1, func(v int) int {
		return v + 1
	}).Limit(100).ForEach(func(v int) {
		lastValue = v
	})

	if lastValue != 100 {
		t.Errorf("lastValue is %d, want 100", lastValue)
	}
}

func TestStream_IterateNFunc(t *testing.T) {
	lastValue := 0

	IterateN[int](
		1, // seed
		func(v int) bool { // hasNext
			return v <= 100
		},
		func(v int) int { // next
			return v + 1
		}).ForEach(func(v int) {
		lastValue = v
	})

	if lastValue != 100 {
		t.Errorf("lastValue is %d, want 100", lastValue)
	}
}

func TestStream_GenerateFunc(t *testing.T) {
	t.Run("constant", func(t *testing.T) {
		Generate[int](func() int {
			return 777
		}).Limit(100).ForEach(func(v int) {
			if v != 777 {
				t.Errorf("v is %d, want 777", v)
			}
		})
	})

	t.Run("rand", func(t *testing.T) {
		var results []int

		Generate(rand.Int).Limit(100).ForEach(func(v int) {
			results = append(results, v)
		})

		sort.Slice(results, func(i, j int) bool {
			return results[i] < results[j]
		})

		preValue := results[0]
		for i := 1; i < len(results); i++ {
			if preValue == results[i] {
				t.Errorf("preValue and results[%d] are %d", i, preValue)
			}
		}
	})
}

func TestStream_ConcatFunc(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1},
		{dataSize: 100},
		{dataSize: 100},
	} {
		var dataA []int
		var dataB []int
		for i := 0; i < tc.dataSize; i++ {
			dataA = append(dataA, i)
			dataB = append(dataB, i+tc.dataSize)
		}
		var want []int
		want = append(want, dataA...)
		want = append(want, dataB...)

		for _, parallel := range [...]bool{false, true} {
			a := Of(dataA...)
			b := Of(dataB...)
			if parallel {
				a = a.Parallel()
				b = b.Parallel()
			}

			result := Concat(a, b).ToSlice()
			if !slices.Equal(result, want) {
				t.Errorf("result is %v, want %v", result, want)
			}
		}

	}
}

func TestStream_FlatMapFunc(t *testing.T) {
	mapToRuneStream := func(s string) Stream[rune] {
		runes := []rune(s)
		return Of(runes...)
	}

	result := FlatMap(
		Of("abc", "d", "efgh", "ijklmn"),
		mapToRuneStream).ToSlice()
	want := "abcdefghijklmn"

	if len(result) != len(want) {
		t.Errorf("string(result) is %q, want %q", string(result), want)
		t.Fatalf("len(result) is %d, want %d", len(result), len(want))
	}
	if string(result) != want {
		t.Errorf("string(result) is %q, want %q", string(result), want)
	}
}

func TestStream_RangeFunc(t *testing.T) {
	rangeValues := Range(0, 100).ToSlice()

	if len(rangeValues) != 100 {
		t.Fatalf("len(rangeValues) is %v, want 100", len(rangeValues))
	}
	for i := 0; i < 100; i++ {
		if rangeValues[i] != i {
			t.Errorf("rangeValues[%d] is %d, want %[1]d", i, rangeValues[i])
		}
	}
}

func TestStream_RangeClosedFunc(t *testing.T) {
	rangeValues := RangeClosed(0, 99).ToSlice()

	if len(rangeValues) != 100 {
		t.Fatalf("len(rangeValues) is %v, want 100", len(rangeValues))
	}
	for i := 0; i < 100; i++ {
		if rangeValues[i] != i {
			t.Errorf("rangeValues[%d] is %d, want %[1]d", i, rangeValues[i])
		}
	}
}

func TestStream_MaxFunc(t *testing.T) {
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

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			max := Max(s)
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

func TestStream_MinFunc(t *testing.T) {
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

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			min := Min(s)
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
