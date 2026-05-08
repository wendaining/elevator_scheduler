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
	system.Requests = map[int64]*Request{
		1: {ID: 1, Floor: 3, Direction: DirectionDown, Kind: RequestKindHall, Status: RequestPending},
		2: {ID: 2, Floor: 8, Direction: DirectionUp, Kind: RequestKindHall, Status: RequestPending},
	}
	system.nextRequestID = 3

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	stops := system.Elevators[0].Stops
	if len(stops) != 1 || stops[0].Floor != 8 {
		t.Fatalf("stops = %v, want floor 8", stops)
	}
	if system.Elevators[0].ScanDirection != DirectionUp {
		t.Fatalf("scan direction = %q, want %q", system.Elevators[0].ScanDirection, DirectionUp)
	}
	if system.Requests[2].Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q", system.Requests[2].Status, RequestAssigned)
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
	system.Requests = map[int64]*Request{
		1: {ID: 1, Floor: 4, Direction: DirectionDown, Kind: RequestKindHall, Status: RequestPending},
	}
	system.nextRequestID = 2

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	stops := system.Elevators[0].Stops
	if len(stops) != 1 || stops[0].Floor != 4 {
		t.Fatalf("stops = %v, want floor 4", stops)
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
	system.Elevators[0].Stops = []StopPlan{
		{Floor: 10, Reason: StopReasonHallUp, Direction: DirectionUp, RequestIDs: []int64{99}},
	}
	system.Requests = map[int64]*Request{
		1: {ID: 1, Floor: 6, Direction: DirectionUp, Kind: RequestKindHall, Status: RequestPending},
	}
	system.nextRequestID = 2

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	stops := system.Elevators[0].Stops
	if len(stops) != 2 || stops[0].Floor != 6 || stops[1].Floor != 10 {
		t.Fatalf("stops = %v, want floors [6 10]", stops)
	}
	if system.Requests[1].Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q", system.Requests[1].Status, RequestAssigned)
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
	system.Elevators[0].Stops = []StopPlan{
		{Floor: 2, Reason: StopReasonHallDown, Direction: DirectionDown, RequestIDs: []int64{99}},
	}
	system.Requests = map[int64]*Request{
		1: {ID: 1, Floor: 8, Direction: DirectionDown, Kind: RequestKindHall, Status: RequestPending},
	}
	system.nextRequestID = 2

	if !system.scheduler.Assign(system) {
		t.Fatal("Assign returned false, want true")
	}

	stops := system.Elevators[0].Stops
	if len(stops) != 2 || stops[0].Floor != 8 || stops[1].Floor != 2 {
		t.Fatalf("stops = %v, want floors [8 2]", stops)
	}
	if system.Requests[1].Status != RequestAssigned {
		t.Fatalf("request status = %q, want %q", system.Requests[1].Status, RequestAssigned)
	}
}
