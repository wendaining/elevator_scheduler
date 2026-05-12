package elevator

import (
	"path/filepath"
	"testing"
)

func TestLOOKSchedulerAppendsRequestAlongTheWay(t *testing.T) {
	system := newLOOKTestSystem(t, 1)
	system.Elevators[0].CurrentFloor = 3
	system.Elevators[0].ScanDirection = DirectionUp
	system.Elevators[0].Stops = []StopPlan{
		{Floor: 10, Reason: StopReasonHallUp, Direction: DirectionUp, RequestIDs: []int64{99}},
	}
	request := addLOOKTestRequest(system, 6, DirectionUp, RequestKindHall)

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

func TestLOOKSchedulerTurnsAroundWhenNoStopAhead(t *testing.T) {
	system := newLOOKTestSystem(t, 1)
	system.Elevators[0].CurrentFloor = 8
	system.Elevators[0].ScanDirection = DirectionUp
	system.Elevators[0].Stops = []StopPlan{
		{Floor: 3, Reason: StopReasonHallDown, Direction: DirectionDown, RequestIDs: []int64{99}},
	}
	request := addLOOKTestRequest(system, 6, DirectionDown, RequestKindHall)

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	elevator := system.Elevators[0]
	if elevator.ScanDirection != DirectionDown {
		t.Fatalf("scan direction = %q, want %q", elevator.ScanDirection, DirectionDown)
	}
	if len(elevator.Stops) != 2 || elevator.Stops[0].Floor != 6 || elevator.Stops[1].Floor != 3 {
		t.Fatalf("stops = %v, want floors [6 3]", elevator.Stops)
	}
	if request.AssignedElevatorID != 1 {
		t.Fatalf("assigned elevator ID = %d, want 1", request.AssignedElevatorID)
	}
}

func TestLOOKSchedulerUsesCostStopPenalty(t *testing.T) {
	system := newLOOKTestSystem(t, 2)
	system.Elevators[0].CurrentFloor = 4
	system.Elevators[0].ScanDirection = DirectionUp
	system.Elevators[0].Stops = []StopPlan{
		{Floor: 5, Reason: StopReasonHallUp, Direction: DirectionUp, RequestIDs: []int64{90}},
		{Floor: 7, Reason: StopReasonHallUp, Direction: DirectionUp, RequestIDs: []int64{91}},
	}
	system.Elevators[1].CurrentFloor = 1
	system.Elevators[1].ScanDirection = DirectionUp
	request := addLOOKTestRequest(system, 6, DirectionUp, RequestKindHall)

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	if request.AssignedElevatorID != 2 {
		t.Fatalf("assigned elevator ID = %d, want 2", request.AssignedElevatorID)
	}
}

func newLOOKTestSystem(t *testing.T, elevatorCount int) *System {
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

	if err := system.SetScheduler("look"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}
	return system
}

func addLOOKTestRequest(system *System, floor int, direction Direction, kind RequestKind) *Request {
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
