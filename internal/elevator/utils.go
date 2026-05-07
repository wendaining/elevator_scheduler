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
	return !e.EmergencyStop && len(e.TargetFloors) == 0
}

func firstPendingRequestIndex(s *System) int {
	indices := requestIndicesByStatus(s, RequestPending)
	if len(indices) == 0 {
		return -1
	}
	return indices[0]
}

func hasPendingRequests(s *System) bool {
	return firstPendingRequestIndex(s) != -1
}

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

// 用于判断目标楼层数组中有无目标楼层
func containsFloor(floors []int, target int) bool {
	for _, floor := range floors {
		if floor == target {
			return true
		}
	}
	return false
}

// 去除 index 处的 request
func removeRequestAt(requests []Request, index int) []Request {
	// 注意 go 里面的 append 返回的也是 slice
	return append(requests[:index], requests[index+1:]...)
}
