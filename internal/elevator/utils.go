package elevator

func IsValidDirection(direction Direction) bool {
	return direction == DirectionUp ||
		direction == DirectionDown ||
		direction == DirectionIdle
}

func IsValidRequestKind(kind RequestKind) bool {
	return kind == RequestKindHall ||
		kind == RequestKindCabin
}

func canAcceptRequest(e Elevator) bool {
	return !e.EmergencyStop && len(e.Stops) == 0
}

// 找到所有 pending 请求中最早的一个，返回它在 s.Requests 中的下标。如果没有 pending 请求，返回 -1。
func firstPendingRequestIndex(s *System) int {
	indices := requestIndicesByStatus(s, RequestPending)
	if len(indices) == 0 {
		return -1
	}
	return indices[0]
}

// 判断系统中是否有待分配的请求
func hasPendingRequests(s *System) bool {
	return firstPendingRequestIndex(s) != -1
}

// 返回系统中所有请求状态为 status 的请求在 s.Requests 中的下标列表。
func requestIndicesByStatus(s *System, status RequestStatus) []int {
	indices := []int{}
	for i, request := range s.Requests {
		if request.Status == status {
			indices = append(indices, i)
		}
	}
	return indices
}

func floorDistance(a int, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func stopPlanFromRequest(request Request) StopPlan {
	reason := StopReasonCabin
	if request.Kind == RequestKindHall && request.Direction == DirectionUp {
		reason = StopReasonHallUp
	}
	if request.Kind == RequestKindHall && request.Direction == DirectionDown {
		reason = StopReasonHallDown
	}

	return StopPlan{
		Floor:      request.Floor,
		Reason:     reason,
		Direction:  request.Direction,
		RequestIDs: []int64{request.ID},
	}
}

func addStopPlan(e *Elevator, stop StopPlan) {
	for i := range e.Stops {
		if !isSameStop(e.Stops[i], stop) {
			continue
		}
		e.Stops[i].RequestIDs = append(e.Stops[i].RequestIDs, stop.RequestIDs...)
		return
	}
	e.Stops = append(e.Stops, stop)
}

func isSameStop(a StopPlan, b StopPlan) bool {
	return a.Floor == b.Floor &&
		a.Reason == b.Reason &&
		a.Direction == b.Direction
}
