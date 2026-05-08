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

	request := s.Requests[requestID] // 取最早的 pending request
	bestIndex := -1                  // 目标电梯序号
	bestDistance := 0                // 距离的最近值

	for i, elevator := range s.Elevators { // 下标和对象同时遍历
		if !canAcceptRequest(elevator) {
			continue
		}

		distance := floorDistance(elevator.CurrentFloor, request.Floor)

		if bestIndex == -1 || distance < bestDistance {
			bestIndex = i
			bestDistance = distance
		}
		if distance == bestDistance {
			numTargetNow := len(s.Elevators[bestIndex].Stops)
			numTargetCandidate := len(s.Elevators[i].Stops)
			if numTargetNow >= numTargetCandidate {
				if numTargetNow == numTargetCandidate {
					bestIndex = min(i, bestIndex) // 目标数量相同时选择编号小的
				} else {
					bestIndex = i // 否则选择目标更少的
				}
			}
		}
	}

	if bestIndex == -1 {
		return false
	}

	s.assignRequestToElevator(requestID, bestIndex)
	return true
}
