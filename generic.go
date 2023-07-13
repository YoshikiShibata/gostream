// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"fmt"
	"runtime"
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/YoshikiShibata/gostream/function"
)

type orderedData[T any] struct {
	order uint64
	data  T
}

type genericStream[T any] struct {
	lock   sync.Mutex
	closed bool

	parallel      bool
	parallelCount int
	ordered       bool

	terminalCloseCount int

	prevReq  chan struct{}
	prevData chan orderedData[T]
	prevDone chan struct{}

	nextReq  chan struct{}
	nextData chan orderedData[T]
}

var (
	goMaxProcs = runtime.GOMAXPROCS(-1)
)

func newGenericStream[T any](gs *genericStream[T]) *genericStream[T] {
	return &genericStream[T]{
		parallel:      gs.parallel,
		parallelCount: gs.parallelCount,

		terminalCloseCount: gs.terminalCloseCount,

		prevReq:  gs.nextReq,
		prevData: gs.nextData,
		prevDone: gs.prevDone,

		nextReq:  make(chan struct{}, gs.parallelCount),
		nextData: make(chan orderedData[T], gs.parallelCount*2),
	}
}

func (gs *genericStream[T]) validateState() {
	gs.lock.Lock()
	defer gs.lock.Unlock()

	if gs.closed {
		panic("stream has already been closed")
	}
}

func (gs *genericStream[T]) discard(c <-chan struct{}) {
	go func() {
		for range c {
		}
	}()
}

func (gs *genericStream[T]) close() {
	gs.lock.Lock()
	defer gs.lock.Unlock()

	if gs.closed {
		return
	}

	gs.parallelCount--
	if gs.parallelCount > 0 {
		return
	}

	close(gs.nextData)
	close(gs.prevReq)
	gs.discard(gs.nextReq)
	gs.closed = true
}

func (gs *genericStream[T]) terminalClose() {
	gs.lock.Lock()
	defer gs.lock.Unlock()

	if gs.terminalCloseCount == 0 {
		return
	}

	gs.terminalCloseCount--
	if gs.terminalCloseCount > 0 {
		return
	}

	close(gs.nextReq)
}

func (gs *genericStream[T]) getNextReq() bool {
	select {
	case _, ok := <-gs.nextReq:
		return ok
	case <-gs.prevDone:
		return false
	}
}

func (gs *genericStream[T]) getPrevData() (orderedData[T], bool) {
	gs.prevReq <- struct{}{}
	data, ok := <-gs.prevData
	return data, ok
}

func (gs *genericStream[T]) terminalOp(op function.Consumer[T]) {
	gs.nextReq <- struct{}{}
	for od := range gs.nextData {
		op(od.data)
		gs.nextReq <- struct{}{}
	}
	gs.terminalClose()
}

func (gs *genericStream[T]) terminalOpOrderedData(
	op function.Consumer[orderedData[T]]) {
	gs.nextReq <- struct{}{}
	for od := range gs.nextData {
		op(od)
		gs.nextReq <- struct{}{}
	}

	gs.terminalClose()
}

func (gs *genericStream[T]) terminalOpMatch(match func(t T) bool) {
	gs.nextReq <- struct{}{}
	for od := range gs.nextData {
		if !match(od.data) {
			break
		}
		gs.nextReq <- struct{}{}
	}

	gs.terminalClose()
}

func (gs *genericStream[T]) Parallel() Stream[T] {
	gs.validateState()

	if gs.parallel {
		return gs
	}

	newGS := newGenericStream(gs)
	newGS.parallel = true
	newGS.parallelCount = goMaxProcs
	newGS.terminalCloseCount = goMaxProcs
	newGS.nextReq = make(chan struct{}, gs.parallelCount)
	newGS.nextData = make(chan orderedData[T], gs.parallelCount)

	parallelCount := newGS.parallelCount
	for i := 0; i < parallelCount; i++ {
		go newGS.drain()
	}

	return newGS
}

func (gs *genericStream[T]) drain() {
	for gs.getNextReq() {
		data, ok := gs.getPrevData()
		if !ok {
			gs.close()
			return
		}
		gs.nextData <- data
	}
	gs.close()
}

func (gs *genericStream[T]) Filter(predicate function.Predicate[T]) Stream[T] {
	gs.validateState()

	newGS := newGenericStream(gs)

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go newGS.filter(predicate)
	}
	return newGS
}

func (gs *genericStream[T]) filter(predicate function.Predicate[T]) {
	for gs.getNextReq() {
		od, ok := gs.getPrevData()
		if !ok {
			gs.close()
			return
		}

		for !predicate(od.data) {
			od, ok = gs.getPrevData()
			if !ok {
				gs.close()
				return
			}
		}
		gs.nextData <- od
	}

	gs.close()
}

func (gs *genericStream[T]) Close() {
	gs.validateState()

	panic("Not Implemented Yet")
}

func (gs *genericStream[T]) ForEach(action function.Consumer[T]) {
	gs.validateState()

	if !gs.parallel {
		gs.terminalOp(action)
		return
	}

	var wg sync.WaitGroup

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		wg.Add(1)
		go func() {
			gs.terminalOp(action)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (gs *genericStream[T]) Sorted(cmp func(a, b T) int) Stream[T] {
	gs.validateState()

	var dataSlice []T

	if !gs.parallel {
		gs.terminalOp(func(t T) {
			dataSlice = append(dataSlice, t)
		})
	} else {
		slices := make(chan []T)

		parallelCount := gs.parallelCount
		for i := 0; i < parallelCount; i++ {
			go func() {
				var slice []T

				gs.terminalOp(func(t T) {
					slice = append(slice, t)
				})

				slices <- slice
			}()
		}

		for i := 0; i < parallelCount; i++ {
			slice := <-slices
			dataSlice = append(dataSlice, slice...)
		}
	}

	slices.SortFunc(dataSlice, cmp)
	/*
		sort.Slice(dataSlice, func(i, j int) bool {
			return less(dataSlice[i], dataSlice[j])
		})
	*/

	return Of(dataSlice...)
}

func (gs *genericStream[T]) Peek(action function.Consumer[T]) Stream[T] {
	gs.validateState()

	newGS := newGenericStream(gs)

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go newGS.peek(action)
	}

	return newGS
}

func (gs *genericStream[T]) peek(action function.Consumer[T]) {
	for gs.getNextReq() {
		od, ok := gs.getPrevData()
		if !ok {
			gs.close()
			return
		}
		action(od.data)
		gs.nextData <- od
	}
	gs.close()
}

func (gs *genericStream[T]) Limit(maxSize int) Stream[T] {
	gs.validateState()

	if gs.ordered && gs.parallelCount > 1 {
		panic("Limit doesn't support ordered parallel stream")
	}

	newGS := newGenericStream(gs)

	// we don't process elements in parallel to limit the
	// number of elements.
	newGS.parallelCount = 1
	go newGS.limit(maxSize)
	return newGS
}

func (gs *genericStream[T]) limit(maxSize int) {
	if maxSize < 0 {
		panic(fmt.Sprintf("maxSize must not be negative: %v", maxSize))
	}
	if maxSize == 0 {
		gs.close()
		return
	}

	count := maxSize
	for gs.getNextReq() {
		data, ok := gs.getPrevData()
		if !ok {
			gs.close()
			return
		}
		gs.nextData <- data
		count--
		if count == 0 {
			gs.close()
			return
		}
	}
	gs.close()
}

func (gs *genericStream[T]) Skip(n int) Stream[T] {
	gs.validateState()

	if gs.ordered && gs.parallelCount > 1 {
		panic("Skip doesn't support ordered parallel stream")
	}

	newGS := newGenericStream(gs)

	// we don't process elements in parallel to limit the
	// number of elements.
	newGS.parallelCount = 1

	go newGS.skip(n)
	return newGS
}

func (gs *genericStream[T]) skip(n int) {
	for gs.getNextReq() {
		// Skip n elements
		for n > 0 {
			_, ok := gs.getPrevData()
			if !ok {
				gs.close()
				return
			}
			n--
		}

		data, ok := gs.getPrevData()
		if !ok {
			gs.close()
			return
		}
		gs.nextData <- data
	}

	gs.close()
}

func (gs *genericStream[T]) ToSlice() []T {
	gs.validateState()

	results := make(chan []orderedData[T])

	// collect in parallel
	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {
			var ods []orderedData[T]

			gs.terminalOpOrderedData(func(od orderedData[T]) {
				ods = append(ods, od)
			})

			results <- ods
		}()
	}

	// combine all results
	var ods []orderedData[T]
	for i := 0; i < parallelCount; i++ {
		result := <-results
		ods = append(ods, result...)
	}
	close(results)

	// sort
	sort.Slice(ods, func(i, j int) bool {
		return ods[i].order < ods[j].order
	})

	// copy sorted result to []T
	result := make([]T, len(ods))
	for i := 0; i < len(ods); i++ {
		result[i] = ods[i].data
	}

	return result
}

func (gs *genericStream[T]) Reduce(
	identity T,
	accumulator function.BinaryOperator[T],
) T {
	gs.validateState()

	results := make(chan T)
	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {
			result := identity

			gs.terminalOp(func(t T) {
				result = accumulator(result, t)
			})
			results <- result
		}()
	}

	result := identity
	for i := 0; i < parallelCount; i++ {
		result = accumulator(result, <-results)
	}
	return result
}

func (gs *genericStream[T]) ReduceToOptional(
	accumulator function.BinaryOperator[T],
) *Optional[T] {
	gs.validateState()

	results := make(chan *Optional[T])

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {
			foundAny := false
			var result T

			gs.terminalOp(func(t T) {
				if !foundAny {
					foundAny = true
					result = t
				} else {
					result = accumulator(result, t)
				}
			})

			if foundAny {
				results <- OptionalOf(result)
			} else {
				results <- OptionalEmpty[T]()
			}
		}()
	}

	foundAny := false
	var result T
	for i := 0; i < parallelCount; i++ {
		oResult := <-results
		if !oResult.IsPresent() {
			continue
		}
		if !foundAny {
			foundAny = true
			result = oResult.Get()
		} else {
			result = accumulator(result, oResult.Get())
		}
	}

	if foundAny {
		return OptionalOf(result)
	}
	return OptionalEmpty[T]()
}

func (gs *genericStream[T]) Min(less Less[T]) *Optional[T] {
	gs.validateState()

	results := make(chan *Optional[T])

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {
			foundAny := false
			var result T

			gs.terminalOp(func(t T) {
				if !foundAny {
					foundAny = true
					result = t
				} else if less(t, result) {
					result = t
				}
			})

			if foundAny {
				results <- OptionalOf(result)
			} else {
				results <- OptionalEmpty[T]()
			}
		}()
	}

	foundAny := false
	var result T
	for i := 0; i < parallelCount; i++ {
		oResult := <-results
		if !oResult.IsPresent() {
			continue
		}

		if !foundAny {
			foundAny = true
			result = oResult.Get()
			continue
		}

		if less(oResult.Get(), result) {
			result = oResult.Get()
		}
	}

	if foundAny {
		return OptionalOf(result)
	}
	return OptionalEmpty[T]()
}

func (gs *genericStream[T]) Max(less Less[T]) *Optional[T] {
	gs.validateState()

	results := make(chan *Optional[T])

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {
			foundAny := false
			var result T

			gs.terminalOp(func(t T) {
				if !foundAny {
					foundAny = true
					result = t
				} else if less(result, t) {
					result = t
				}
			})

			if foundAny {
				results <- OptionalOf(result)
			} else {
				results <- OptionalEmpty[T]()
			}
		}()
	}

	foundAny := false
	var result T
	for i := 0; i < parallelCount; i++ {
		oResult := <-results
		if !oResult.IsPresent() {
			continue
		}

		if !foundAny {
			foundAny = true
			result = oResult.Get()
			continue
		}

		if less(result, oResult.Get()) {
			result = oResult.Get()
		}
	}

	if foundAny {
		return OptionalOf(result)
	}
	return OptionalEmpty[T]()
}

func (gs *genericStream[T]) Count() int {
	gs.validateState()

	results := make(chan int)

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		go func() {
			count := 0
			gs.terminalOp(func(t T) { count++ })
			results <- count
		}()
	}

	count := 0
	for i := 0; i < parallelCount; i++ {
		count += <-results
	}

	return count
}

func (gs *genericStream[T]) AnyMatch(predicate function.Predicate[T]) bool {
	gs.validateState()

	var matched int64
	var wg sync.WaitGroup

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			gs.terminalOpMatch(func(t T) bool {
				if atomic.LoadInt64(&matched) == 1 {
					return false
				}

				if !predicate(t) {
					return true // continue
				}

				atomic.StoreInt64(&matched, 1)
				return false
			})
		}()
	}
	wg.Wait()

	return atomic.LoadInt64(&matched) == 1
}

func (gs *genericStream[T]) AllMatch(predicate function.Predicate[T]) bool {
	gs.validateState()

	var matched int64 = 1
	var wg sync.WaitGroup

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			gs.terminalOpMatch(func(t T) bool {
				if atomic.LoadInt64(&matched) == 0 {
					return false
				}
				if predicate(t) {
					return true // contine
				}
				atomic.StoreInt64(&matched, 0)
				return false
			})

		}()
	}

	wg.Wait()

	return atomic.LoadInt64(&matched) == 1
}

func (gs *genericStream[T]) NoneMatch(predicate function.Predicate[T]) bool {
	gs.validateState()

	var matched int64
	var wg sync.WaitGroup

	parallelCount := gs.parallelCount
	for i := 0; i < parallelCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			gs.terminalOpMatch(func(t T) bool {
				if atomic.LoadInt64(&matched) == 1 {
					return false
				}

				if !predicate(t) {
					return true // continue
				}
				atomic.StoreInt64(&matched, 1)
				return false
			})
		}()
	}

	wg.Wait()

	return atomic.LoadInt64(&matched) != 1
}

func (gs *genericStream[T]) FindFirst() *Optional[T] {
	gs.validateState()

	// We don't process in parallel.
	gs.terminalCloseCount = 1

	foundAny := false
	var result T

	// TODO: find the first one in order.
	gs.terminalOpMatch(func(t T) bool {
		foundAny = true
		result = t

		return false
	})

	if foundAny {
		return OptionalOf(result)
	}
	return OptionalEmpty[T]()
}

func (gs *genericStream[T]) FindAny() *Optional[T] {
	gs.validateState()

	var found int64 = 0
	var wg sync.WaitGroup

	parallelCount := gs.parallelCount
	results := make(chan T, parallelCount)
	for i := 0; i < parallelCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			gs.terminalOpMatch(func(t T) bool {
				if atomic.LoadInt64(&found) == 1 {
					return false
				}

				results <- t
				atomic.StoreInt64(&found, 1)

				return false
			})

		}()
	}
	wg.Wait()

	close(results)
	if atomic.LoadInt64(&found) == 1 {
		return OptionalOf(<-results)
	}
	return OptionalEmpty[T]()
}
