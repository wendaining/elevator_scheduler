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

func TestStartAutoStepAdvancesSystemTick(t *testing.T) {
	server := newTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

func newTestServer(t *testing.T) *Server {
	t.Helper()

	system, err := elevator.NewSystem(elevator.SystemConfig{
		Floors:           20,
		ElevatorCount:    5,
		TicksPerFloor:    1,
		DoorBaseTicks:    2,
		TickPerPassenger: 1,
		DatabasePath:     ":memory:",
	})
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := system.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})

	return &Server{System: system}
}

func assertTextError(t *testing.T, body string) {
	t.Helper()

	if strings.TrimSpace(body) == "" {
		t.Fatal("error response body is empty")
	}
}
