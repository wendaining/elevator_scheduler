package elevator

import (
	"path/filepath"
	"testing"
)

func TestSCANSchedulerAppendsRequestAlongTheWay(t *testing.T) {
	system := newSCANTestSystem(t, 1)
	system.Elevators[0].CurrentFloor = 3
	system.Elevators[0].ScanDirection = DirectionUp
	system.Elevators[0].Stops = []StopPlan{
		{Floor: 10, Reason: StopReasonHallUp, Direction: DirectionUp, RequestIDs: []int64{99}},
	}
	request := addSCANTestRequest(system, 6, DirectionUp, RequestKindHall)

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	stops := system.Elevators[0].Stops
	if len(stops) != 2 || stops[0].Floor != 6 || stops[1].Floor != 10 {
		t.Fatalf("stops = %v, want floors [6 10]", stops)
	}
	if request.Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q", request.Status, RequestAssigned)
	}
}

func TestSCANSchedulerChoosesNearestIdleWhenNoElevatorAlongTheWay(t *testing.T) {
	system := newSCANTestSystem(t, 2)
	system.Elevators[0].CurrentFloor = 1
	system.Elevators[0].ScanDirection = DirectionUp
	system.Elevators[1].CurrentFloor = 12
	system.Elevators[1].ScanDirection = DirectionUp
	request := addSCANTestRequest(system, 9, DirectionDown, RequestKindHall)

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	if request.AssignedElevatorID != 2 {
		t.Fatalf("assigned elevator ID = %d, want 2", request.AssignedElevatorID)
	}
	if system.Elevators[1].ScanDirection != DirectionDown {
		t.Fatalf("scan direction = %q, want %q", system.Elevators[1].ScanDirection, DirectionDown)
	}
}

func TestSCANSchedulerSortsDownStopsInDescendingOrder(t *testing.T) {
	system := newSCANTestSystem(t, 1)
	system.Elevators[0].CurrentFloor = 12
	system.Elevators[0].ScanDirection = DirectionDown
	system.Elevators[0].Stops = []StopPlan{
		{Floor: 2, Reason: StopReasonHallDown, Direction: DirectionDown, RequestIDs: []int64{99}},
	}
	request := addSCANTestRequest(system, 8, DirectionDown, RequestKindHall)

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	stops := system.Elevators[0].Stops
	if len(stops) != 2 || stops[0].Floor != 8 || stops[1].Floor != 2 {
		t.Fatalf("stops = %v, want floors [8 2]", stops)
	}
	if request.AssignedElevatorID != 1 {
		t.Fatalf("assigned elevator ID = %d, want 1", request.AssignedElevatorID)
	}
}

func TestSCANSchedulerUsesCostStopPenalty(t *testing.T) {
	system := newSCANTestSystem(t, 2)
	system.Elevators[0].CurrentFloor = 4
	system.Elevators[0].ScanDirection = DirectionUp
	system.Elevators[0].Stops = []StopPlan{
		{Floor: 5, Reason: StopReasonHallUp, Direction: DirectionUp, RequestIDs: []int64{90}},
		{Floor: 7, Reason: StopReasonHallUp, Direction: DirectionUp, RequestIDs: []int64{91}},
	}
	system.Elevators[1].CurrentFloor = 1
	system.Elevators[1].ScanDirection = DirectionUp
	request := addSCANTestRequest(system, 6, DirectionUp, RequestKindHall)

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	if request.AssignedElevatorID != 2 {
		t.Fatalf("assigned elevator ID = %d, want 2", request.AssignedElevatorID)
	}
}

func newSCANTestSystem(t *testing.T, elevatorCount int) *System {
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

	if err := system.SetScheduler("scan"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}
	return system
}

func addSCANTestRequest(system *System, floor int, direction Direction, kind RequestKind) *Request {
	request := Request{
		ID:          system.nextRequestID,
		Floor:       floor,
		Direction:   direction,
		Kind:        kind,
		Status:      RequestPending,
		CreatedTick: system.CurrentTick,
	}
	system.Requests[request.ID] = &request
	system.nextRequestID++
	return system.Requests[request.ID]
}
