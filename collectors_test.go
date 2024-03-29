// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestCollectors_ToSliceCollector(t *testing.T) {
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

			result := CollectByCollector(s, ToSliceCollector[int]())
			if parallel {
				sort.Slice(result, func(i, j int) bool {
					return result[i] < result[j]
				})
			}
			if !slices.Equal(result, data) {
				t.Errorf("result is %v, want %v", result, data)
			}
		}
	}
}

func TestCollectors_ToSetCollector(t *testing.T) {
	for _, tc := range [...]struct {
		dataSize int
	}{
		{dataSize: 0},
		{dataSize: 1},
		{dataSize: 1000},
	} {

		var data []int
		var want = make(map[int]bool)
		for i := 0; i < tc.dataSize; i++ {
			data = append(data, i, i, i)
			want[i] = true
		}

		for _, parallel := range [...]bool{false, true} {
			s := Of(data...)
			if parallel {
				s = s.Parallel()
			}

			result := CollectByCollector(s, ToSetCollector[int]())
			if len(result) != tc.dataSize {
				t.Errorf("len(result) is %d, want %d", len(result), tc.dataSize)
			}
			for i := 0; i < tc.dataSize; i++ {
				if !result[i] {
					t.Errorf("result[%d] is false, want true", i)
				}
			}
		}

	}
}

func TestCollectors_JoiningCollector(t *testing.T) {
	data := []string{"hello", "world", "こんにちは", "世界"}
	result := CollectByCollector(Of(data...), JoiningCollector(" "))
	want := strings.Join(data, " ")
	if result != want {
		t.Errorf("result is %q, want %q", result, want)
	}
}

func TestCollectors_MappingCollector(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := CollectByCollector(Of(data...),
		MappingCollector(
			strconv.Itoa,
			JoiningCollector(" ")))

	want := CollectByCollector(
		Map(Of(data...),
			strconv.Itoa),
		JoiningCollector(" "))

	if result != want {
		t.Errorf("result is %q, want %q", result, want)
	}
}

func TestCollectors_FlatMappingCollector(t *testing.T) {
	data := []int{0, 10, 20}
	result := CollectByCollector(
		Of(data...),
		FlatMappingCollector(
			func(t int) Stream[string] {
				return Map(
					Iterate(t, func(v int) int { return v + 1 }).Limit(10),
					strconv.Itoa,
				)
			},
			JoiningCollector(" "),
		),
	)

	var data2 []string
	for i := 0; i < 30; i++ {
		data2 = append(data2, strconv.Itoa(i))
	}
	want := strings.Join(data2, " ")

	if result != want {
		t.Errorf("result is %q, want %q", result, want)
	}
}

func TestCollectors_FilteringCollector(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	result := CollectByCollector(
		Of(data...),
		FilteringCollector(
			func(t int) bool { return t&1 == 0 },
			ToSliceCollector[int](),
		),
	)

	want := []int{2, 4, 6, 8, 10}
	if !slices.Equal(result, want) {
		t.Errorf("result is %v, want %v", result, want)
	}
}

func TestCollectors_GroupingByToSliceCollector(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := CollectByCollector(
		Of(data...),
		GroupingByToSliceCollector(
			func(t int) string {
				if t&1 == 0 {
					return "even"
				}
				return "odd"
			},
		),
	)
	want := "map[even:[2 4 6 8 10] odd:[1 3 5 7 9]]"
	resultStr := fmt.Sprintf("%v", result)
	if resultStr != want {
		t.Errorf("resultStr is %q, but want %q", resultStr, want)
	}
}

func TestCollectors_GroupingByCollector(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	t.Run("Set", func(t *testing.T) {
		result := CollectByCollector(
			Of(data...),
			GroupingByCollector(
				func(t int) string {
					if t&1 == 0 {
						return "even"
					}
					return "odd"
				},
				ToSetCollector[int]()),
		)

		want := "map[even:map[2:true 4:true 6:true 8:true 10:true] odd:map[1:true 3:true 5:true 7:true 9:true]]"
		resultStr := fmt.Sprintf("%v", result)
		if resultStr != want {
			t.Errorf("resultStr is %q, but want %q", resultStr, want)
		}
	})

	t.Run("Slice", func(t *testing.T) {
		result := CollectByCollector(
			Of(data...),
			GroupingByCollector(
				func(t int) string {
					if t&1 == 0 {
						return "even"
					}
					return "odd"
				},
				ToSliceCollector[int]()),
		)
		want := "map[even:[2 4 6 8 10] odd:[1 3 5 7 9]]"
		resultStr := fmt.Sprintf("%v", result)
		if resultStr != want {
			t.Errorf("resultStr is %q, but want %q", resultStr, want)
		}
	})

	t.Run("Counting", func(t *testing.T) {
		result := CollectByCollector(
			Of(data...),
			GroupingByCollector(
				func(t int) string {
					if t&1 == 0 {
						return "even"
					}
					return "odd"
				},
				CountingCollector[int]()),
		)
		want := "map[even:5 odd:5]"
		resultStr := fmt.Sprintf("%v", result)
		if resultStr != want {
			t.Errorf("resultStr is %q, but want %q", resultStr, want)
		}
	})

	t.Run("MaxBy", func(t *testing.T) {
		result := CollectByCollector(
			Of(data...),
			GroupingByCollector(
				func(t int) string {
					if t&1 == 0 {
						return "even"
					}
					return "odd"
				},
				MaxByCollector(func(x, y int) bool {
					return x < y
				})),
		)
		want := "map[even:Optional[10] odd:Optional[9]]"
		resultStr := fmt.Sprintf("%v", result)
		if resultStr != want {
			t.Errorf("resultStr is %q, but want %q", resultStr, want)
		}
	})

	t.Run("MinBy", func(t *testing.T) {
		result := CollectByCollector(
			Of(data...),
			GroupingByCollector(
				func(t int) string {
					if t&1 == 0 {
						return "even"
					}
					return "odd"
				},
				MinByCollector(func(x, y int) bool {
					return x < y
				})),
		)
		want := "map[even:Optional[2] odd:Optional[1]]"
		resultStr := fmt.Sprintf("%v", result)
		if resultStr != want {
			t.Errorf("resultStr is %q, but want %q", resultStr, want)
		}
	})
}

func TestCollectors_PartitioningByToSliceCollector(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	t.Run("Slice", func(t *testing.T) {
		result := CollectByCollector(
			Of(data...),
			PartitioningByToSliceCollector(
				func(t int) bool { return t&1 == 0 },
			),
		)
		want := "map[false:[1 3 5 7 9] true:[2 4 6 8 10]]"
		resultStr := fmt.Sprintf("%v", result)
		if resultStr != want {
			t.Errorf("resultStr is %q, but want %q", resultStr, want)
		}
	})
}
func TestCollectors_PartitioningByCollector(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	t.Run("Slice", func(t *testing.T) {
		result := CollectByCollector(
			Of(data...),
			PartitioningByCollector(
				func(t int) bool { return t&1 == 0 },
				ToSliceCollector[int]()),
		)
		want := "map[false:[1 3 5 7 9] true:[2 4 6 8 10]]"
		resultStr := fmt.Sprintf("%v", result)
		if resultStr != want {
			t.Errorf("resultStr is %q, but want %q", resultStr, want)
		}
	})
}

func TestCollectors_ToMapCollector(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	result := CollectByCollector(
		Of(data...),
		ToMapCollector(
			func(t int) string {
				if t&1 == 0 {
					return "even"
				}
				return "odd"
			},
			Identity[int],
			func(v1, v2 int) int { return v1 + v2 },
		),
	)
	evenSum := 2 + 4 + 6 + 8 + 10
	oddSum := 1 + 3 + 5 + 7 + 9
	if result["even"] != evenSum {
		t.Errorf("result[even] is %d, want %d", result["even"], evenSum)
	}
	if result["odd"] != oddSum {
		t.Errorf("result[odd] is %d, want %d", result["odd"], oddSum)
	}
}

func TestCollectors_SummarizingCollector(t *testing.T) {
	count := 1000
	s := Iterate(1, func(t int) int {
		return t + 1
	}).Limit(count).Parallel()

	result := CollectByCollector(
		s,
		SummarizingCollector(func(i int) int {
			return i
		}),
	)

	if result.GetCount() != int64(count) {
		t.Errorf("result.GetCount() is %d, want %d", result.GetCount(), count)
	}
	if result.GetMin() != 1 {
		t.Errorf("result.GetMin() is %d, want 1", result.GetMin())
	}
	if result.GetMax() != int64(count) {
		t.Errorf("result.GetMax() is %d, want %d", result.GetMax(), count)
	}
	wantSum := (1 + count) * count / 2
	if result.GetSum() != int64(wantSum) {
		t.Errorf("result.GetSum() is %d, want %d", result.GetSum(), wantSum)
	}
	wantAverage := float64(wantSum) / float64(count)
	if result.GetAverage() != wantAverage {
		t.Errorf("result.GetAverage() is %e, want %e", result.GetAverage(), wantAverage)
	}
	t.Logf("result : %v\n", result)
}

func TestCollectors_SummingCollector(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := Empty[string]()
		sum := CollectByCollector(s,
			SummingCollector(func(t string) int {
				return len(t)
			}),
		)

		if sum != 0 {
			t.Errorf("sum is %d, want 0", sum)
		}
	})

	t.Run("file lines", func(t *testing.T) {
		s, err := FileLines("testdata/alice.txt")
		if err != nil {
			t.Fatalf("FileLines failed: %v", err)
		}
		count := s.Count()

		s, err = FileLines("testdata/alice.txt")
		if err != nil {
			t.Fatalf("FileLines failed: %v", err)
		}
		count2 := CollectByCollector(s,
			SummingCollector(func(t string) int {
				return 1
			}),
		)

		if count2 != count {
			t.Errorf("count2 is %d, want %d", count2, count)
		}
	})
}

func TestCollectors_AveragingInt64Collector(t *testing.T) {
	start := -777
	end := 9999
	average := CollectByCollector(
		RangeClosed(start, end).Parallel(),
		AveragingInt64Collector(func(t int) int64 {
			return int64(t)
		}),
	)

	sum := 0
	for i := start; i <= end; i++ {
		sum += i
	}
	count := end - start + 1
	wantAverage := float64(sum) / float64(count)
	if average != wantAverage {
		t.Errorf("average is %e, want %e", average, wantAverage)
	}
}

func TestCollectors_AveragingFloat64Collector(t *testing.T) {
	start := -777
	end := 9999
	average := CollectByCollector(
		RangeClosed(start, end).Parallel(),
		AveragingFloat64Collector(func(t int) float64 {
			return float64(t)
		}),
	)

	sum := 0
	for i := start; i <= end; i++ {
		sum += i
	}
	count := end - start + 1
	wantAverage := float64(sum) / float64(count)
	if average != wantAverage {
		t.Errorf("average is %e, want %e", average, wantAverage)
	}
}
