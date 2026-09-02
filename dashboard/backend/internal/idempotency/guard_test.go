package idempotency

import (
	"testing"
	"time"
)

func TestGuard(t *testing.T) {
	g := NewGuard(50 * time.Millisecond)

	if _, _, found := g.Check("k1"); found {
		t.Fatal("unseen key should not be found")
	}

	g.Record("k1", 200, []byte(`{"ok":true}`))
	status, body, found := g.Check("k1")
	if !found || status != 200 || string(body) != `{"ok":true}` {
		t.Fatalf("Check() = %d, %s, %v; want 200, {\"ok\":true}, true", status, body, found)
	}

	// Empty key is always a miss — no Idempotency-Key header sent.
	g.Record("", 200, []byte("x"))
	if _, _, found := g.Check(""); found {
		t.Fatal("empty key should never be found")
	}

	time.Sleep(60 * time.Millisecond)
	if _, _, found := g.Check("k1"); found {
		t.Fatal("expired entry should not be found")
	}
}
