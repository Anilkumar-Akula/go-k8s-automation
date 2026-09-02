package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s-automation-dashboard/internal/collector"
	"k8s-automation-dashboard/internal/models"
)

func setupTestServer() *Server {
	mgr := collector.NewManager(nil, true)
	return NewServer(mgr, "")
}

func TestOverviewEndpoint(t *testing.T) {
	srv := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/overview", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var overview models.OverviewSummary
	if err := json.NewDecoder(rec.Body).Decode(&overview); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !overview.DemoMode {
		t.Errorf("expected demoMode to be true in mock setup")
	}
	if overview.TotalPodsMonitored == 0 {
		t.Errorf("expected monitored pods count > 0")
	}
}

func TestHealerEndpoints(t *testing.T) {
	srv := setupTestServer()

	// GET /api/healer
	req := httptest.NewRequest(http.MethodGet, "/api/healer", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// POST /api/healer/remediate
	payload := []byte(`{"namespace":"production","pod":"auth-gateway-55d8c7-4k8p2"}`)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/healer/remediate", bytes.NewBuffer(payload))
	reqPost.Header.Set("Content-Type", "application/json")
	recPost := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusOK {
		t.Fatalf("expected 200 on remediate, got %d", recPost.Code)
	}
}

func TestRolloutsEndpoints(t *testing.T) {
	srv := setupTestServer()

	// GET /api/rollouts
	req := httptest.NewRequest(http.MethodGet, "/api/rollouts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// POST /api/rollouts/rollback
	payload := []byte(`{"namespace":"production","deployment":"checkout-v2"}`)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/rollouts/rollback", bytes.NewBuffer(payload))
	reqPost.Header.Set("Content-Type", "application/json")
	recPost := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusOK {
		t.Fatalf("expected 200 on rollback, got %d", recPost.Code)
	}
}

func TestOptimizerEndpoints(t *testing.T) {
	srv := setupTestServer()

	// GET /api/optimizer
	req := httptest.NewRequest(http.MethodGet, "/api/optimizer", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// POST /api/optimizer/apply
	patchReq := models.ApplyResourcePatchRequest{
		Namespace:   "production",
		Deployment:  "auth-gateway",
		Container:   "auth-proxy",
		ReqCPUMilli: 250,
		LimCPUMilli: 600,
		ReqMemBytes: 384 * 1024 * 1024,
		LimMemBytes: 768 * 1024 * 1024,
	}
	body, _ := json.Marshal(patchReq)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/optimizer/apply", bytes.NewBuffer(body))
	reqPost.Header.Set("Content-Type", "application/json")
	recPost := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusOK {
		t.Fatalf("expected 200 on apply, got %d", recPost.Code)
	}
}

func TestAutoscalerEndpoint(t *testing.T) {
	srv := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/autoscaler", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var status models.AutoscalerStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode autoscaler status: %v", err)
	}

	if status.TargetCPUPercent != 50.0 {
		t.Errorf("expected TargetCPUPercent 50.0, got %v", status.TargetCPUPercent)
	}
}
