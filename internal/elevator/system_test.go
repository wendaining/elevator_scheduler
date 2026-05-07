package elevator

import "testing"

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
	if system.Requests[0].Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q", system.Requests[0].Status, RequestAssigned)
	}
	if system.Requests[0].AssignedTick != 0 {
		t.Fatalf("assigned tick = %d, want 0", system.Requests[0].AssignedTick)
	}
	if system.Requests[0].AssignedElevatorID != 1 {
		t.Fatalf("assigned elevator ID = %d, want 1", system.Requests[0].AssignedElevatorID)
	}
}

func TestStepOpensDoorAfterReachingTarget(t *testing.T) {
	system, err := NewSystem(20, 5, 1, 2, 1)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	if _, err := system.AddRequest(2, DirectionUp, RequestKindHall); err != nil {
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
	if system.Requests[0].Status != RequestDone {
		t.Fatalf("request status = %q, want %q", system.Requests[0].Status, RequestDone)
	}
	if system.Requests[0].CompletedTick != 1 {
		t.Fatalf("completed tick = %d, want 1", system.Requests[0].CompletedTick)
	}
}

func TestStepUsesTicksPerFloor(t *testing.T) {
	system, err := NewSystem(20, 5, 3, 2, 1)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	if _, err := system.AddRequest(2, DirectionUp, RequestKindHall); err != nil {
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
	if system.Requests[0].Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q before door opens", system.Requests[0].Status, RequestAssigned)
	}

	if err := system.Step(); err != nil {
		t.Fatalf("fourth Step returned error: %v", err)
	}
	if system.Requests[0].Status != RequestDone {
		t.Fatalf("request status = %q, want %q after door opens", system.Requests[0].Status, RequestDone)
	}
	if system.Requests[0].CompletedTick != 3 {
		t.Fatalf("completed tick = %d, want 3", system.Requests[0].CompletedTick)
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
