// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"fmt"
	"math"

	"github.com/YoshikiShibata/gostream/alg"
)

type SummaryStatistics[T Number] struct {
	count int64
	sum   int64
	min   int64
	max   int64
}

func NewSummaryStatistics[T Number]() *SummaryStatistics[T] {
	return &SummaryStatistics[T]{
		count: 0,
		sum:   0,
		min:   math.MaxInt64,
		max:   math.MinInt64,
	}
}

func (i *SummaryStatistics[T]) accept(value T) {
	i.count++
	i.sum += int64(value)
	i.min = alg.Min(i.min, int64(value))
	i.max = alg.Max(i.max, int64(value))
}

func (i *SummaryStatistics[T]) combine(other *SummaryStatistics[T]) {
	i.count += other.count
	i.sum += other.sum
	i.min = alg.Min(i.min, other.min)
	i.max = alg.Max(i.max, other.max)
}

func (i *SummaryStatistics[T]) GetCount() int64 {
	return i.count
}

func (i *SummaryStatistics[T]) GetSum() int64 {
	return i.sum
}

func (i *SummaryStatistics[T]) GetMin() int64 {
	return i.min
}

func (i *SummaryStatistics[T]) GetMax() int64 {
	return i.max
}

func (i *SummaryStatistics[T]) GetAverage() float64 {
	if i.count > 0 {
		return float64(i.sum) / float64(i.count)
	}
	return 0.0
}

func (i *SummaryStatistics[T]) String() string {
	return fmt.Sprintf("%#v", i)
}
