package elevator

import "testing"

func TestStepMovesElevatorAfterRequest(t *testing.T) {
	system, err := NewSystem(20, 5)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	if err := system.AddRequest(4, DirectionUp, RequestKindHall); err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
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
	if len(system.PendingRequests) != 0 {
		t.Fatalf("pending request count = %d, want 0", len(system.PendingRequests))
	}
}

func TestStepOpensDoorAfterReachingTarget(t *testing.T) {
	system, err := NewSystem(20, 5)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	if err := system.AddRequest(2, DirectionUp, RequestKindHall); err != nil {
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
}
