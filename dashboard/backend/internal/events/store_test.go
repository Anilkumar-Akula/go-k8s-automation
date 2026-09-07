package events

import "testing"

func TestSinceReturnsOnlyNewerOldestFirst(t *testing.T) {
	s := NewStore(10)
	s.Add(Event{Message: "one"})
	s.Add(Event{Message: "two"})
	s.Add(Event{Message: "three"})

	got := s.Since(1) // after "one"
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Message != "two" || got[1].Message != "three" {
		t.Fatalf("got %+v, want oldest-first [two three]", got)
	}

	if got := s.Since(3); len(got) != 0 {
		t.Fatalf("Since(latest) = %+v, want empty", got)
	}
}
