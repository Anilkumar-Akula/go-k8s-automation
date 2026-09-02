package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"dashboard-api/internal/actions"
	"dashboard-api/internal/audit"
	"dashboard-api/internal/controllers"
	"dashboard-api/internal/metrics"
)

var errNoRecommendation = errors.New("no cached recommendation for that namespace/pod/container")

func findRecommendation(recs []controllers.Recommendation, namespace, pod, container string) (controllers.Recommendation, bool) {
	for _, r := range recs {
		if r.Namespace == namespace && r.Pod == pod && r.Container == container {
			return r, true
		}
	}
	return controllers.Recommendation{}, false
}

type actionResult struct {
	Status   string      `json:"status"`
	OldValue string      `json:"oldValue,omitempty"`
	NewValue string      `json:"newValue,omitempty"`
	Audit    audit.Event `json:"audit"`
}

type actionError struct {
	Error string `json:"error"`
}

// runAction is the shared pipeline every action handler goes through:
// idempotency replay -> execute -> audit -> metrics. project/action
// label the Prometheus metrics and the audit record; namespace/resource
// identify what was acted on for the audit trail.
func (s *Server) runAction(
	w http.ResponseWriter, r *http.Request,
	project, action, namespace, resource, reason string,
	execute func(ctx context.Context) (oldValue, newValue string, err error),
) {
	key := r.Header.Get("Idempotency-Key")
	if status, body, found := s.idem.Check(key); found {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(body)
		return
	}

	metrics.ActionsTotal.WithLabelValues(project, action).Inc()
	start := time.Now()
	oldValue, newValue, err := execute(r.Context())
	metrics.ActionDurationSeconds.WithLabelValues(project, action).Observe(time.Since(start).Seconds())

	status := "success"
	if err != nil {
		status = "failed"
		metrics.ActionsFailedTotal.WithLabelValues(project, action).Inc()
	} else {
		metrics.ActionsSuccessTotal.WithLabelValues(project, action).Inc()
	}

	entry, auditErr := s.audit.Insert(r.Context(), audit.Event{
		Actor: s.cfg.Actor, Action: action, Project: project,
		Namespace: namespace, Resource: resource,
		OldValue: oldValue, NewValue: newValue, Reason: reason, Status: status,
	})
	if auditErr != nil {
		// Auditing is best-effort logging, not a reason to fail an
		// otherwise-successful action or hide one that failed.
		entry = audit.Event{Actor: s.cfg.Actor, Action: action, Project: project, Namespace: namespace, Resource: resource, Status: status}
	}

	var httpStatus int
	var respBody []byte
	if err != nil {
		httpStatus = http.StatusBadGateway
		respBody, _ = json.Marshal(actionError{Error: err.Error()})
	} else {
		httpStatus = http.StatusOK
		respBody, _ = json.Marshal(actionResult{Status: status, OldValue: oldValue, NewValue: newValue, Audit: entry})
	}

	s.idem.Record(key, httpStatus, respBody)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	w.Write(respBody)
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(actionError{Error: "invalid request body: " + err.Error()})
		var zero T
		return zero, false
	}
	return v, true
}

func writeValidationError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(actionError{Error: err.Error()})
}

func (s *Server) handleActionRestart(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[actions.RestartRequest](w, r)
	if !ok {
		return
	}
	if err := req.Validate(); err != nil {
		writeValidationError(w, err)
		return
	}
	s.runAction(w, r, "auto-healer", "restart", req.Namespace, req.Pod, req.Reason,
		func(ctx context.Context) (string, string, error) {
			err := s.executor.RestartPod(ctx, req.Namespace, req.Pod)
			return "", "", err
		})
}

func (s *Server) handleActionRollback(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[actions.RollbackRequest](w, r)
	if !ok {
		return
	}
	if err := req.Validate(); err != nil {
		writeValidationError(w, err)
		return
	}
	s.runAction(w, r, "rollout-manager", "rollback", req.Namespace, req.Deployment, req.Reason,
		func(ctx context.Context) (string, string, error) {
			return s.executor.RollbackDeployment(ctx, req.Namespace, req.Deployment)
		})
}

func (s *Server) handleActionApply(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[actions.ApplyRequest](w, r)
	if !ok {
		return
	}
	if err := req.Validate(); err != nil {
		writeValidationError(w, err)
		return
	}

	rec, found := findRecommendation(s.cache.Snapshot().Optimizer.Recommendations, req.Namespace, req.Pod, req.Container)
	if !found {
		writeValidationError(w, errNoRecommendation)
		return
	}

	s.runAction(w, r, "resource-optimizer", "apply", req.Namespace, req.Pod+"/"+req.Container, req.Reason,
		func(ctx context.Context) (string, string, error) {
			return s.executor.ApplyRecommendation(ctx, req.Namespace, req.Pod, req.Container, actions.RecommendedResources{
				ReqCPUMilli: int64(rec.ReqCPUMilli), LimCPUMilli: int64(rec.LimCPUMilli),
				ReqMemBytes: int64(rec.ReqMemBytes), LimMemBytes: int64(rec.LimMemBytes),
			})
		})
}

func (s *Server) handleActionScale(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[actions.ScaleRequest](w, r)
	if !ok {
		return
	}
	if err := req.Validate(s.cfg.ActionsMaxReplicas); err != nil {
		writeValidationError(w, err)
		return
	}
	s.runAction(w, r, "autoscaler", "scale", req.Namespace, req.Deployment, req.Reason,
		func(ctx context.Context) (string, string, error) {
			return s.executor.ScaleDeployment(ctx, req.Namespace, req.Deployment, req.Replicas)
		})
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	events, err := s.audit.List(r.Context(), limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(actionError{Error: err.Error()})
		return
	}
	writeJSON(w, events)
}
