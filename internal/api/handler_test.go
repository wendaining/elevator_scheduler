package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os_sp26_proj1/internal/elevator"
	"strings"
	"testing"
	"time"
)

func TestHandleRequestCreatesBackendOwnedRequest(t *testing.T) {
	server := newTestServer(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/request",
		strings.NewReader(`{"floor":4,"direction":"up","kind":"hall"}`),
	)

	server.handleRequest(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var body struct {
		Status      string           `json:"status"`
		CurrentTick int              `json:"currentTick"`
		Request     elevator.Request `json:"request"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode response returned error: %v", err)
	}

	if body.Status != "accepted" {
		t.Fatalf("status = %q, want accepted", body.Status)
	}
	if body.Request.ID != 1 {
		t.Fatalf("request ID = %d, want 1", body.Request.ID)
	}
	if body.Request.Status != elevator.RequestPending {
		t.Fatalf("request status = %q, want %q", body.Request.Status, elevator.RequestPending)
	}
	if body.Request.CreatedTick != 0 {
		t.Fatalf("created tick = %d, want 0", body.Request.CreatedTick)
	}
	if _, ok := server.System.Requests[body.Request.ID]; !ok {
		t.Fatal("created request was not stored in active request map")
	}
}

func TestHandleRequestRejectsClientOwnedRequestFields(t *testing.T) {
	server := newTestServer(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/request",
		strings.NewReader(`{"id":99,"floor":4,"direction":"up","kind":"hall","status":"done"}`),
	)

	server.handleRequest(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
	assertTextError(t, response.Body.String())
}

func TestHandleRequestRejectsInvalidHallDirection(t *testing.T) {
	server := newTestServer(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/request",
		strings.NewReader(`{"floor":4,"direction":"idle","kind":"hall"}`),
	)

	server.handleRequest(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
	assertTextError(t, response.Body.String())
}

func TestHandleRequestRejectsInvalidCabinDirection(t *testing.T) {
	server := newTestServer(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/request",
		strings.NewReader(`{"floor":4,"direction":"up","kind":"cabin"}`),
	)

	server.handleRequest(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
	assertTextError(t, response.Body.String())
}

func TestHandleRequestRejectsInvalidJSON(t *testing.T) {
	server := newTestServer(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/request",
		strings.NewReader(`{"floor":4`),
	)

	server.handleRequest(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
	assertTextError(t, response.Body.String())
}

func TestHandleRequestRejectsInvalidFloor(t *testing.T) {
	server := newTestServer(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/request",
		strings.NewReader(`{"floor":21,"direction":"up","kind":"hall"}`),
	)

	server.handleRequest(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
	assertTextError(t, response.Body.String())
}

func TestHandleRequestAcceptsBoundaryFloors(t *testing.T) {
	server := newTestServer(t)

	bodies := []string{
		`{"floor":1,"direction":"up","kind":"hall"}`,
		`{"floor":20,"direction":"down","kind":"hall"}`,
	}

	for _, body := range bodies {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/request", strings.NewReader(body))

		server.handleRequest(response, request)

		if response.Code != http.StatusCreated {
			t.Fatalf("status code = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
		}
	}
}

func TestHandleRequestAllowsDuplicateRequestsWithDifferentIDs(t *testing.T) {
	server := newTestServer(t)

	first := createRequestForTest(t, server, `{"floor":4,"direction":"up","kind":"hall"}`)
	second := createRequestForTest(t, server, `{"floor":4,"direction":"up","kind":"hall"}`)

	if first.ID == second.ID {
		t.Fatalf("duplicate requests received same ID %d", first.ID)
	}
	if len(server.System.Requests) != 2 {
		t.Fatalf("active request count = %d, want 2", len(server.System.Requests))
	}
}

func TestStartAutoStepAdvancesSystemTick(t *testing.T) {
	server := newTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.System.StartElevatorRunners(ctx)
	server.StartAutoStep(ctx, time.Millisecond)

	deadline := time.After(200 * time.Millisecond)
	for {
		data, err := server.System.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot returned error: %v", err)
		}

		var snapshot struct {
			CurrentTick int `json:"currentTick"`
		}
		if err := json.Unmarshal(data, &snapshot); err != nil {
			t.Fatalf("Unmarshal snapshot returned error: %v", err)
		}
		if snapshot.CurrentTick > 0 {
			return
		}

		select {
		case <-deadline:
			t.Fatal("auto step did not advance CurrentTick")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestHandleConfigReturnsAutoStepInterval(t *testing.T) {
	server := newTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server.StartAutoStep(ctx, 250*time.Millisecond)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/config", nil)

	server.handleConfig(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var body struct {
		AutoStepIntervalMs int64 `json:"autoStepIntervalMs"`
		TicksPerFloor      int   `json:"ticksPerFloor"`
		DoorBaseTicks      int   `json:"doorBaseTicks"`
		TickPerPassenger   int   `json:"tickPerPassenger"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode response returned error: %v", err)
	}

	if body.AutoStepIntervalMs != 250 {
		t.Fatalf("auto step interval ms = %d, want 250", body.AutoStepIntervalMs)
	}
	if body.TicksPerFloor != 1 {
		t.Fatalf("ticks per floor = %d, want 1", body.TicksPerFloor)
	}
	if body.DoorBaseTicks != 2 {
		t.Fatalf("door base ticks = %d, want 2", body.DoorBaseTicks)
	}
	if body.TickPerPassenger != 1 {
		t.Fatalf("tick per passenger = %d, want 1", body.TickPerPassenger)
	}
}

func TestHandleSchedulerSwitchesScheduler(t *testing.T) {
	server := newTestServer(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/scheduler",
		strings.NewReader(`{"name":"nearest-idle"}`),
	)

	server.handleScheduler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if server.System.SchedulerName != "nearest-idle" {
		t.Fatalf("scheduler name = %q, want nearest-idle", server.System.SchedulerName)
	}
}

func TestHandleSchedulerRejectsUnknownScheduler(t *testing.T) {
	server := newTestServer(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/scheduler",
		strings.NewReader(`{"name":"unknown"}`),
	)

	server.handleScheduler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
	assertTextError(t, response.Body.String())
}

func TestRegisterRoutesDoesNotExposeManualStep(t *testing.T) {
	server := newTestServer(t)
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/step", nil)

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func createRequestForTest(t *testing.T, server *Server, body string) elevator.Request {
	t.Helper()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/request", strings.NewReader(body))

	server.handleRequest(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var decoded struct {
		Request elevator.Request `json:"request"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("Decode response returned error: %v", err)
	}
	return decoded.Request
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	config := elevator.SystemConfig{
		Floors:           20,
		ElevatorCount:    5,
		TicksPerFloor:    1,
		DoorBaseTicks:    2,
		TickPerPassenger: 1,
		DatabasePath:     ":memory:",
	}
	system, err := elevator.NewSystem(config)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := system.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})

	return NewServer(system, config)
}

func assertTextError(t *testing.T, body string) {
	t.Helper()

	if strings.TrimSpace(body) == "" {
		t.Fatal("error response body is empty")
	}
}
