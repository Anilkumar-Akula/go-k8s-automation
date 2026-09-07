package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDisabledAuthAllowsEverything(t *testing.T) {
	a := New("")
	if a.Enabled() {
		t.Fatal("expected auth disabled for empty token config")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/actions/healer/restart", nil)
	rec := httptest.NewRecorder()
	a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with auth disabled, got %d", rec.Code)
	}
}

func TestViewerCannotWrite(t *testing.T) {
	a := New("v-token:alice:viewer,o-token:bob:operator")

	post := func(token string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/actions/healer/restart", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)
		return rec.Code
	}

	if code := post("v-token"); code != http.StatusForbidden {
		t.Fatalf("viewer POST: want 403, got %d", code)
	}
	if code := post("o-token"); code != http.StatusOK {
		t.Fatalf("operator POST: want 200, got %d", code)
	}
	if code := post("bogus"); code != http.StatusUnauthorized {
		t.Fatalf("unknown token: want 401, got %d", code)
	}
}
