package elevator

import "sort"

func IsValidDirection(direction Direction) bool {
	return direction == DirectionUp ||
		direction == DirectionDown ||
		direction == DirectionIdle
}

func IsValidRequestKind(kind RequestKind) bool {
	return kind == RequestKindHall ||
		kind == RequestKindCabin
}

func IsValidRequestStatus(status RequestStatus) bool {
	return status == RequestPending ||
		status == RequestAssigned ||
		status == RequestDone
}

func canAcceptRequest(e Elevator) bool {
	return !e.EmergencyStop && len(e.Stops) == 0
}

// firstPendingRequestID 找到所有 pending 请求中 ID 最小的一个，返回其请求 ID。
// 如果没有 pending 请求，返回 0。
func firstPendingRequestID(s *System) int64 {
	ids := requestIDsByStatus(s, RequestPending)
	if len(ids) == 0 {
		return 0
	}
	return ids[0]
}

// hasPendingRequests 判断系统中是否有待分配的请求。
func hasPendingRequests(s *System) bool {
	return firstPendingRequestID(s) != 0
}

// requestIDsByStatus 返回系统中所有请求状态为 status 的请求 ID 列表。
func requestIDsByStatus(s *System, status RequestStatus) []int64 {
	ids := []int64{}
	for id, request := range s.Requests {
		if request.Status == status {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func floorDistance(a int, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

// stopPlanFromRequest 将一个 Request 转换成对应的 StopPlan，方便调度器处理。
// 对于 cabin 请求，reason 固定为 cabin；
// 对于 hall 请求，根据 direction 区分 reason 是 hall_up 还是 hall_down。
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

// addStopPlan 将一个停靠计划添加到电梯的停靠计划列表中。
// 如果已经有一个相同的停靠计划（同一层、同一原因、同一方向），
// 则将请求 ID 合并到已有的停靠计划中，而不是添加一个新的停靠计划。
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

// isSameStop 判断两个 StopPlan 是否表示同一停靠：
// 同一层、同一原因、同一方向。
func isSameStop(a StopPlan, b StopPlan) bool {
	return a.Floor == b.Floor &&
		a.Reason == b.Reason &&
		a.Direction == b.Direction
}
