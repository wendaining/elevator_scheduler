package elevator

import (
	"path/filepath"
	"testing"
)

func TestStepMovesElevatorAfterRequest(t *testing.T) {
	system, err := NewSystem(20, 5, 1, 2, 1)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	request, err := system.AddRequest(4, DirectionUp, RequestKindHall)
	if err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
	}
	if request.ID != 1 {
		t.Fatalf("request ID = %d, want 1", request.ID)
	}
	if request.CreatedTick != 0 {
		t.Fatalf("created tick = %d, want 0", request.CreatedTick)
	}

	if err := system.Step(); err != nil {
		t.Fatalf("Step returned error: %v", err)
	}

	firstElevator := system.Elevators[0]
	if firstElevator.CurrentFloor != 2 {
		t.Fatalf("first elevator floor = %d, want 2", firstElevator.CurrentFloor)
	}
	if firstElevator.Direction != DirectionUp {
		t.Fatalf("first elevator direction = %q, want %q", firstElevator.Direction, DirectionUp)
	}
	if system.CurrentTick != 1 {
		t.Fatalf("current tick = %d, want 1", system.CurrentTick)
	}
	if request.Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q", request.Status, RequestAssigned)
	}
	if request.AssignedTick != 0 {
		t.Fatalf("assigned tick = %d, want 0", request.AssignedTick)
	}
	if request.AssignedElevatorID != 1 {
		t.Fatalf("assigned elevator ID = %d, want 1", request.AssignedElevatorID)
	}
}

func TestStepOpensDoorAfterReachingTarget(t *testing.T) {
	system, err := NewSystem(20, 5, 1, 2, 1)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	request, err := system.AddRequest(2, DirectionUp, RequestKindHall)
	if err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
	}

	if err := system.Step(); err != nil {
		t.Fatalf("first Step returned error: %v", err)
	}
	if err := system.Step(); err != nil {
		t.Fatalf("second Step returned error: %v", err)
	}

	firstElevator := system.Elevators[0]
	if firstElevator.CurrentFloor != 2 {
		t.Fatalf("first elevator floor = %d, want 2", firstElevator.CurrentFloor)
	}
	if firstElevator.Direction != DirectionIdle {
		t.Fatalf("first elevator direction = %q, want %q", firstElevator.Direction, DirectionIdle)
	}
	if !firstElevator.DoorOpen {
		t.Fatal("first elevator door is closed, want open")
	}
	if len(firstElevator.Stops) != 0 {
		t.Fatalf("stop count = %d, want 0", len(firstElevator.Stops))
	}

	if request.Status != RequestDone {
		t.Fatalf("request status = %q, want %q", request.Status, RequestDone)
	}
	if request.CompletedTick != 1 {
		t.Fatalf("completed tick = %d, want 1", request.CompletedTick)
	}

	// 运行态 Requests 中不应再包含已完成的请求
	if _, ok := system.Requests[request.ID]; ok {
		t.Fatal("completed request should not be in active Requests")
	}

	count, err := system.requestStore.CompletedRequestCount()
	if err != nil {
		t.Fatalf("CompletedRequestCount returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("completed request count = %d, want 1", count)
	}

	storedRequest, err := system.requestStore.CompletedRequestByID(request.ID)
	if err != nil {
		t.Fatalf("CompletedRequestByID returned error: %v", err)
	}
	if storedRequest.ID != request.ID {
		t.Fatalf("stored request ID = %d, want %d", storedRequest.ID, request.ID)
	}
	if storedRequest.Status != RequestDone {
		t.Fatalf("stored request status = %q, want %q", storedRequest.Status, RequestDone)
	}
	if storedRequest.CompletedTick != 1 {
		t.Fatalf("stored completed tick = %d, want 1", storedRequest.CompletedTick)
	}
}

func TestStepUsesTicksPerFloor(t *testing.T) {
	system, err := NewSystem(20, 5, 3, 2, 1)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	request, err := system.AddRequest(2, DirectionUp, RequestKindHall)
	if err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
	}

	for step := 1; step <= 2; step++ {
		if err := system.Step(); err != nil {
			t.Fatalf("Step %d returned error: %v", step, err)
		}
		if system.Elevators[0].CurrentFloor != 1 {
			t.Fatalf("after step %d floor = %d, want 1", step, system.Elevators[0].CurrentFloor)
		}
	}

	if err := system.Step(); err != nil {
		t.Fatalf("third Step returned error: %v", err)
	}
	if system.Elevators[0].CurrentFloor != 2 {
		t.Fatalf("after third step floor = %d, want 2", system.Elevators[0].CurrentFloor)
	}
	if request.Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q before door opens", request.Status, RequestAssigned)
	}

	if err := system.Step(); err != nil {
		t.Fatalf("fourth Step returned error: %v", err)
	}
	if request.Status != RequestDone {
		t.Fatalf("request status = %q, want %q after door opens", request.Status, RequestDone)
	}
	if request.CompletedTick != 3 {
		t.Fatalf("completed tick = %d, want 3", request.CompletedTick)
	}
	if _, ok := system.Requests[request.ID]; ok {
		t.Fatal("completed request should not be in active Requests")
	}
}

func TestNewSystemStoresTimingParameters(t *testing.T) {
	system, err := NewSystem(12, 3, 7, 4, 2)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	if system.FloorCount != 12 {
		t.Fatalf("floor count = %d, want 12", system.FloorCount)
	}
	if len(system.Elevators) != 3 {
		t.Fatalf("elevator count = %d, want 3", len(system.Elevators))
	}
	if system.TicksPerFloor != 7 {
		t.Fatalf("ticks per floor = %d, want 7", system.TicksPerFloor)
	}
	if system.DoorBaseTicks != 4 {
		t.Fatalf("door base ticks = %d, want 4", system.DoorBaseTicks)
	}
	if system.TickPerPassenger != 2 {
		t.Fatalf("tick per passenger = %d, want 2", system.TickPerPassenger)
	}
}

func TestNewSystemWithDatabaseContinuesRequestIDAfterRestart(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "requests.db")

	firstSystem, err := NewSystemWithDatabase(20, 5, 1, 2, 1, databasePath)
	if err != nil {
		t.Fatalf("NewSystemWithDatabase returned error: %v", err)
	}
	request, err := firstSystem.AddRequest(2, DirectionUp, RequestKindHall)
	if err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
	}
	if err := firstSystem.Step(); err != nil {
		t.Fatalf("first Step returned error: %v", err)
	}
	if err := firstSystem.Step(); err != nil {
		t.Fatalf("second Step returned error: %v", err)
	}
	if err := firstSystem.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	secondSystem, err := NewSystemWithDatabase(20, 5, 1, 2, 1, databasePath)
	if err != nil {
		t.Fatalf("second NewSystemWithDatabase returned error: %v", err)
	}
	defer secondSystem.Close()

	nextRequest, err := secondSystem.AddRequest(3, DirectionUp, RequestKindHall)
	if err != nil {
		t.Fatalf("second AddRequest returned error: %v", err)
	}
	if nextRequest.ID != request.ID+1 {
		t.Fatalf("next request ID = %d, want %d", nextRequest.ID, request.ID+1)
	}
}

func TestStopPlanKeepsSameFloorDifferentReasonsSeparate(t *testing.T) {
	elevator := Elevator{}

	addStopPlan(&elevator, StopPlan{
		Floor:      6,
		Reason:     StopReasonHallUp,
		Direction:  DirectionUp,
		RequestIDs: []int64{1},
	})
	addStopPlan(&elevator, StopPlan{
		Floor:      6,
		Reason:     StopReasonHallDown,
		Direction:  DirectionDown,
		RequestIDs: []int64{2},
	})
	addStopPlan(&elevator, StopPlan{
		Floor:      6,
		Reason:     StopReasonHallUp,
		Direction:  DirectionUp,
		RequestIDs: []int64{3},
	})

	if len(elevator.Stops) != 2 {
		t.Fatalf("stop count = %d, want 2", len(elevator.Stops))
	}
	if len(elevator.Stops[0].RequestIDs) != 2 {
		t.Fatalf("first stop request IDs = %v, want two IDs", elevator.Stops[0].RequestIDs)
	}
	if elevator.Stops[1].Reason != StopReasonHallDown {
		t.Fatalf("second stop reason = %q, want %q", elevator.Stops[1].Reason, StopReasonHallDown)
	}
}
