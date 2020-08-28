// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

// Builder is a mutable builder for a Stream. This allows the creation of a
// Stream by generating elements individually and adding them to the Builder.
type Builder[T any] struct {
	built bool
	data  []T
}

// Add adds an element to the stream being built.
func (b *Builder[T]) Add(t T) {
	if b.built {
		panic("Already built state")
	}
	b.data = append(b.data, t)
}

// Build builds the stream, transitioning this builder to the built state.
// If there are further attempts to operate on the builder after it has
// entered the built state, then panic.
func (b *Builder[T]) Build() Stream[T] {
	if b.built {
		panic("Already build state")
	}
	b.built = true
	if len(b.data) == 0 {
		return Empty[T]()
	}
	return Of(b.data...)
}
