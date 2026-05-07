package api

import (
	"encoding/json"
	"net/http"
	"os_sp26_proj1/internal/elevator"
)

// Server 持有所有 HTTP handler 需要的依赖。
// 后续新增的依赖（配置、日志等）只需要在这里加字段，不影响 handler 函数签名。
type Server struct {
	System *elevator.System
}

// RegisterRoutes 把所有 API 路由注册到 mux 上。
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/request", s.handleRequest)
	mux.HandleFunc("/api/step", s.handleStep)
	mux.Handle("/", http.FileServer(http.Dir("web")))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]string{
		"status": "ok",
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleState 以 JSON 格式返回电梯系统当前状态。
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := s.System.Snapshot()
	if err != nil {
		http.Error(w, "failed to get snapshot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Floor     int                  `json:"floor"`
		Direction elevator.Direction   `json:"direction"`
		Kind      elevator.RequestKind `json:"kind"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}
	if request.Floor < 1 || request.Floor > s.System.FloorCount {
		http.Error(w, "floor out of range", http.StatusBadRequest)
		return
	}
	if !elevator.IsValidDirection(request.Direction) {
		http.Error(w, "invalid direction", http.StatusBadRequest)
		return
	}
	if !elevator.IsValidRequestKind(request.Kind) {
		http.Error(w, "invalid request kind", http.StatusBadRequest)
		return
	}
	createdRequest, err := s.System.AddRequest(request.Floor, request.Direction, request.Kind)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"status":      "accepted",
		"currentTick": s.System.CurrentTick,
		"request":     createdRequest,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.System.Step(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := s.System.Snapshot()
	if err != nil {
		http.Error(w, "failed to get snapshot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}
