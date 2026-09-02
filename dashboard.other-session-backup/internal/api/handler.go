package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"k8s-automation-dashboard/internal/collector"
	"k8s-automation-dashboard/internal/models"
)

type Server struct {
	manager  *collector.Manager
	webDir   string
	mux      *http.ServeMux
}

func NewServer(manager *collector.Manager, webDir string) *Server {
	s := &Server{
		manager: manager,
		webDir:  webDir,
		mux:     http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	// REST APIs
	s.mux.HandleFunc("GET /api/overview", s.handleOverview)
	s.mux.HandleFunc("GET /api/healer", s.handleHealer)
	s.mux.HandleFunc("POST /api/healer/remediate", s.handleHealerRemediate)
	s.mux.HandleFunc("GET /api/rollouts", s.handleRollouts)
	s.mux.HandleFunc("POST /api/rollouts/rollback", s.handleRolloutsRollback)
	s.mux.HandleFunc("GET /api/optimizer", s.handleOptimizer)
	s.mux.HandleFunc("POST /api/optimizer/apply", s.handleOptimizerApply)
	s.mux.HandleFunc("GET /api/autoscaler", s.handleAutoscaler)
	s.mux.HandleFunc("POST /api/mode/toggle", s.handleToggleMode)
	s.mux.HandleFunc("GET /api/events", s.handleGetEvents)
	s.mux.HandleFunc("GET /api/events/stream", s.handleEventsStream)

	// Static Web Assets
	if s.webDir != "" {
		fs := http.FileServer(http.Dir(s.webDir))
		s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(s.webDir, filepath.Clean(r.URL.Path))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, filepath.Join(s.webDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	overview := s.manager.GetOverview(r.Context())
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) handleHealer(w http.ResponseWriter, r *http.Request) {
	pods := s.manager.GetPodHealth(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"pods": pods,
	})
}

func (s *Server) handleHealerRemediate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Namespace string `json:"namespace"`
		Pod       string `json:"pod"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	success, msg := s.manager.TriggerManualHeal(body.Namespace, body.Pod)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": success,
		"message": msg,
	})
}

func (s *Server) handleRollouts(w http.ResponseWriter, r *http.Request) {
	rollouts := s.manager.GetRollouts(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"rollouts": rollouts,
	})
}

func (s *Server) handleRolloutsRollback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Namespace  string `json:"namespace"`
		Deployment string `json:"deployment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	success, msg := s.manager.TriggerManualRollback(body.Namespace, body.Deployment)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": success,
		"message": msg,
	})
}

func (s *Server) handleOptimizer(w http.ResponseWriter, r *http.Request) {
	recs := s.manager.GetRecommendations(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"recommendations": recs,
	})
}

func (s *Server) handleOptimizerApply(w http.ResponseWriter, r *http.Request) {
	var body models.ApplyResourcePatchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	success, msg := s.manager.ApplyOptimizerPatch(body)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": success,
		"message": msg,
	})
}

func (s *Server) handleAutoscaler(w http.ResponseWriter, r *http.Request) {
	status := s.manager.GetAutoscalerStatus(r.Context())
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleToggleMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Demo bool `json:"demo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	s.manager.SetDemoMode(body.Demo)
	writeJSON(w, http.StatusOK, map[string]any{
		"demoMode": s.manager.IsDemoMode(),
		"message":  fmt.Sprintf("Mode switched to %s", map[bool]string{true: "Demo/Simulator", false: "Live Kubernetes"}[body.Demo]),
	})
}

func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	events := s.manager.GetEvents()
	writeJSON(w, http.StatusOK, map[string]any{
		"events": events,
	})
}

func (s *Server) handleEventsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	eventChan, unsubscribe := s.manager.Subscribe()
	defer unsubscribe()

	// Initial ping
	fmt.Fprintf(w, "event: ping\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case evt, open := <-eventChan:
			if !open {
				return
			}
			data, err := json.Marshal(evt)
			if err == nil {
				fmt.Fprintf(w, "event: automation-event\ndata: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}
