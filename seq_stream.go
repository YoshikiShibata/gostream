// Copyright © 2026 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"fmt"
	"iter"
	"slices"

	"github.com/YoshikiShibata/gostream/function"
)

// seqStream is a sequential stream implementation backed by iter.Seq[T].
// It avoids channel and goroutine overhead for sequential operations.
type seqStream[T any] struct {
	seq iter.Seq[T]
}

func (s *seqStream[T]) Close() {}

func (s *seqStream[T]) Sequential() streamImpl[T] { return s }

func (s *seqStream[T]) IsParallel() bool { return false }

func (s *seqStream[T]) Parallel() streamImpl[T] {
	// Directly create a parallel genericStream from the iter.Seq source.
	// This eliminates the intermediate drain goroutines that
	// genericStream.Parallel() would otherwise create.
	nextReq := make(chan struct{}, goMaxProcs)
	nextData := make(chan orderedData[T], goMaxProcs*2)
	prevDone := make(chan struct{})

	upstream := s.seq
	go func() {
		order := uint64(0)
		upstream(func(v T) bool {
			_, ok := <-nextReq
			if !ok {
				return false
			}
			nextData <- orderedData[T]{order: order, data: v}
			order++
			return true
		})
		close(nextData)
		close(prevDone)
		go func() {
			for range nextReq {
			}
		}()
	}()

	return &genericStream[T]{
		parallel:           true,
		parallelCount:      goMaxProcs,
		terminalCloseCount: goMaxProcs,
		prevDone:           prevDone,
		nextReq:            nextReq,
		nextData:           nextData,
	}
}

// toGenericStream converts a seqStream to a channel-based genericStream.
func (s *seqStream[T]) toGenericStream() *genericStream[T] {
	nextReq := make(chan struct{}, goMaxProcs)
	nextData := make(chan orderedData[T], goMaxProcs*2)
	prevDone := make(chan struct{})

	upstream := s.seq
	go func() {
		order := uint64(0)
		upstream(func(v T) bool {
			_, ok := <-nextReq
			if !ok {
				return false
			}
			nextData <- orderedData[T]{order: order, data: v}
			order++
			return true
		})
		close(nextData)
		close(prevDone)
		go func() {
			for range nextReq {
			}
		}()
	}()

	return &genericStream[T]{
		parallelCount: 1,
		prevDone:      prevDone,
		nextReq:       nextReq,
		nextData:      nextData,
	}
}

// asGenericStream converts a streamImpl to a *genericStream, converting
// seqStream to genericStream if necessary.
func asGenericStream[T any](s streamImpl[T]) *genericStream[T] {
	if ss, ok := s.(*seqStream[T]); ok {
		return ss.toGenericStream()
	}
	return s.(*genericStream[T])
}

// --- Intermediate operations ---

func (s *seqStream[T]) Filter(predicate function.Predicate[T]) streamImpl[T] {
	upstream := s.seq
	return &seqStream[T]{
		seq: func(yield func(T) bool) {
			upstream(func(v T) bool {
				if predicate(v) {
					return yield(v)
				}
				return true
			})
		},
	}
}

func (s *seqStream[T]) Sorted(cmp func(a, b T) int) streamImpl[T] {
	upstream := s.seq
	return &seqStream[T]{
		seq: func(yield func(T) bool) {
			var data []T
			upstream(func(v T) bool {
				data = append(data, v)
				return true
			})
			slices.SortFunc(data, cmp)
			for _, v := range data {
				if !yield(v) {
					return
				}
			}
		},
	}
}

func (s *seqStream[T]) Peek(action function.Consumer[T]) streamImpl[T] {
	upstream := s.seq
	return &seqStream[T]{
		seq: func(yield func(T) bool) {
			upstream(func(v T) bool {
				action(v)
				return yield(v)
			})
		},
	}
}

func (s *seqStream[T]) Limit(maxSize int) streamImpl[T] {
	if maxSize < 0 {
		panic(fmt.Sprintf("maxSize must not be negative: %v", maxSize))
	}
	if maxSize == 0 {
		return &seqStream[T]{
			seq: func(yield func(T) bool) {},
		}
	}
	upstream := s.seq
	return &seqStream[T]{
		seq: func(yield func(T) bool) {
			count := 0
			upstream(func(v T) bool {
				count++
				if !yield(v) {
					return false
				}
				return count < maxSize
			})
		},
	}
}

func (s *seqStream[T]) Skip(n int) streamImpl[T] {
	upstream := s.seq
	return &seqStream[T]{
		seq: func(yield func(T) bool) {
			skipped := 0
			upstream(func(v T) bool {
				if skipped < n {
					skipped++
					return true
				}
				return yield(v)
			})
		},
	}
}

func (s *seqStream[T]) TakeWhile(predicate function.Predicate[T]) streamImpl[T] {
	upstream := s.seq
	return &seqStream[T]{
		seq: func(yield func(T) bool) {
			upstream(func(v T) bool {
				if !predicate(v) {
					return false
				}
				return yield(v)
			})
		},
	}
}

func (s *seqStream[T]) DropWhile(predicate function.Predicate[T]) streamImpl[T] {
	upstream := s.seq
	return &seqStream[T]{
		seq: func(yield func(T) bool) {
			dropping := true
			upstream(func(v T) bool {
				if dropping {
					if predicate(v) {
						return true
					}
					dropping = false
				}
				return yield(v)
			})
		},
	}
}

// --- Terminal operations ---

func (s *seqStream[T]) ForEach(action function.Consumer[T]) {
	s.seq(func(v T) bool {
		action(v)
		return true
	})
}

func (s *seqStream[T]) ToSlice() []T {
	var result []T
	s.seq(func(v T) bool {
		result = append(result, v)
		return true
	})
	return result
}

func (s *seqStream[T]) Reduce(
	identity T,
	accumulator function.BinaryOperator[T],
) T {
	result := identity
	s.seq(func(v T) bool {
		result = accumulator(result, v)
		return true
	})
	return result
}

func (s *seqStream[T]) ReduceToOptional(
	accumulator function.BinaryOperator[T],
) *Optional[T] {
	foundAny := false
	var result T
	s.seq(func(v T) bool {
		if !foundAny {
			foundAny = true
			result = v
		} else {
			result = accumulator(result, v)
		}
		return true
	})
	if foundAny {
		return OptionalOf(result)
	}
	return OptionalEmpty[T]()
}

func (s *seqStream[T]) Min(less Less[T]) *Optional[T] {
	foundAny := false
	var result T
	s.seq(func(v T) bool {
		if !foundAny {
			foundAny = true
			result = v
		} else if less(v, result) {
			result = v
		}
		return true
	})
	if foundAny {
		return OptionalOf(result)
	}
	return OptionalEmpty[T]()
}

func (s *seqStream[T]) Max(less Less[T]) *Optional[T] {
	foundAny := false
	var result T
	s.seq(func(v T) bool {
		if !foundAny {
			foundAny = true
			result = v
		} else if less(result, v) {
			result = v
		}
		return true
	})
	if foundAny {
		return OptionalOf(result)
	}
	return OptionalEmpty[T]()
}

func (s *seqStream[T]) Count() int {
	count := 0
	s.seq(func(T) bool {
		count++
		return true
	})
	return count
}

func (s *seqStream[T]) AnyMatch(predicate function.Predicate[T]) bool {
	found := false
	s.seq(func(v T) bool {
		if predicate(v) {
			found = true
			return false
		}
		return true
	})
	return found
}

func (s *seqStream[T]) AllMatch(predicate function.Predicate[T]) bool {
	allMatch := true
	s.seq(func(v T) bool {
		if !predicate(v) {
			allMatch = false
			return false
		}
		return true
	})
	return allMatch
}

func (s *seqStream[T]) NoneMatch(predicate function.Predicate[T]) bool {
	return !s.AnyMatch(predicate)
}

func (s *seqStream[T]) FindFirst() *Optional[T] {
	var result T
	found := false
	s.seq(func(v T) bool {
		result = v
		found = true
		return false
	})
	if found {
		return OptionalOf(result)
	}
	return OptionalEmpty[T]()
}

func (s *seqStream[T]) FindAny() *Optional[T] {
	return s.FindFirst()
}
