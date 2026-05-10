package elevator

// NearestIdleScheduler 是留给你继续完成的"最近空闲电梯优先"骨架。
//
// 目标行为：
//   - 取最早的 pending request。
//   - 在所有空闲且未紧急停止的电梯中，找到距离请求楼层最近的电梯。
//   - 把请求转换为 StopPlan 加入那部电梯。
type NearestIdleScheduler struct{}

func (NearestIdleScheduler) Name() string {
	return "nearest-idle"
}

func (NearestIdleScheduler) Assign(s *System) bool {
	requestID := firstPendingRequestID(s)
	if requestID == 0 || len(s.Elevators) == 0 {
		return false
	}

	candidate, ok := BestIdleAssignmentCandidate(s, requestID)
	if !ok {
		return false
	}

	s.assignRequestToElevator(candidate.RequestID, candidate.ElevatorIndex)
	return true
}
