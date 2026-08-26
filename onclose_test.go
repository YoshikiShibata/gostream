// Copyright © 2026 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"slices"
	"testing"
)

func TestStream_OnClose_RunsHandlerOnce(t *testing.T) {
	count := 0
	s := Of(1, 2, 3).OnClose(func() { count++ })
	s.Close()
	s.Close() // second call is a no-op
	if count != 1 {
		t.Errorf("handler ran %d times, want 1", count)
	}
}

func TestStream_OnClose_RunsHandlersInOrder(t *testing.T) {
	var order []int
	s := Of(1, 2, 3).
		OnClose(func() { order = append(order, 1) }).
		OnClose(func() { order = append(order, 2) }).
		OnClose(func() { order = append(order, 3) })
	s.Close()
	want := []int{1, 2, 3}
	if !slices.Equal(order, want) {
		t.Errorf("order %v, want %v", order, want)
	}
}

func TestStream_OnClose_HandlersSharedAcrossIntermediates(t *testing.T) {
	// A handler registered before intermediate operations should still
	// fire when close is called on the derived Stream.
	closed := false
	derived := Of(1, 2, 3, 4).
		OnClose(func() { closed = true }).
		Filter(func(v int) bool { return v > 1 }).
		Map(func(v int) int { return v * 2 })
	_ = derived.ToSlice()
	derived.Close()
	if !closed {
		t.Errorf("handler did not run when derived stream closed")
	}
}

func TestStream_OnClose_NoHandlerIsFine(t *testing.T) {
	// Close on a stream with no handlers should not panic.
	Of(1, 2, 3).Close()
}
