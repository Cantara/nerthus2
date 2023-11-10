package stack

import "testing"

func TestStack(t *testing.T) {
	inn := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	s := New[string]()
	for _, v := range inn {
		s.Push(v)
	}
	i := len(inn) - 1
	for v, ok := s.Pop(); ok; i-- {
		if v != inn[i] {
			t.Fatalf("incorrect pop value, %d: %s != %s", i, v, inn[i])
		}
	}
}
