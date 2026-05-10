package elevator

import (
	"path/filepath"
	"testing"
)

func TestEstimateAssignmentScoreBreakdown(t *testing.T) {
	system := newCostTestSystem(t)
	system.CurrentTick = 12

	elevator := Elevator{
		ID:           1,
		CurrentFloor: 3,
		Direction:    DirectionUp,
		Stops: []StopPlan{
			{Floor: 5, Reason: StopReasonHallUp, Direction: DirectionUp, RequestIDs: []int64{99}},
		},
	}
	request := Request{
		ID:          1,
		Floor:       9,
		Direction:   DirectionUp,
		Kind:        RequestKindHall,
		Status:      RequestPending,
		CreatedTick: 7,
	}

	score := EstimateAssignmentScore(system, elevator, request)

	if score.DistanceCost != 6 {
		t.Fatalf("distance cost = %d, want 6", score.DistanceCost)
	}
	if score.TurnPenalty != 0 {
		t.Fatalf("turn penalty = %d, want 0", score.TurnPenalty)
	}
	if score.StopPenalty != stopPenaltyCost {
		t.Fatalf("stop penalty = %d, want %d", score.StopPenalty, stopPenaltyCost)
	}
	if score.WaitCompensation != 5 {
		t.Fatalf("wait compensation = %d, want 5", score.WaitCompensation)
	}
	if score.Total != 11 {
		t.Fatalf("total cost = %d, want 11", score.Total)
	}
}

func TestEstimateAssignmentScoreAddsTurnPenalty(t *testing.T) {
	system := newCostTestSystem(t)
	elevator := Elevator{
		ID:           1,
		CurrentFloor: 10,
		Direction:    DirectionUp,
	}
	request := Request{
		ID:        1,
		Floor:     3,
		Direction: DirectionDown,
		Kind:      RequestKindHall,
		Status:    RequestPending,
	}

	score := EstimateAssignmentScore(system, elevator, request)

	if score.TurnPenalty != turnPenaltyCost {
		t.Fatalf("turn penalty = %d, want %d", score.TurnPenalty, turnPenaltyCost)
	}
}

func TestBestIdleAssignmentCandidateChoosesLowestCostElevator(t *testing.T) {
	system := newCostTestSystem(t)
	system.Elevators[0].CurrentFloor = 1
	system.Elevators[1].CurrentFloor = 8
	system.Elevators[2].CurrentFloor = 12

	request, err := system.AddRequest(9, DirectionUp, RequestKindHall)
	if err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
	}

	candidate, ok := BestIdleAssignmentCandidate(system, request.ID)
	if !ok {
		t.Fatal("BestIdleAssignmentCandidate returned false, want true")
	}

	if candidate.ElevatorIndex != 1 {
		t.Fatalf("elevator index = %d, want 1", candidate.ElevatorIndex)
	}
	if candidate.RequestID != request.ID {
		t.Fatalf("request ID = %d, want %d", candidate.RequestID, request.ID)
	}
}

func TestNearestIdleSchedulerUsesCostCandidate(t *testing.T) {
	system := newCostTestSystem(t)
	if err := system.SetScheduler("nearest-idle"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}

	system.Elevators[0].CurrentFloor = 1
	system.Elevators[1].CurrentFloor = 8
	system.Elevators[2].CurrentFloor = 12

	request, err := system.AddRequest(9, DirectionUp, RequestKindHall)
	if err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
	}

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	if request.AssignedElevatorID != 2 {
		t.Fatalf("assigned elevator ID = %d, want 2", request.AssignedElevatorID)
	}
}

func newCostTestSystem(t *testing.T) *System {
	t.Helper()

	system, err := NewSystem(SystemConfig{
		Floors:           20,
		ElevatorCount:    3,
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
