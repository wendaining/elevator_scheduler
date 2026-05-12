package elevator

// LOOKScheduler 实现 LOOK 电梯调度算法。
//
// LOOK 和 SCAN 的主要区别：
//   - SCAN 的思想是沿当前扫描方向一直走到物理边界，再反向。
//   - LOOK 不会为了到达物理边界而空跑；如果当前方向前方已经没有任务，
//     但反方向还有任务，就立即切换 ScanDirection。
//
// 当前项目中电梯只会按照 Stops 移动，不会在没有停靠点时继续空跑到顶层或底层。
// 因此 LOOK 的关键实现点是：每次分配前先根据现有 Stops 判断是否需要提前反向，
// 然后和 SCAN 一样，在候选电梯中使用 cost 函数选择最合适的分配。
type LOOKScheduler struct{}

func (LOOKScheduler) Name() string {
	return "look"
}

func (LOOKScheduler) Assign(s *System) bool {
	if !hasPendingRequests(s) || len(s.Elevators) == 0 {
		return false
	}

	candidate, ok := bestLOOKAssignmentCandidate(s)
	if !ok {
		return false
	}

	assignLOOKRequest(s, candidate.ElevatorIndex, candidate.RequestID)
	return true
}

func bestLOOKAssignmentCandidate(s *System) (AssignmentCandidate, bool) {
	bestCandidate := AssignmentCandidate{}
	found := false

	for _, requestID := range requestIDsByStatus(s, RequestPending) {
		request := s.Requests[requestID]

		for elevatorIndex := range s.Elevators {
			elevator := &s.Elevators[elevatorIndex]
			prepareLOOKDirection(elevator)
			if !canConsiderLOOKRequest(*elevator, *request) {
				continue
			}

			scoreElevator := elevatorForLOOKCost(*elevator)
			score := EstimateAssignmentScore(s, scoreElevator, *request)
			candidate := AssignmentCandidate{
				RequestID:     requestID,
				ElevatorIndex: elevatorIndex,
				Score:         score,
			}

			if !found || isBetterLOOKAssignmentCandidate(candidate, bestCandidate, s) {
				bestCandidate = candidate
				found = true
			}
		}
	}

	return bestCandidate, found
}

func prepareLOOKDirection(e *Elevator) {
	normalizeLOOKDirection(e)
	if len(e.Stops) == 0 {
		return
	}

	if hasLOOKStopAhead(*e, e.ScanDirection) {
		return
	}

	oppositeDirection := oppositeLOOKDirection(e.ScanDirection)
	if hasLOOKStopAhead(*e, oppositeDirection) {
		e.ScanDirection = oppositeDirection
	}
}

func hasLOOKStopAhead(e Elevator, direction Direction) bool {
	for _, stop := range e.Stops {
		if isFloorAheadInLOOK(e.CurrentFloor, stop.Floor, direction) {
			return true
		}
	}
	return false
}

func canConsiderLOOKRequest(e Elevator, request Request) bool {
	if e.EmergencyStop {
		return false
	}
	if len(e.Stops) == 0 {
		return true
	}
	return canAppendLOOKRequest(e, request)
}

func canAppendLOOKRequest(e Elevator, request Request) bool {
	return isFloorAheadInLOOK(e.CurrentFloor, request.Floor, e.ScanDirection) &&
		requestMatchesLOOKDirection(request, e.ScanDirection)
}

func requestMatchesLOOKDirection(request Request, direction Direction) bool {
	if request.Kind == RequestKindCabin {
		return true
	}
	return request.Direction == direction
}

func isFloorAheadInLOOK(currentFloor int, requestFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return requestFloor <= currentFloor
	}
	return requestFloor >= currentFloor
}

func elevatorForLOOKCost(e Elevator) Elevator {
	if e.Direction == DirectionIdle && len(e.Stops) > 0 {
		e.Direction = e.ScanDirection
	}
	return e
}

func isBetterLOOKAssignmentCandidate(candidate AssignmentCandidate, current AssignmentCandidate, s *System) bool {
	if candidate.Score.Total != current.Score.Total {
		return candidate.Score.Total < current.Score.Total
	}
	if candidate.RequestID != current.RequestID {
		return candidate.RequestID < current.RequestID
	}
	return s.Elevators[candidate.ElevatorIndex].ID < s.Elevators[current.ElevatorIndex].ID
}

func normalizeLOOKDirection(e *Elevator) {
	if e.ScanDirection != DirectionUp && e.ScanDirection != DirectionDown {
		e.ScanDirection = DirectionUp
	}
}

func oppositeLOOKDirection(direction Direction) Direction {
	if direction == DirectionDown {
		return DirectionUp
	}
	return DirectionDown
}

func alignLOOKDirectionToRequest(e *Elevator, request Request) {
	if request.Floor < e.CurrentFloor {
		e.ScanDirection = DirectionDown
		return
	}
	e.ScanDirection = DirectionUp
}

func assignLOOKRequest(s *System, elevatorIndex int, requestID int64) {
	request := s.Requests[requestID]
	elevator := &s.Elevators[elevatorIndex]
	if len(elevator.Stops) == 0 {
		alignLOOKDirectionToRequest(elevator, *request)
	} else {
		prepareLOOKDirection(elevator)
	}

	request.Status = RequestAssigned
	request.AssignedTick = s.CurrentTick
	request.AssignedElevatorID = elevator.ID

	insertLOOKStop(elevator, stopPlanFromRequest(*request))
	logRequestAssigned(s, *request, elevator.ID)
}

func insertLOOKStop(e *Elevator, stop StopPlan) {
	addStopPlan(e, stop)
	sortLOOKStops(e)
}

func sortLOOKStops(e *Elevator) {
	for i := 1; i < len(e.Stops); i++ {
		stop := e.Stops[i]
		j := i - 1

		for j >= 0 && shouldLOOKStopMoveBefore(stop.Floor, e.Stops[j].Floor, e.ScanDirection) {
			e.Stops[j+1] = e.Stops[j]
			j--
		}

		e.Stops[j+1] = stop
	}
}

func shouldLOOKStopMoveBefore(candidateFloor int, currentFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return candidateFloor > currentFloor
	}
	return candidateFloor < currentFloor
}
