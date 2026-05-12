package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os_sp26_proj1/internal/elevator"
	"sync"
	"time"
)

// Server 持有所有 HTTP handler 需要的依赖。
type Server struct {
	System *elevator.System

	mu               sync.Mutex
	config           elevator.SystemConfig
	baseCtx          context.Context
	autoStepCancel   context.CancelFunc
	autoStepInterval time.Duration
	autoStepStarted  bool
}

// NewServer 创建 API server，并保存重建系统所需的配置。
func NewServer(system *elevator.System, config elevator.SystemConfig) *Server {
	return &Server{
		System: system,
		config: config,
	}
}

// RegisterRoutes 把所有 API 路由注册到 mux 上。
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/request", s.handleRequest)
	mux.HandleFunc("/api/scheduler", s.handleScheduler)
	mux.HandleFunc("/api/floor-count", s.handleFloorCount)
	mux.HandleFunc("/api/elevator-count", s.handleElevatorCount)
	mux.Handle("/", http.FileServer(http.Dir("web")))
}

// restartSystemLocked 用新的楼层数和电梯数重建整个电梯系统。
// 调用者必须持有 s.mu。
func (s *Server) restartSystemLocked(floorCount, elevatorCount int) error {
	if floorCount < 2 || floorCount > 40 {
		return fmt.Errorf("floor count must be between 2 and 40, got %d", floorCount)
	}
	if elevatorCount < 1 || elevatorCount > 10 {
		return fmt.Errorf("elevator count must be between 1 and 10, got %d", elevatorCount)
	}

	// 用新参数重建 System
	newConfig := s.config
	newConfig.Floors = floorCount
	newConfig.ElevatorCount = elevatorCount
	newSystem, err := elevator.NewSystem(newConfig)
	if err != nil {
		return err
	}

	// 为新 System 启动电梯 goroutine。
	// 如果 baseCtx 尚未被 StartAutoStep 设置（例如测试路径），用 Background 兜底。
	runnerCtx := s.baseCtx
	if runnerCtx == nil {
		runnerCtx = context.Background()
	}
	newSystem.StartElevatorRunners(runnerCtx)

	// 新系统已就绪，停止旧 auto-step 并替换
	if s.autoStepCancel != nil {
		s.autoStepCancel()
		s.autoStepCancel = nil
	}

	oldSystem := s.System
	s.System = newSystem

	if oldSystem != nil {
		oldSystem.Close()
	}

	if s.autoStepStarted {
		s.startAutoStepLocked()
	}

	return nil
}

// handleConfig 返回前端轮询和展示需要的运行配置。
//
// 前端不要硬编码轮询间隔，而是读取 autoStepIntervalMs。
// 这样后续只改后端默认节奏，页面刷新频率会自动保持同步。
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	response := map[string]any{
		"autoStepIntervalMs": s.autoStepInterval.Milliseconds(),
		"ticksPerFloor":      s.config.TicksPerFloor,
		"doorBaseTicks":      s.config.DoorBaseTicks,
		"tickPerPassenger":   s.config.TickPerPassenger,
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
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

	s.mu.Lock()
	data, err := s.System.Snapshot()
	s.mu.Unlock()

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

	var payload createRequestPayload

	if err := decodeJSONBody(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validateCreateRequestPayload(payload, s.System.FloorCount); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req, err := s.System.AddRequest(payload.Floor, payload.Direction, payload.Kind)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"status":      "accepted",
		"currentTick": req.CreatedTick,
		"request":     *req,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleScheduler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type SchedulerPayload struct {
		Name string `json:"name"`
	}
	var schedulerPayload SchedulerPayload
	if err := decodeJSONBody(r, &schedulerPayload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.System.SetScheduler(schedulerPayload.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"status": "scheduler switched",
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleFloorCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type floorCountPayload struct {
		FloorCount int `json:"floorCount"`
	}
	var p floorCountPayload
	if err := decodeJSONBody(r, &p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	elevatorCount := len(s.System.Elevators)
	if err := s.restartSystemLocked(p.FloorCount, elevatorCount); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "system restarted",
		"floorCount": p.FloorCount,
	})
}

func (s *Server) handleElevatorCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type elevatorCountPayload struct {
		ElevatorCount int `json:"elevatorCount"`
	}
	var p elevatorCountPayload
	if err := decodeJSONBody(r, &p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	floorCount := s.System.FloorCount
	if err := s.restartSystemLocked(floorCount, p.ElevatorCount); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{
		"status":        "system restarted",
		"elevatorCount": p.ElevatorCount,
	})
}

// createRequestPayload 是 POST /api/request 接收的请求体。
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
