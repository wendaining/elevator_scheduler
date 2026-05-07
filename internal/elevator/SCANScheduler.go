package elevator

// SCANScheduler 实现 SCAN 调度算法。
//
// SCAN 也常被叫作“电梯算法”：
//   - 每部电梯都有一个长期扫描方向 ScanDirection。
//   - 调度时优先选择该方向上的请求。
//   - 如果该方向上没有请求，就反转 ScanDirection，再尝试反方向请求。
//
// 注意这里区分两个方向：
//   - Direction：电梯当前是否正在移动，用于展示和 stepElevator。
//   - ScanDirection：SCAN 算法的长期扫描方向，用于决定下一次接单时先看哪边。
//
// 当前实现支持顺路请求追加：
//   - 运行中的电梯如果能顺路服务某个请求，就把该楼层插入 TargetFloors。
//   - 空闲电梯没有顺路任务可追加时，才接收新的请求。
type SCANScheduler struct{}

func (SCANScheduler) Name() string {
	return "scan"
}

func (SCANScheduler) Assign(s *System) bool {
	if len(s.PendingRequests) == 0 || len(s.Elevators) == 0 {
		return false
	}

	if assignAlongTheWay(s) {
		return true
	}

	return assignToIdleElevator(s)
}

func assignAlongTheWay(s *System) bool {
	for requestIndex, request := range s.PendingRequests {
		bestElevatorIndex := -1
		bestDistance := 0

		for elevatorIndex := range s.Elevators {
			elevator := &s.Elevators[elevatorIndex]
			normalizeScanDirection(elevator)
			if !canTakeRequestInSCAN(*elevator, request) {
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
		insertTargetFloor(elevator, request.Floor)
		s.PendingRequests = removeRequestAt(s.PendingRequests, requestIndex)
		return true
	}

	return false
}

func assignToIdleElevator(s *System) bool {
	bestRequestIndex := -1
	bestElevatorIndex := -1
	bestPriority := 0
	bestDistance := 0

	for requestIndex, request := range s.PendingRequests {
		for elevatorIndex := range s.Elevators {
			elevator := &s.Elevators[elevatorIndex]
			if !canAcceptRequest(*elevator) {
				continue
			}

			normalizeScanDirection(elevator)

			priority := scanRequestPriority(*elevator, request)
			distance := floorDistance(elevator.CurrentFloor, request.Floor)
			if bestElevatorIndex == -1 ||
				priority < bestPriority ||
				(priority == bestPriority && distance < bestDistance) {
				bestRequestIndex = requestIndex
				bestElevatorIndex = elevatorIndex
				bestPriority = priority
				bestDistance = distance
			}
		}
	}

	if bestElevatorIndex == -1 {
		return false
	}

	request := s.PendingRequests[bestRequestIndex]
	elevator := &s.Elevators[bestElevatorIndex]
	if bestPriority > 0 {
		alignIdleElevatorScanDirection(elevator, request.Floor)
	}
	insertTargetFloor(elevator, request.Floor)
	s.PendingRequests = removeRequestAt(s.PendingRequests, bestRequestIndex)
	return true
}

func canTakeRequestInSCAN(e Elevator, request Request) bool {
	if e.EmergencyStop || len(e.TargetFloors) == 0 {
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

func insertTargetFloor(e *Elevator, floor int) {
	if containsFloor(e.TargetFloors, floor) {
		return
	}

	e.TargetFloors = append(e.TargetFloors, floor)
	sortTargetFloors(e)
}

func sortTargetFloors(e *Elevator) {
	for i := 1; i < len(e.TargetFloors); i++ {
		value := e.TargetFloors[i]
		j := i - 1

		for j >= 0 && shouldMoveTargetBefore(value, e.TargetFloors[j], e.ScanDirection) {
			e.TargetFloors[j+1] = e.TargetFloors[j]
			j--
		}

		e.TargetFloors[j+1] = value
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
