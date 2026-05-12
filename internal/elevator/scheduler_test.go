package elevator

import (
	"path/filepath"
	"testing"
)

func TestFCFSSchedulerAssignsLowestPendingRequestID(t *testing.T) {
	system := newSchedulerTestSystem(t, 2)
	if err := system.SetScheduler("fcfs"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}

	system.Requests = map[int64]*Request{
		9: {ID: 9, Floor: 9, Direction: DirectionUp, Kind: RequestKindHall, Status: RequestPending},
		2: {ID: 2, Floor: 4, Direction: DirectionUp, Kind: RequestKindHall, Status: RequestPending},
		5: {ID: 5, Floor: 7, Direction: DirectionDown, Kind: RequestKindHall, Status: RequestAssigned},
	}

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	if system.Requests[2].Status != RequestAssigned {
		t.Fatalf("request 2 status = %q, want %q", system.Requests[2].Status, RequestAssigned)
	}
	if system.Requests[9].Status != RequestPending {
		t.Fatalf("request 9 status = %q, want %q", system.Requests[9].Status, RequestPending)
	}
	if got := system.Requests[2].AssignedElevatorID; got != 1 {
		t.Fatalf("assigned elevator ID = %d, want 1", got)
	}
}

func TestFCFSSchedulerSkipsBusyAndEmergencyElevators(t *testing.T) {
	system := newSchedulerTestSystem(t, 3)
	if err := system.SetScheduler("fcfs"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}

	system.Elevators[0].Stops = []StopPlan{
		{Floor: 8, Reason: StopReasonCabin, Direction: DirectionIdle, RequestIDs: []int64{99}},
	}
	system.Elevators[1].EmergencyStop = true

	request, err := system.AddRequest(6, DirectionUp, RequestKindHall, 0)
	if err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
	}

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	if request.AssignedElevatorID != 3 {
		t.Fatalf("assigned elevator ID = %d, want 3", request.AssignedElevatorID)
	}
	if len(system.Elevators[2].Stops) != 1 || system.Elevators[2].Stops[0].Floor != 6 {
		t.Fatalf("third elevator stops = %v, want one stop at floor 6", system.Elevators[2].Stops)
	}
}

func TestFirstAvailableSchedulerLeavesRequestPendingWhenFirstElevatorBusy(t *testing.T) {
	system := newSchedulerTestSystem(t, 2)

	system.Elevators[0].Stops = []StopPlan{
		{Floor: 8, Reason: StopReasonCabin, Direction: DirectionIdle, RequestIDs: []int64{99}},
	}
	request, err := system.AddRequest(6, DirectionUp, RequestKindHall, 0)
	if err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
	}

	if system.scheduler.Assign(system) {
		t.Fatal("Assign returned true, want false")
	}
	if request.Status != RequestPending {
		t.Fatalf("request status = %q, want %q", request.Status, RequestPending)
	}
}

func TestDuplicateRequestsCanShareSameStopPlan(t *testing.T) {
	system := newSchedulerTestSystem(t, 1)
	if err := system.SetScheduler("fcfs"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}

	first, err := system.AddRequest(5, DirectionUp, RequestKindHall, 0)
	if err != nil {
		t.Fatalf("first AddRequest returned error: %v", err)
	}
	second, err := system.AddRequest(5, DirectionUp, RequestKindHall, 0)
	if err != nil {
		t.Fatalf("second AddRequest returned error: %v", err)
	}

	if first.ID == second.ID {
		t.Fatalf("duplicate requests received same ID %d", first.ID)
	}

	system.assignRequestToElevator(first.ID, 0)
	system.assignRequestToElevator(second.ID, 0)

	stops := system.Elevators[0].Stops
	if len(stops) != 1 {
		t.Fatalf("stop count = %d, want 1", len(stops))
	}
	if stops[0].Floor != 5 || stops[0].Reason != StopReasonHallUp {
		t.Fatalf("stop = %+v, want floor 5 hall_up", stops[0])
	}
	if len(stops[0].RequestIDs) != 2 {
		t.Fatalf("request IDs = %v, want two IDs", stops[0].RequestIDs)
	}
}

func newSchedulerTestSystem(t *testing.T, elevatorCount int) *System {
	t.Helper()

	system, err := NewSystem(SystemConfig{
		Floors:           20,
		ElevatorCount:    elevatorCount,
		TicksPerFloor:    1,
		DoorBaseTicks:    2,
		TickPerPassenger: 1,
		DatabasePath:     filepath.Join(t.TempDir(), "requests.db"),
	})
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := system.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})

	return system
}
