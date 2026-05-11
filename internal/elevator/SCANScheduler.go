package elevator

// SCANScheduler 实现经典 SCAN 电梯调度算法。
//
// 核心规则：
//  1. 电梯维护长期扫描方向 ScanDirection。
//  2. 运行中的电梯可以追加顺路请求，而不是只能等空闲后再接单。
//  3. 上行时停靠计划按楼层升序排列；下行时按楼层降序排列。
//  4. 没有顺路电梯时，把请求分配给最近的空闲电梯。
type SCANScheduler struct{}

func (SCANScheduler) Name() string {
	return "scan"
}

func (SCANScheduler) Assign(s *System) bool {
	if !hasPendingRequests(s) || len(s.Elevators) == 0 {
		return false
	}

	if assignSCANAlongTheWay(s) {
		return true
	}

	return assignSCANToIdleElevator(s)
}

func assignSCANAlongTheWay(s *System) bool {
	for _, requestID := range requestIDsByStatus(s, RequestPending) {
		request := s.Requests[requestID]
		bestElevatorIndex := -1
		bestDistance := 0

		for elevatorIndex := range s.Elevators {
			elevator := &s.Elevators[elevatorIndex]
			normalizeSCANDirection(elevator)
			if !canAppendSCANRequest(*elevator, *request) {
				continue
			}

			distance := floorDistance(elevator.CurrentFloor, request.Floor)
			if bestElevatorIndex == -1 || distance < bestDistance {
				bestElevatorIndex = elevatorIndex
				bestDistance = distance
			}
		}

		if bestElevatorIndex == -1 {
			continue
		}

		assignSCANRequest(s, bestElevatorIndex, requestID)
		return true
	}

	return false
}

func assignSCANToIdleElevator(s *System) bool {
	bestRequestID := int64(0)
	bestElevatorIndex := -1
	bestPriority := 0
	bestDistance := 0

	for _, requestID := range requestIDsByStatus(s, RequestPending) {
		request := s.Requests[requestID]

		for elevatorIndex := range s.Elevators {
			elevator := &s.Elevators[elevatorIndex]
			if !canAcceptRequest(*elevator) {
				continue
			}

			normalizeSCANDirection(elevator)
			priority := idleSCANPriority(*elevator, *request)
			distance := floorDistance(elevator.CurrentFloor, request.Floor)
			if bestElevatorIndex == -1 ||
				priority < bestPriority ||
				(priority == bestPriority && distance < bestDistance) ||
				(priority == bestPriority && distance == bestDistance && requestID < bestRequestID) {
				bestRequestID = requestID
				bestElevatorIndex = elevatorIndex
				bestPriority = priority
				bestDistance = distance
			}
		}
	}

	if bestElevatorIndex == -1 {
		return false
	}

	elevator := &s.Elevators[bestElevatorIndex]
	request := s.Requests[bestRequestID]
	if bestPriority > 0 {
		alignSCANDirectionToRequest(elevator, *request)
	}
	assignSCANRequest(s, bestElevatorIndex, bestRequestID)
	return true
}

func canAppendSCANRequest(e Elevator, request Request) bool {
	if e.EmergencyStop || len(e.Stops) == 0 {
		return false
	}

	if !isFloorAheadInSCAN(e.CurrentFloor, request.Floor, e.ScanDirection) {
		return false
	}

	return requestMatchesSCANDirection(request, e.ScanDirection)
}

func idleSCANPriority(e Elevator, request Request) int {
	if isFloorAheadInSCAN(e.CurrentFloor, request.Floor, e.ScanDirection) &&
		requestMatchesSCANDirection(request, e.ScanDirection) {
		return 0
	}
	return 1
}

func requestMatchesSCANDirection(request Request, direction Direction) bool {
	if request.Kind == RequestKindCabin {
		return true
	}
	return request.Direction == direction
}

func isFloorAheadInSCAN(currentFloor int, requestFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return requestFloor <= currentFloor
	}
	return requestFloor >= currentFloor
}

func normalizeSCANDirection(e *Elevator) {
	if e.ScanDirection != DirectionUp && e.ScanDirection != DirectionDown {
		e.ScanDirection = DirectionUp
	}
}

func alignSCANDirectionToRequest(e *Elevator, request Request) {
	if request.Floor < e.CurrentFloor {
		e.ScanDirection = DirectionDown
		return
	}
	e.ScanDirection = DirectionUp
}

func assignSCANRequest(s *System, elevatorIndex int, requestID int64) {
	request := s.Requests[requestID]
	elevator := &s.Elevators[elevatorIndex]

	request.Status = RequestAssigned
	request.AssignedTick = s.CurrentTick
	request.AssignedElevatorID = elevator.ID

	insertSCANStop(elevator, stopPlanFromRequest(*request))
	logRequestAssigned(s, *request, elevator.ID)
}

func insertSCANStop(e *Elevator, stop StopPlan) {
	addStopPlan(e, stop)
	sortSCANStops(e)
}

func sortSCANStops(e *Elevator) {
	for i := 1; i < len(e.Stops); i++ {
		stop := e.Stops[i]
		j := i - 1

		for j >= 0 && shouldSCANStopMoveBefore(stop.Floor, e.Stops[j].Floor, e.ScanDirection) {
			e.Stops[j+1] = e.Stops[j]
			j--
		}

		e.Stops[j+1] = stop
	}
}

func shouldSCANStopMoveBefore(candidateFloor int, currentFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return candidateFloor > currentFloor
	}
	return candidateFloor < currentFloor
}
