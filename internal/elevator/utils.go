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

func floorDistance(a int, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func containsFloor(floors []int, target int) bool {
	for _, floor := range floors {
		if floor == target {
			return true
		}
	}
	return false
}

func removeRequestAt(requests []Request, index int) []Request {
	return append(requests[:index], requests[index+1:]...)
}
