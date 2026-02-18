// Copyright © 2026 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
)

// --- Helper functions ---

func makeIntData(size int) []int {
	data := make([]int, size)
	for i := range data {
		data[i] = i
	}
	return data
}

func makeStringData(size int) []string {
	data := make([]string, size)
	for i := range data {
		data[i] = "item-" + strconv.Itoa(i)
	}
	return data
}

func makeShuffledData(size int) []int {
	data := makeIntData(size)
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(data), func(i, j int) {
		data[i], data[j] = data[j], data[i]
	})
	return data
}

var benchSizes = []int{100, 10_000, 1_000_000}

// --- 1. Intermediate operations ---

func BenchmarkFilter(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					s.Filter(func(v int) bool {
						return v%2 == 0
					}).Count()
				}
			})
		}
	}
}

func BenchmarkMap(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					Map(s, func(v int) int {
						return v * 2
					}).Count()
				}
			})
		}
	}
}

func BenchmarkFlatMap(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					FlatMap(s, func(v int) Stream[int] {
						return Of(v, v+1, v+2)
					}).Count()
				}
			})
		}
	}
}

func BenchmarkSorted(b *testing.B) {
	for _, size := range benchSizes {
		data := makeShuffledData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					Sorted(s).ToSlice()
				}
			})
		}
	}
}

func BenchmarkDistinct(b *testing.B) {
	for _, size := range benchSizes {
		// 50% duplicate rate: values in [0, size/2)
		rng := rand.New(rand.NewSource(42))
		data := make([]int, size)
		half := size / 2
		for i := range data {
			data[i] = rng.Intn(half)
		}
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					Distinct(s).Count()
				}
			})
		}
	}
}

func BenchmarkPeek(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					count := 0
					s.Peek(func(v int) {
						count++
					}).Count()
				}
			})
		}
	}
}

func BenchmarkLimitSkip(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		skipN := size / 4
		limitN := size / 2
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					s.Skip(skipN).Limit(limitN).ToSlice()
				}
			})
		}
	}
}

// --- 2. Terminal operations ---

func BenchmarkReduce(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					s.Reduce(0, func(a, b int) int {
						return a + b
					})
				}
			})
		}
	}
}

func BenchmarkReduceToOptional(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					s.ReduceToOptional(func(a, b int) int {
						return a + b
					}).Get()
				}
			})
		}
	}
}

func BenchmarkSum(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					Sum(s)
				}
			})
		}
	}
}

func BenchmarkMinMax(b *testing.B) {
	for _, size := range benchSizes {
		data := makeShuffledData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s1 := Of(data...)
					if parallel {
						s1 = s1.Parallel()
					}
					Min(s1).Get()

					s2 := Of(data...)
					if parallel {
						s2 = s2.Parallel()
					}
					Max(s2).Get()
				}
			})
		}
	}
}

func BenchmarkCount(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					s.Count()
				}
			})
		}
	}
}

func BenchmarkAnyMatch(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		target := size - 1 // last element
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					s.AnyMatch(func(v int) bool {
						return v == target
					})
				}
			})
		}
	}
}

func BenchmarkAllMatch(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					s.AllMatch(func(v int) bool {
						return v >= 0
					})
				}
			})
		}
	}
}

func BenchmarkFindFirst(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					s.FindFirst().Get()
				}
			})
		}
	}
}

func BenchmarkToSlice(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					s.ToSlice()
				}
			})
		}
	}
}

func BenchmarkForEach(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					sum := 0
					s.ForEach(func(v int) {
						sum += v
					})
					_ = sum
				}
			})
		}
	}
}

// --- 3. Collectors ---

func BenchmarkCollectToSlice(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					CollectByCollector(s, ToSliceCollector[int]())
				}
			})
		}
	}
}

func BenchmarkCollectToSet(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					CollectByCollector(s, ToSetCollector[int]())
				}
			})
		}
	}
}

func BenchmarkCollectJoining(b *testing.B) {
	for _, size := range benchSizes {
		data := makeStringData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					CollectByCollector(s, JoiningCollector(","))
				}
			})
		}
	}
}

func BenchmarkCollectGroupingBy(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					CollectByCollector(s,
						GroupingByToSliceCollector[int, int](func(v int) int {
							return v % 10
						}))
				}
			})
		}
	}
}

func BenchmarkCollectPartitioning(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					CollectByCollector(s,
						PartitioningByToSliceCollector(func(v int) bool {
							return v%2 == 0
						}))
				}
			})
		}
	}
}

func BenchmarkCollectCounting(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					CollectByCollector(s, CountingCollector[int]())
				}
			})
		}
	}
}

// --- 4. Composite pipelines ---

func BenchmarkPipelineFilterMapReduce(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					Map(s.Filter(func(v int) bool {
						return v%2 == 0
					}), func(v int) int {
						return v * 3
					}).Reduce(0, func(a, b int) int {
						return a + b
					})
				}
			})
		}
	}
}

func BenchmarkPipelineFilterSortCollect(b *testing.B) {
	for _, size := range benchSizes {
		data := makeShuffledData(size)
		for _, parallel := range []bool{false, true} {
			name := fmt.Sprintf("size=%d/parallel=%v", size, parallel)
			b.Run(name, func(b *testing.B) {
				for b.Loop() {
					s := Of(data...)
					if parallel {
						s = s.Parallel()
					}
					Sorted(s.Filter(func(v int) bool {
						return v%2 == 0
					})).ToSlice()
				}
			})
		}
	}
}

// --- 5. Stream creation ---

func BenchmarkOf(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			for b.Loop() {
				Of(data...).Count()
			}
		})
	}
}

func BenchmarkRange(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			for b.Loop() {
				Range(0, size).Count()
			}
		})
	}
}

func BenchmarkIterate(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			for b.Loop() {
				Iterate(0, func(v int) int {
					return v + 1
				}).Limit(size).Count()
			}
		})
	}
}

func BenchmarkGenerate(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			for b.Loop() {
				n := 0
				Generate(func() int {
					n++
					return n
				}).Limit(size).Count()
			}
		})
	}
}

func BenchmarkConcat(b *testing.B) {
	for _, size := range benchSizes {
		data := makeIntData(size)
		half := size / 2
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			for b.Loop() {
				Concat(Of(data[:half]...), Of(data[half:]...)).Count()
			}
		})
	}
}
