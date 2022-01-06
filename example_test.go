// Copyright © 2020, 2021 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"bufio"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/exp/slices"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

func TestExample_00(t *testing.T) {
	f, err := os.Open("testdata/alice.txt")
	if err != nil {
		t.Fatalf("os.Open failed: %v\n", err)
	}
	defer f.Close()
	input := bufio.NewScanner(f)
	input.Split(bufio.ScanWords)

	var words []string
	for input.Scan() {
		words = append(words, input.Text())
	}

	t.Run("Filter", func(t *testing.T) {
		count := Of(words...).Parallel().Filter(func(w string) bool {
			return len(w) > 12
		}).Count()
		t.Logf("count is %d\n", count)
	})

	t.Run("Map", func(t *testing.T) {
		tt := t
		Map(Of(words...), strings.ToLower).
			Limit(10).ForEach(func(t string) {
			tt.Logf("%s ", t)
		})
	})

	t.Run("first runes", func(t *testing.T) {
		firstRunes := Map(Of(words...), func(t string) rune {
			for _, r := range t {
				return r
			}
			panic("emty string")
		}).Limit(10).ToSlice()

		t.Logf("first runes: %c\n", firstRunes)
	})

	runeStream := func(s string) Stream[rune] {
		runes := []rune(s)
		return Of(runes...)
	}

	t.Run("FlatMap", func(t *testing.T) {
		result := FlatMap(Of("your", "boat"), runeStream).ToSlice()

		resultStr := fmt.Sprintf("%c", result)
		want := "[y o u r b o a t]"

		if resultStr != want {
			t.Errorf("resultStr is %q, want %q", resultStr, want)
		}
	})

	t.Run("Skip", func(t *testing.T) {
		first100 := Of(words...).Limit(100).ToSlice()
		skip10 := Of(words...).Skip(10).Limit(90).ToSlice()

		if !slices.Equal(skip10, first100[10:]) {
			t.Errorf("skip10 is %v, \nwant %v", skip10, first100[10:])
		}
	})

	t.Run("Concat", func(t *testing.T) {
		result := Concat(runeStream("Hello"), runeStream("World")).ToSlice()

		resultStr := fmt.Sprintf("%c", result)
		want := "[H e l l o W o r l d]"

		if resultStr != want {
			t.Errorf("resultStr is %q, want %q", resultStr, want)
		}
	})

	t.Run("Sorted", func(t *testing.T) {
		tt := t
		Of(words...).Sorted(func(t1, t2 string) bool {
			// reversed
			return len(t2) < len(t1)
		}).Limit(10).ForEach(func(t string) {
			tt.Logf("%s ", t)
		})
	})

	t.Run("Max", func(t *testing.T) {
		largest := Of(words...).Max(func(t1, t2 string) bool {
			return t1 < t2
		})
		if largest.IsPresent() {
			t.Logf("largest: %s\n", largest.Get())
		}
	})

	t.Run("FindFirst", func(t *testing.T) {
		startsWithQ := Of(words...).Filter(func(t string) bool {
			return strings.HasPrefix(t, "Q")
		}).FindFirst()

		if startsWithQ.IsPresent() {
			t.Logf("startsWithQ: %s\n", startsWithQ.Get())
		}

		startsWithQ = Of(words...).Parallel().Filter(func(t string) bool {
			return strings.HasPrefix(t, "Q")
		}).FindAny()

		if startsWithQ.IsPresent() {
			t.Logf("startsWithQ: %s\n", startsWithQ.Get())
		}

		aWordStartWithQ := Of(words...).Parallel().AnyMatch(func(t string) bool {
			return strings.HasPrefix(t, "Q")
		})

		if !aWordStartWithQ {
			t.Errorf("aWordStartWithQ is false, want true")
		}

	})

	t.Run("Reduce", func(t *testing.T) {
		result1 := Sum(Map(Of(words...).Parallel(),
			func(s string) int { return len(s) },
		))

		result2 := Reduce(Of(words...).Parallel(),
			0, // identity
			func(t int, s string) int {
				return t + int(len(s))
			}, // accumulator
			func(t1, t2 int) int {
				return t1 + t2
			},
		)

		if result1 != result2 {
			t.Errorf("result1 is %d, want %d", result1, result2)
		}
	})

	t.Run("Joining", func(t *testing.T) {
		result := CollectByCollector(
			Of(words...).Limit(10),
			JoiningCollector(" "),
		)

		t.Logf("result := %s\n", result)
	})

	t.Run("ToMapCollector", func(t *testing.T) {
		result := CollectByCollector(
			Of(words...),
			ToMapCollector(
				Identity[string],
				func(t string) int { return 1 },
				func(v1, v2 int) int { return v1 + v2 },
			),
		)

		count := 0
		for k, v := range result {
			t.Logf("%q : %d\n", k, v)
			count++
			if count > 10 {
				return
			}
		}
	})

	t.Run("js8ri ch02_ex13", func(t *testing.T) {
		shortWords := Of(words...).Filter(func(t string) bool {
			return len(t) < 12
		})

		result := CollectByCollector(
			shortWords,
			GroupingByCollector(
				func(t string) int {
					return len(t)
				},
				CountingCollector[string](),
			),
		)
		t.Logf("result : %v\n", result)
	})

	t.Run("Effective Java, 3rd p.212", func(t *testing.T) {
		freq := CollectByCollector(
			Of(words...),
			GroupingByCollector(
				func(w string) string { return strings.ToLower(w) },
				CountingCollector[string](),
			))
		keySet := make([]string, 0, len(freq))
		for k, _ := range freq {
			keySet = append(keySet, k)
		}
		result := Of(keySet...).Sorted(
			func(k1, k2 string) bool {
				return freq[k1] > freq[k2]
			}).Limit(10).ToSlice()

		t.Logf("result : %v\n", result)
	})
}

func TestExample_01(t *testing.T) {
	tt := t
	// generate same value repeatedly
	Generate(func() string {
		return "Echo"
	}).Limit(10).ForEach(func(t string) {
		tt.Logf("%s ", t)
	})
}

func TestExample_02(t *testing.T) {
	tt := t
	// generate random stream
	Generate(rand.Int).Limit(10).ForEach(func(t int) {
		tt.Logf("%d ", t)
	})

	Generate(rand.Float64).Limit(10).ForEach(func(t float64) {
		tt.Logf("%e ", t)
	})
}

func TestExample_03(t *testing.T) {
	tt := t
	// generate 0, 1, 2, 3, 4, 5, ...
	Iterate(0, func(t int) int {
		return t + 1
	}).Limit(10).ForEach(func(t int) {
		tt.Logf("%d ", t)
	})
}

func TestExample_04(t *testing.T) {
	tt := t
	s, err := FileLines("testdata/alice.txt")
	if err != nil {
		tt.Fatalf("LinesOfFile failed: %v", err)
	}
	s.Limit(10).ForEach(func(t string) {
		tt.Logf("%v\n", t)
	})
}

func TestExample_05(t *testing.T) {
	tt := t
	Iterate(1.0, func(t float64) float64 {
		return t * 2
	}).Peek(func(t float64) {
		tt.Logf("Fetching %e\n", t)
	}).Limit(10).ForEach(func(t float64) {
		time.Sleep(time.Second / 2)
	})
}

func TestExample_06(t *testing.T) {
	result := Distinct(Of("merrily", "merrily", "merrily", "gently")).ToSlice()

	want := "[merrily gently]"
	resultStr := fmt.Sprintf("%s", result)
	if resultStr != want {
		t.Errorf("resultStr is %q, want %q", resultStr, want)
	}
}

func TestExample_07(t *testing.T) {
	inverse := func(x float64) *Optional[float64] {
		if x == 0.0 {
			return OptionalEmpty[float64]()
		}
		return OptionalOf(1.0 / x)
	}

	squareRoot := func(x float64) *Optional[float64] {
		if x < 0.0 {
			return OptionalEmpty[float64]()
		}
		return OptionalOf(math.Sqrt(x))
	}

	a := OptionalFlatMap(OptionalOf(4.0), inverse)
	result := OptionalFlatMap(a, squareRoot)

	t.Logf("result : %v", result)
}

func TestExample_RandomNumbers(t *testing.T) {
	random := func(a, c, m, seed int64) Stream[int64] {
		return Iterate(seed, func(x int64) int64 {
			return (a*x + c) % m
		})
	}
	t.Run("Simple Generation", func(t *testing.T) {
		const noOfRandoms = 100

		randomStream := random(25214903917, 11, 1<<48, 0)
		randoms := randomStream.Limit(noOfRandoms).ToSlice()

		if len(randoms) != noOfRandoms {
			t.Errorf("len(randoms) is %d, want %d", len(randoms), noOfRandoms)
		}
	})

	t.Run("Negative Values", func(t *testing.T) {
		const noOfRandoms = 100

		randomStream := random(25214903917, 11, 1<<48, 0)
		randoms := randomStream.Limit(noOfRandoms).ToSlice()

		for _, v := range randoms {
			if v < 0 {
				return
			}
		}
		t.Errorf("No negative value found")
	})

	t.Run("Million Values", func(t *testing.T) {
		const noOfRandoms = 1_000_000

		randomStream := random(25214903917, 11, math.MinInt64, 0).
			Limit(noOfRandoms)

		const length = 20
		interval := int64(math.MaxInt64 / (length / 2))
		classifier := func(value int64) int {
			for i := 0; i < length; i++ {
				lower := math.MinInt64 + int64(i)*interval
				upper := lower + interval

				if lower <= value && value < upper {
					return i
				}
			}
			panic("Impossible")
		}

		result := CollectByCollector(
			randomStream,
			GroupingByCollector(
				classifier,
				CountingCollector[int64](),
			),
		)

		t.Logf("%v", result)
	})
}

func TestEample_Pi(t *testing.T) {
	result := CollectByCollector(
		RangeClosed[float64](0, 10_000_000).Parallel(),
		SummingCollector(
			func(k float64) float64 {
				return 4 * math.Pow(-1, k) / (2*k + 1)
			}),
	)
	t.Logf("%v", result)

}

func TestExample_08(t *testing.T) {
	locales := display.Supported.Tags()
	en := display.English.Tags()

	result := CollectByCollector(
		Of(locales...),
		ToMapCollector(
			func(tag language.Tag) string {
				return en.Name(tag) // key: English name
			},
			func(tag language.Tag) string {
				return display.Self.Name(tag) // value: its own language
			},
			func(existingValue, newValue string) string {
				return existingValue
			},
		),
	)

	t.Logf("%v", result)
}

// π(1e6) is the number of primes less than or equal to 1e6
func TestExample_π(t *testing.T) {
	noOfPrimeNumbers := Map(RangeClosed[int64](2, 1e6).Parallel(), func(i int64) *big.Int {
		return big.NewInt(i)
	}).Filter(func(i *big.Int) bool {
		return i.ProbablyPrime(1)
	}).Count()
	t.Logf("noOfPrimeNumbers is %d\n", noOfPrimeNumbers)
}

// This code is same logic above without using Stream.
func TestExample_π_nostream(t *testing.T) {
	var (
		wg        sync.WaitGroup
		sem       = make(chan struct{}, runtime.GOMAXPROCS(-1))
		result    = make(chan bool)
		countChan = make(chan int)
	)

	go func() {
		count := 0
		for b := range result {
			if b {
				count++
			}
		}
		countChan <- count
	}()

	for i := int64(2); i <= 1e6; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int64) {
			result <- big.NewInt(i).ProbablyPrime(1)
			<-sem
			defer wg.Done()
		}(i)
	}

	wg.Wait()
	close(result)
	t.Logf("noOfPrimeNumbers is %d\n", <-countChan)
	close(countChan)
}
