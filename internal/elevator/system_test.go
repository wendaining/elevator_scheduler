package elevator

import "testing"

func TestStepMovesElevatorAfterRequest(t *testing.T) {
	system, err := NewSystem(20, 5)
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
	if system.Requests[0].AssignedTick != 1 {
		t.Fatalf("assigned tick = %d, want 1", system.Requests[0].AssignedTick)
	}
	if system.Requests[0].AssignedElevatorID != 1 {
		t.Fatalf("assigned elevator ID = %d, want 1", system.Requests[0].AssignedElevatorID)
	}
}

func TestStepOpensDoorAfterReachingTarget(t *testing.T) {
	system, err := NewSystem(20, 5)
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
	if len(firstElevator.TargetFloors) != 0 {
		t.Fatalf("target floor count = %d, want 0", len(firstElevator.TargetFloors))
	}
	if system.Requests[0].Status != RequestDone {
		t.Fatalf("request status = %q, want %q", system.Requests[0].Status, RequestDone)
	}
	if system.Requests[0].CompletedTick != 2 {
		t.Fatalf("completed tick = %d, want 2", system.Requests[0].CompletedTick)
	}
}
