package elevator

// SCANScheduler 实现 SCAN 调度算法。
//
// SCAN 也常被叫作"电梯算法"：
//   - 每部电梯都有一个长期扫描方向 ScanDirection。
//   - 调度时优先选择该方向上的请求。
//   - 如果该方向上没有请求，就反转 ScanDirection，再尝试反方向请求。
//
// 注意这里区分两个方向：
//   - Direction：电梯当前是否正在移动，用于展示和 stepElevator。
//   - ScanDirection：SCAN 算法的长期扫描方向，用于决定下一次接单时先看哪边。
//
// 当前实现支持顺路请求追加：
//   - 运行中的电梯如果能顺路服务某个请求，就把该请求转换为 StopPlan 插入 Stops。
//   - 空闲电梯没有顺路任务可追加时，才接收新的请求。
type SCANScheduler struct{}

func (SCANScheduler) Name() string {
	return "scan"
}

func (SCANScheduler) Assign(s *System) bool {
	if !hasPendingRequests(s) || len(s.Elevators) == 0 {
		return false
	}

	if assignAlongTheWay(s) {
		return true
	}

	return assignToIdleElevator(s)
}

func assignAlongTheWay(s *System) bool {
	for requestID, request := range s.Requests {
		if request.Status != RequestPending {
			continue
		}

		bestElevatorIndex := -1
		bestDistance := 0

		for elevatorIndex := range s.Elevators {
			elevator := &s.Elevators[elevatorIndex]
			normalizeScanDirection(elevator)
			if !canTakeRequestInSCAN(*elevator, *request) {
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

		elevator := &s.Elevators[bestElevatorIndex]
		assignSCANRequest(s, elevator, requestID)
		return true
	}

	return false
}

func assignToIdleElevator(s *System) bool {
	var bestRequestID int64
	bestElevatorIndex := -1
	bestPriority := 0
	bestDistance := 0

	for requestID, request := range s.Requests {
		if request.Status != RequestPending {
			continue
		}

		for elevatorIndex := range s.Elevators {
			elevator := &s.Elevators[elevatorIndex]
			if !canAcceptRequest(*elevator) {
				continue
			}

			normalizeScanDirection(elevator)

			priority := scanRequestPriority(*elevator, *request)
			distance := floorDistance(elevator.CurrentFloor, request.Floor)
			if bestElevatorIndex == -1 ||
				priority < bestPriority ||
				(priority == bestPriority && distance < bestDistance) {
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

	request := s.Requests[bestRequestID]
	elevator := &s.Elevators[bestElevatorIndex]
	if bestPriority > 0 {
		alignIdleElevatorScanDirection(elevator, request.Floor)
	}
	assignSCANRequest(s, elevator, bestRequestID)
	return true
}

// assignSCANRequest 将一个请求分配给指定电梯，更新请求状态并插入停靠计划。
func assignSCANRequest(s *System, elevator *Elevator, requestID int64) {
	request := s.Requests[requestID]
	request.Status = RequestAssigned
	request.AssignedTick = s.CurrentTick
	request.AssignedElevatorID = elevator.ID

	insertStopPlan(elevator, stopPlanFromRequest(*request))
}

func canTakeRequestInSCAN(e Elevator, request Request) bool {
	if e.EmergencyStop || len(e.Stops) == 0 {
		return false
	}

	return isFloorInScanDirection(request.Floor, e.CurrentFloor, e.ScanDirection) &&
		requestMatchesScanDirection(request, e.ScanDirection)
}

func scanRequestPriority(e Elevator, request Request) int {
	if isFloorInScanDirection(request.Floor, e.CurrentFloor, e.ScanDirection) &&
		requestMatchesScanDirection(request, e.ScanDirection) {
		return 0
	}

	return 1
}

func requestMatchesScanDirection(request Request, scanDirection Direction) bool {
	if request.Kind == RequestKindCabin {
		return true
	}

	return request.Direction == scanDirection
}

func normalizeScanDirection(e *Elevator) {
	if e.ScanDirection != DirectionUp && e.ScanDirection != DirectionDown {
		e.ScanDirection = DirectionUp
	}
}

func alignIdleElevatorScanDirection(e *Elevator, requestFloor int) {
	if requestFloor < e.CurrentFloor {
		e.ScanDirection = DirectionDown
		return
	}
	e.ScanDirection = DirectionUp
}

func insertStopPlan(e *Elevator, stop StopPlan) {
	addStopPlan(e, stop)
	sortStops(e)
}

func sortStops(e *Elevator) {
	for i := 1; i < len(e.Stops); i++ {
		stop := e.Stops[i]
		j := i - 1

		for j >= 0 && shouldMoveTargetBefore(stop.Floor, e.Stops[j].Floor, e.ScanDirection) {
			e.Stops[j+1] = e.Stops[j]
			j--
		}

		e.Stops[j+1] = stop
	}
}

func shouldMoveTargetBefore(candidate int, current int, direction Direction) bool {
	if direction == DirectionDown {
		return candidate > current
	}
	return candidate < current
}

func isFloorInScanDirection(floor int, currentFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return floor <= currentFloor
	}
	return floor >= currentFloor
}
