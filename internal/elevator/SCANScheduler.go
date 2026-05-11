package elevator

// SCANScheduler 实现经典 SCAN 电梯调度算法。
//
// 核心规则：
//  1. 电梯维护长期扫描方向 ScanDirection。
//  2. 候选电梯包含空闲电梯和正在顺路运行的电梯。
//  3. 在候选集合内使用 cost 函数比较距离、已有停靠数量和等待补偿。
//  4. 上行时停靠计划按楼层升序排列；下行时按楼层降序排列。
type SCANScheduler struct{}

func (SCANScheduler) Name() string {
	return "scan"
}

func (SCANScheduler) Assign(s *System) bool {
	if !hasPendingRequests(s) || len(s.Elevators) == 0 {
		return false
	}

	candidate, ok := bestSCANAssignmentCandidate(s)
	if !ok {
		return false
	}

	assignSCANRequest(s, candidate.ElevatorIndex, candidate.RequestID)
	return true
}

func bestSCANAssignmentCandidate(s *System) (AssignmentCandidate, bool) {
	bestCandidate := AssignmentCandidate{}
	found := false

	for _, requestID := range requestIDsByStatus(s, RequestPending) {
		request := s.Requests[requestID]

		for elevatorIndex := range s.Elevators {
			elevator := &s.Elevators[elevatorIndex]
			normalizeSCANDirection(elevator)
			if !canConsiderSCANRequest(*elevator, *request) {
				continue
			}

			scoreElevator := elevatorForSCANCost(*elevator)
			score := EstimateAssignmentScore(s, scoreElevator, *request)
			candidate := AssignmentCandidate{
				RequestID:     requestID,
				ElevatorIndex: elevatorIndex,
				Score:         score,
			}

			if !found || isBetterSCANAssignmentCandidate(candidate, bestCandidate, s) {
				bestCandidate = candidate
				found = true
			}
		}
	}

	return bestCandidate, found
}

func canConsiderSCANRequest(e Elevator, request Request) bool {
	if e.EmergencyStop {
		return false
	}
	if len(e.Stops) == 0 {
		return true
	}
	return canAppendSCANRequest(e, request)
}

func canAppendSCANRequest(e Elevator, request Request) bool {
	return isFloorAheadInSCAN(e.CurrentFloor, request.Floor, e.ScanDirection) &&
		requestMatchesSCANDirection(request, e.ScanDirection)
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

func elevatorForSCANCost(e Elevator) Elevator {
	if e.Direction == DirectionIdle && len(e.Stops) > 0 {
		e.Direction = e.ScanDirection
	}
	return e
}

func isBetterSCANAssignmentCandidate(candidate AssignmentCandidate, current AssignmentCandidate, s *System) bool {
	if candidate.Score.Total != current.Score.Total {
		return candidate.Score.Total < current.Score.Total
	}
	if candidate.RequestID != current.RequestID {
		return candidate.RequestID < current.RequestID
	}
	return s.Elevators[candidate.ElevatorIndex].ID < s.Elevators[current.ElevatorIndex].ID
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
	if len(elevator.Stops) == 0 {
		alignSCANDirectionToRequest(elevator, *request)
	}

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
