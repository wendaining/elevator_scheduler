package elevator

func isValidDirection(direction Direction) bool {
	return direction == DirectionUp ||
		direction == DirectionDown ||
		direction == DirectionIdle
}

func isValidRequestKind(kind RequestKind) bool {
	return kind == RequestKindHall ||
		kind == RequestKindCabin
}
