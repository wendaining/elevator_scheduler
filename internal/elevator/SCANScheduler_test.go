package elevator

import "testing"

func TestSCANSchedulerAssignsRequestInScanDirection(t *testing.T) {
	system, err := NewSystem(20, 1, 5, 2, 1)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}
	if err := system.SetScheduler("scan"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}

	system.Elevators[0].CurrentFloor = 5
	system.Elevators[0].ScanDirection = DirectionUp
	system.Requests = []Request{
		{ID: 1, Floor: 3, Direction: DirectionDown, Kind: RequestKindHall, Status: RequestPending},
		{ID: 2, Floor: 8, Direction: DirectionUp, Kind: RequestKindHall, Status: RequestPending},
	}

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	targets := system.Elevators[0].TargetFloors
	if len(targets) != 1 || targets[0] != 8 {
		t.Fatalf("targets = %v, want [8]", targets)
	}
	if system.Elevators[0].ScanDirection != DirectionUp {
		t.Fatalf("scan direction = %q, want %q", system.Elevators[0].ScanDirection, DirectionUp)
	}
	if system.Requests[1].Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q", system.Requests[1].Status, RequestAssigned)
	}
}

func TestSCANSchedulerReversesWhenNoRequestInScanDirection(t *testing.T) {
	system, err := NewSystem(20, 1, 5, 2, 1)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}
	if err := system.SetScheduler("scan"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}

	system.Elevators[0].CurrentFloor = 10
	system.Elevators[0].ScanDirection = DirectionUp
	system.Requests = []Request{
		{ID: 1, Floor: 4, Direction: DirectionDown, Kind: RequestKindHall, Status: RequestPending},
	}

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	targets := system.Elevators[0].TargetFloors
	if len(targets) != 1 || targets[0] != 4 {
		t.Fatalf("targets = %v, want [4]", targets)
	}
	if system.Elevators[0].ScanDirection != DirectionDown {
		t.Fatalf("scan direction = %q, want %q", system.Elevators[0].ScanDirection, DirectionDown)
	}
}

func TestSCANSchedulerAppendsUpRequestAlongTheWay(t *testing.T) {
	system, err := NewSystem(20, 1, 5, 2, 1)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}
	if err := system.SetScheduler("scan"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}

	system.Elevators[0].CurrentFloor = 3
	system.Elevators[0].ScanDirection = DirectionUp
	system.Elevators[0].TargetFloors = []int{10}
	system.Elevators[0].TargetRequestIDs = []int64{99}
	system.Requests = []Request{
		{ID: 1, Floor: 6, Direction: DirectionUp, Kind: RequestKindHall, Status: RequestPending},
	}

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	targets := system.Elevators[0].TargetFloors
	if len(targets) != 2 || targets[0] != 6 || targets[1] != 10 {
		t.Fatalf("targets = %v, want [6 10]", targets)
	}
	if system.Requests[0].Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q", system.Requests[0].Status, RequestAssigned)
	}
}

func TestSCANSchedulerAppendsDownRequestAlongTheWay(t *testing.T) {
	system, err := NewSystem(20, 1, 5, 2, 1)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}
	if err := system.SetScheduler("scan"); err != nil {
		t.Fatalf("SetScheduler returned error: %v", err)
	}

	system.Elevators[0].CurrentFloor = 12
	system.Elevators[0].ScanDirection = DirectionDown
	system.Elevators[0].TargetFloors = []int{2}
	system.Elevators[0].TargetRequestIDs = []int64{99}
	system.Requests = []Request{
		{ID: 1, Floor: 8, Direction: DirectionDown, Kind: RequestKindHall, Status: RequestPending},
	}

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	targets := system.Elevators[0].TargetFloors
	if len(targets) != 2 || targets[0] != 8 || targets[1] != 2 {
		t.Fatalf("targets = %v, want [8 2]", targets)
	}
	if system.Requests[0].Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q", system.Requests[0].Status, RequestAssigned)
	}
}
