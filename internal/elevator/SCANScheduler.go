package elevator

import "sort"

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

	// 先在“请求 × 电梯”的所有可行组合里找出成本最低的候选。
	candidate, ok := bestSCANAssignmentCandidate(s)
	if !ok {
		return false
	}

	// 找到候选后，才真正修改 Request 和 Elevator 的运行态数据。
	assignSCANRequest(s, candidate.ElevatorIndex, candidate.RequestID)
	return true
}

// bestSCANAssignmentCandidate 遍历所有 pending 请求和所有电梯，
// 找到一个最适合执行的“请求-电梯”组合。
//
// 注意：SCAN 不是只看空闲电梯。运行中的电梯如果顺路，也可以继续追加 stop。
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

			// cost 函数只读取 Elevator，不会修改原电梯。
			// 这里用一个副本做必要修正，避免 DirectionIdle 干扰掉头惩罚判断。
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

// canConsiderSCANRequest 判断一部电梯是否可以成为某个请求的候选。
//
// 空闲电梯总是可以考虑；已有停靠计划的电梯只有在“顺路”时才可以考虑。
func canConsiderSCANRequest(e Elevator, request Request) bool {
	if e.EmergencyStop {
		return false
	}
	if len(e.Stops) == 0 {
		return true
	}
	return canAppendSCANRequest(e, request)
}

// canAppendSCANRequest 判断运行中的电梯是否可以把请求追加到现有 Stops 中。
//
// 必须同时满足：
//  1. 请求楼层位于电梯当前扫描方向的前方；
//  2. hall 请求方向和扫描方向一致。cabin 请求没有上下行按钮语义，只看目标楼层。
func canAppendSCANRequest(e Elevator, request Request) bool {
	return isFloorAheadInSCAN(e.CurrentFloor, request.Floor, e.ScanDirection) &&
		requestMatchesSCANDirection(request, e.ScanDirection)
}

// requestMatchesSCANDirection 判断请求方向是否符合扫描方向。
//
// cabin 请求表示“电梯内目标楼层”，不携带楼层外的上/下行等待语义，
// 所以只要目标楼层顺路，就允许追加。
func requestMatchesSCANDirection(request Request, direction Direction) bool {
	if request.Kind == RequestKindCabin {
		return true
	}
	return request.Direction == direction
}

// isFloorAheadInSCAN 判断 requestFloor 是否位于 currentFloor 的扫描前方。
func isFloorAheadInSCAN(currentFloor int, requestFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return requestFloor <= currentFloor
	}
	return requestFloor >= currentFloor
}

// elevatorForSCANCost 生成传给 cost 函数的电梯副本。
//
// 有些测试或边界状态中，电梯已有 Stops，但 Direction 仍是 idle。
// 对调度决策来说，这类电梯实际上应按 ScanDirection 理解，否则 TurnPenalty 会失真。
func elevatorForSCANCost(e Elevator) Elevator {
	if e.Direction == DirectionIdle && len(e.Stops) > 0 {
		e.Direction = e.ScanDirection
	}
	return e
}

// isBetterSCANAssignmentCandidate 比较两个候选。
//
// 主排序键是 cost 总分；分数相同则先处理更早创建的请求；
// 请求也相同则选择编号更小的电梯，让结果稳定、方便测试。
func isBetterSCANAssignmentCandidate(candidate AssignmentCandidate, current AssignmentCandidate, s *System) bool {
	if candidate.Score.Total != current.Score.Total {
		return candidate.Score.Total < current.Score.Total
	}
	if candidate.RequestID != current.RequestID {
		return candidate.RequestID < current.RequestID
	}
	return s.Elevators[candidate.ElevatorIndex].ID < s.Elevators[current.ElevatorIndex].ID
}

// normalizeSCANDirection 保证 ScanDirection 始终是 up 或 down。
//
// 空闲电梯初始可能没有明确扫描方向，这里统一默认为向上。
func normalizeSCANDirection(e *Elevator) {
	if e.ScanDirection != DirectionUp && e.ScanDirection != DirectionDown {
		e.ScanDirection = DirectionUp
	}
}

// alignSCANDirectionToRequest 用于空闲电梯第一次接单。
//
// 空闲电梯没有现有 Stops，因此它的扫描方向可以直接对齐到新请求所在楼层。
func alignSCANDirectionToRequest(e *Elevator, request Request) {
	if request.Floor < e.CurrentFloor {
		e.ScanDirection = DirectionDown
		return
	}
	e.ScanDirection = DirectionUp
}

// assignSCANRequest 真正执行分配动作。
//
// 这个函数会：
//  1. 修改 Request 的状态和分配时间；
//  2. 记录被分配的电梯 ID；
//  3. 把请求转换成 StopPlan 插入电梯 Stops。
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

// insertSCANStop 先合并或追加停靠计划，再按扫描方向重新排序。
func insertSCANStop(e *Elevator, stop StopPlan) {
	addStopPlan(e, stop)
	sortSCANStops(e)
}

// sortSCANStops 排序维护 Stops 顺序。
//
// sort.SliceStable 会按照 less 函数对切片排序，并且在比较结果相同时保留原相对顺序。
// 上行时低楼层排前面；下行时高楼层排前面。
func sortSCANStops(e *Elevator) {
	sort.SliceStable(e.Stops, func(i int, j int) bool {
		return shouldSCANStopMoveBefore(e.Stops[i].Floor, e.Stops[j].Floor, e.ScanDirection)
	})
}

// shouldSCANStopMoveBefore 判断 candidateFloor 是否应该排在 currentFloor 前面。
func shouldSCANStopMoveBefore(candidateFloor int, currentFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return candidateFloor > currentFloor
	}
	return candidateFloor < currentFloor
}
