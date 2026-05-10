package elevator

const (
	// invalidAssignmentCost 表示某个候选分配不可用。
	// 这个值足够大，正常 cost 不应该达到它。
	invalidAssignmentCost = int(^uint(0) >> 1)

	// turnPenaltyCost 表示电梯为了处理请求需要改变当前运行方向的惩罚。
	turnPenaltyCost = 20

	// stopPenaltyCost 表示电梯已有一个停靠计划带来的额外惩罚。
	stopPenaltyCost = 10
)

// AssignmentScore 记录一次候选分配的评分明细。
// Total 越小，说明这个候选越适合执行。
type AssignmentScore struct {
	DistanceCost     int
	TurnPenalty      int
	StopPenalty      int
	WaitCompensation int
	Total            int
}

// AssignmentCandidate 表示“把某个请求分配给某部电梯”的一个候选方案。
type AssignmentCandidate struct {
	RequestID     int64
	ElevatorIndex int
	Score         AssignmentScore
}

// EstimateCost 返回某部电梯处理某个请求的总成本。
// 这个函数是给调度器使用的简单入口；如果需要看评分明细，可以调用 EstimateAssignmentScore。
func EstimateCost(system *System, elevator Elevator, request Request) int {
	return EstimateAssignmentScore(system, elevator, request).Total
}

// EstimateAssignmentScore 计算一次候选分配的评分明细。
// 当前公式还比较基础，但已经预留了距离、掉头、已有停靠和等待时间补偿四个维度。
func EstimateAssignmentScore(system *System, elevator Elevator, request Request) AssignmentScore {
	if elevator.EmergencyStop {
		return AssignmentScore{Total: invalidAssignmentCost}
	}

	distanceCost := floorDistance(elevator.CurrentFloor, request.Floor) * system.TicksPerFloor
	turnPenalty := estimateTurnPenalty(elevator, request)
	stopPenalty := len(elevator.Stops) * stopPenaltyCost
	waitCompensation := estimateWaitCompensation(system, request)

	total := distanceCost + turnPenalty + stopPenalty - waitCompensation
	if total < 0 {
		total = 0
	}

	return AssignmentScore{
		DistanceCost:     distanceCost,
		TurnPenalty:      turnPenalty,
		StopPenalty:      stopPenalty,
		WaitCompensation: waitCompensation,
		Total:            total,
	}
}

// BestIdleAssignmentCandidate 在所有空闲电梯中，为指定请求选择 cost 最低的候选。
func BestIdleAssignmentCandidate(system *System, requestID int64) (AssignmentCandidate, bool) {
	request, ok := system.Requests[requestID]
	if !ok || request.Status != RequestPending {
		return AssignmentCandidate{}, false
	}

	bestCandidate := AssignmentCandidate{}
	found := false

	for elevatorIndex, elevator := range system.Elevators {
		if !canAcceptRequest(elevator) {
			continue
		}

		score := EstimateAssignmentScore(system, elevator, *request)
		candidate := AssignmentCandidate{
			RequestID:     requestID,
			ElevatorIndex: elevatorIndex,
			Score:         score,
		}

		if !found || isBetterAssignmentCandidate(candidate, bestCandidate, system) {
			bestCandidate = candidate
			found = true
		}
	}

	return bestCandidate, found
}

// estimateTurnPenalty 判断电梯处理请求是否需要掉头。
func estimateTurnPenalty(elevator Elevator, request Request) int {
	if elevator.Direction == DirectionIdle {
		return 0
	}

	if isFloorInDirection(request.Floor, elevator.CurrentFloor, elevator.Direction) {
		return 0
	}

	return turnPenaltyCost
}

// estimateWaitCompensation 让等待较久的请求得到一定补偿，避免长期饥饿。
func estimateWaitCompensation(system *System, request Request) int {
	waitTicks := system.CurrentTick - request.CreatedTick
	if waitTicks < 0 {
		return 0
	}
	return waitTicks
}

// isBetterAssignmentCandidate 比较两个候选。
// cost 相同的时候，选择电梯编号更小的候选，让结果稳定、方便测试。
func isBetterAssignmentCandidate(candidate AssignmentCandidate, current AssignmentCandidate, system *System) bool {
	if candidate.Score.Total != current.Score.Total {
		return candidate.Score.Total < current.Score.Total
	}
	return system.Elevators[candidate.ElevatorIndex].ID < system.Elevators[current.ElevatorIndex].ID
}

// isFloorInDirection 判断目标楼层是否在指定方向上。
func isFloorInDirection(floor int, currentFloor int, direction Direction) bool {
	if direction == DirectionDown {
		return floor <= currentFloor
	}
	if direction == DirectionUp {
		return floor >= currentFloor
	}
	return true
}
