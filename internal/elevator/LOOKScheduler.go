package elevator

import "sort"

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

	// LOOK 和 SCAN 一样，也是先找出成本最低的“请求-电梯”候选。
	candidate, ok := bestLOOKAssignmentCandidate(s)
	if !ok {
		return false
	}

	// 找到候选后再写入 Request 和 Elevator，避免选择阶段产生半成品状态。
	assignLOOKRequest(s, candidate.ElevatorIndex, candidate.RequestID)
	return true
}

// bestLOOKAssignmentCandidate 遍历所有 pending 请求和所有电梯，
// 找出最适合执行的“请求-电梯”组合。
//
// LOOK 的候选选择和 SCAN 很像；区别在于每次判断候选前，
// 会先用 prepareLOOKDirection 判断当前方向是否已经没有后续 stop，
// 如果需要，就提前把 ScanDirection 切到反方向。
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

			// 用电梯副本计算 cost，避免 cost 计算过程修改真实运行态。
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

// prepareLOOKDirection 是 LOOK 区别于 SCAN 的核心。
//
// 如果当前扫描方向前方还有停靠计划，就继续保持方向；
// 如果前方没有停靠计划，但反方向还有停靠计划，就立即反向。
// 这就是 LOOK“不空跑到物理边界”的含义。
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

// hasLOOKStopAhead 判断某个方向前方是否还有停靠计划。
//
// LOOK 用它决定是否需要提前反向。
func hasLOOKStopAhead(e Elevator, direction Direction) bool {
	for _, stop := range e.Stops {
		if isFloorAheadInLOOK(e.CurrentFloor, stop.Floor, direction) {
			return true
		}
	}
	return false
}

// canConsiderLOOKRequest 判断一部电梯是否可以成为某个请求的候选。
//
// 空闲电梯可以接单；已有 Stops 的电梯只有在请求顺路时才能接单。
func canConsiderLOOKRequest(e Elevator, request Request) bool {
	if e.EmergencyStop {
		return false
	}
	if len(e.Stops) == 0 {
		return true
	}
	return canAppendLOOKRequest(e, request)
}

// canAppendLOOKRequest 判断运行中电梯是否可以追加新请求。
//
// LOOK 已经通过 prepareLOOKDirection 更新过 ScanDirection，
// 因此这里直接按更新后的方向判断“是否顺路”。
func canAppendLOOKRequest(e Elevator, request Request) bool {
	return isFloorAheadInLOOK(e.CurrentFloor, request.Floor, e.ScanDirection) &&
		requestMatchesLOOKDirection(request, e.ScanDirection)
}

// requestMatchesLOOKDirection 判断请求方向是否匹配 LOOK 的扫描方向。
//
// cabin 请求没有楼层外的上/下行按钮语义，所以只要目标楼层顺路即可。
func requestMatchesLOOKDirection(request Request, direction Direction) bool {
	if request.Kind == RequestKindCabin {
		return true
	}
	return request.Direction == direction
}

// isFloorAheadInLOOK 判断 requestFloor 是否位于 currentFloor 的 direction 前方。
func isFloorAheadInLOOK(currentFloor int, requestFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return requestFloor <= currentFloor
	}
	return requestFloor >= currentFloor
}

// elevatorForLOOKCost 生成传给 cost 函数的电梯副本。
//
// 如果电梯已有 Stops，但当前 Direction 还是 idle，则说明它虽然暂时没移动，
// 但调度语义上应按 ScanDirection 继续执行，用 ScanDirection 计算 TurnPenalty 更合理。
func elevatorForLOOKCost(e Elevator) Elevator {
	if e.Direction == DirectionIdle && len(e.Stops) > 0 {
		e.Direction = e.ScanDirection
	}
	return e
}

// isBetterLOOKAssignmentCandidate 比较两个候选。
//
// 优先选择 cost 更低的候选；cost 相同则优先更早的请求；
// 请求相同则选择 ID 更小的电梯，保证结果稳定。
func isBetterLOOKAssignmentCandidate(candidate AssignmentCandidate, current AssignmentCandidate, s *System) bool {
	if candidate.Score.Total != current.Score.Total {
		return candidate.Score.Total < current.Score.Total
	}
	if candidate.RequestID != current.RequestID {
		return candidate.RequestID < current.RequestID
	}
	return s.Elevators[candidate.ElevatorIndex].ID < s.Elevators[current.ElevatorIndex].ID
}

// normalizeLOOKDirection 保证 ScanDirection 是一个有效的扫描方向。
func normalizeLOOKDirection(e *Elevator) {
	if e.ScanDirection != DirectionUp && e.ScanDirection != DirectionDown {
		e.ScanDirection = DirectionUp
	}
}

// oppositeLOOKDirection 返回当前扫描方向的反方向。
func oppositeLOOKDirection(direction Direction) Direction {
	if direction == DirectionDown {
		return DirectionUp
	}
	return DirectionDown
}

// alignLOOKDirectionToRequest 用于空闲电梯第一次接单。
//
// 空闲电梯没有现有 Stops，所以它可以直接根据请求楼层决定初始扫描方向。
func alignLOOKDirectionToRequest(e *Elevator, request Request) {
	if request.Floor < e.CurrentFloor {
		e.ScanDirection = DirectionDown
		return
	}
	e.ScanDirection = DirectionUp
}

// assignLOOKRequest 真正执行 LOOK 的分配动作。
//
// 选择阶段只计算候选；这里才会修改 Request 状态、记录分配时间，
// 并把请求转换成 StopPlan 插入电梯 Stops。
func assignLOOKRequest(s *System, elevatorIndex int, requestID int64) {
	request := s.Requests[requestID]
	elevator := &s.Elevators[elevatorIndex]
	if len(elevator.Stops) == 0 {
		alignLOOKDirectionToRequest(elevator, *request)
	} else {
		// 非空闲电梯分配前再次确认方向，避免插入 stop 时使用过期 ScanDirection。
		prepareLOOKDirection(elevator)
	}

	request.Status = RequestAssigned
	request.AssignedTick = s.CurrentTick
	request.AssignedElevatorID = elevator.ID

	insertLOOKStop(elevator, stopPlanFromRequest(*request))
	logRequestAssigned(s, *request, elevator.ID)
}

// insertLOOKStop 先合并或追加停靠计划，再按 LOOK 当前扫描方向排序。
func insertLOOKStop(e *Elevator, stop StopPlan) {
	addStopPlan(e, stop)
	sortLOOKStops(e)
}

// sortLOOKStops 按当前 ScanDirection 维护 Stops 顺序。
//
// 上行时低楼层排前面；下行时高楼层排前面。
// 这样 stepElevatorState 每次取 Stops[0] 就能得到下一个最近的顺路目标。
func sortLOOKStops(e *Elevator) {
	sort.SliceStable(e.Stops, func(i int, j int) bool {
		return shouldLOOKStopMoveBefore(e.Stops[i].Floor, e.Stops[j].Floor, e.ScanDirection)
	})
}

// shouldLOOKStopMoveBefore 判断 candidateFloor 是否应排在 currentFloor 前面。
func shouldLOOKStopMoveBefore(candidateFloor int, currentFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return candidateFloor > currentFloor
	}
	return candidateFloor < currentFloor
}
