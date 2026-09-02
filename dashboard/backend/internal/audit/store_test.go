package audit

import (
	"context"
	"testing"
)

func TestStoreInsertAndList(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	inserted, err := s.Insert(ctx, Event{
		Actor: "operator", Action: "scale", Project: "autoscaler",
		Namespace: "demo", Resource: "webapp",
		OldValue: "1", NewValue: "5", Reason: "load test", Status: "success",
	})
	if err != nil {
		t.Fatalf("Insert() error: %v", err)
	}
	if inserted.ID == "" {
		t.Fatal("Insert() did not assign an ID")
	}

	events, err := s.List(ctx, 10)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("List() returned %d events, want 1", len(events))
	}
	got := events[0]
	if got.ID != inserted.ID || got.Action != "scale" || got.NewValue != "5" {
		t.Errorf("List()[0] = %+v, want matching inserted event", got)
	}
}

func TestStoreListRespectsLimit(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	for range 5 {
		if _, err := s.Insert(ctx, Event{Actor: "operator", Action: "scale", Project: "autoscaler", Namespace: "ns", Resource: "r", Reason: "x", Status: "success"}); err != nil {
			t.Fatalf("Insert() error: %v", err)
		}
	}
	events, err := s.List(ctx, 3)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("List(3) returned %d events, want 3", len(events))
	}
}
