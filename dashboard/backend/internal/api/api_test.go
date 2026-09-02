package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s-automation-dashboard-backend/internal/events"
	"k8s-automation-dashboard-backend/internal/kubernetes"
	"k8s-automation-dashboard-backend/internal/models"
	"k8s-automation-dashboard-backend/internal/prometheus"
	"k8s-automation-dashboard-backend/internal/store"
)

func createTestServer() *Server {
	kube := kubernetes.NewClusterClient()
	auditStore := store.NewMemoryAuditStore()
	bus := events.NewEventBus()
	metrics := prometheus.NewMetricsCollector()
	return NewServer(kube, auditStore, bus, metrics)
}

func TestOverviewAPI(t *testing.T) {
	srv := createTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp models.OverviewResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode overview response: %v", err)
	}

	if len(resp.Controllers) != 4 {
		t.Errorf("expected 4 controller statuses, got %d", len(resp.Controllers))
	}
}

func TestHealerAPI(t *testing.T) {
	srv := createTestServer()

	// GET pods
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healer/pods", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// POST remediate
	body := []byte(`{"namespace":"production","pod":"auth-gateway-55d8c7-4k8p2"}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/healer/remediate", bytes.NewBuffer(body))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on remediate, got %d", postRec.Code)
	}
}

func TestRolloutAPI(t *testing.T) {
	srv := createTestServer()

	// GET deployments
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rollout/deployments", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// POST rollback
	body := []byte(`{"namespace":"production","deployment":"checkout-v2"}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/rollout/rollback", bytes.NewBuffer(body))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on rollback, got %d", postRec.Code)
	}
}

func TestOptimizerAPI(t *testing.T) {
	srv := createTestServer()

	// GET recommendations
	req := httptest.NewRequest(http.MethodGet, "/api/v1/optimizer/recommendations", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// POST apply
	body := []byte(`{"namespace":"production","deployment":"auth-gateway","container":"auth-proxy","reqCpuMilli":250,"limCpuMilli":600,"reqMemBytes":400000000,"limMemBytes":800000000}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/optimizer/apply", bytes.NewBuffer(body))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on apply, got %d", postRec.Code)
	}
}

func TestScalerAPI(t *testing.T) {
	srv := createTestServer()

	// GET status
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scaler/status", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// POST scale
	body := []byte(`{"namespace":"autoscaler-demo","deployment":"demo-app","replicas":5}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/scaler/scale", bytes.NewBuffer(body))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on scale, got %d", postRec.Code)
	}
}

func TestAuditLogsAPI(t *testing.T) {
	srv := createTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs?limit=5", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
