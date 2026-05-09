package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload createRequestPayload

	if err := decodeJSONBody(r, &payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateCreateRequestPayload(payload, s.System.FloorCount); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	createdRequest, err := s.System.AddRequest(payload.Floor, payload.Direction, payload.Kind)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
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

// createRequestPayload 是 POST /api/request 接收的请求体。
// 客户端只提交它实际知道的信息；ID、Status 和 tick 都由后端创建。
type createRequestPayload struct {
	Floor     int                  `json:"floor"`
	Direction elevator.Direction   `json:"direction"`
	Kind      elevator.RequestKind `json:"kind"`
}

// validateCreateRequestPayload 检查客户端提交的新请求是否合法。
func validateCreateRequestPayload(payload createRequestPayload, floorCount int) error {
	if payload.Floor < 1 || payload.Floor > floorCount {
		return fmt.Errorf("floor must be between 1 and %d, got %d", floorCount, payload.Floor)
	}
	if !elevator.IsValidRequestKind(payload.Kind) {
		return fmt.Errorf("kind must be hall or cabin, got %q", payload.Kind)
	}
	if !elevator.IsValidDirection(payload.Direction) {
		return fmt.Errorf("direction must be up, down, or idle, got %q", payload.Direction)
	}
	if payload.Kind == elevator.RequestKindHall && payload.Direction == elevator.DirectionIdle {
		return errors.New("hall request direction must be up or down")
	}
	if payload.Kind == elevator.RequestKindCabin && payload.Direction != elevator.DirectionIdle {
		return errors.New("cabin request direction must be idle")
	}
	return nil
}

// decodeJSONBody 解析 JSON 请求体，并拒绝未知字段和多个 JSON 对象。
func decodeJSONBody(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body cannot be empty")
		}
		return fmt.Errorf("invalid JSON request body: %w", err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain only one JSON object")
	}
	return nil
}

// writeJSONError 用统一 JSON 格式返回 API 错误。
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
