package gostream

import "testing"

func TestBuilder(t *testing.T) {
	var builder Builder[int]

	for i := range 100 {
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
