package gostream

import "testing"

func TestBuilder(t *testing.T) {
	defer trace("TestBuilder")()

	var builder Builder[int]

	for i := 0; i < 100; i++ {
		builder.Add(i)
	}
	want := 0
	builder.Build().ForEach(func(v int) {
		if v != want {
			t.Fatalf("v is %d, want %d", v, want)
		}
		want++
	})
}
