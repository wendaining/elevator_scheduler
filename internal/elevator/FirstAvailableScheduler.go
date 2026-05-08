package elevator

// FirstAvailableScheduler 是最小可运行调度器。
// 当前策略非常简单：只看 1 号电梯。如果 1 号电梯空闲，就把最早的请求分配给它。
// 这个算法保留了之前写在 system.go 里的临时行为，便于先完成"迁移调度逻辑"。
type FirstAvailableScheduler struct{}

func (FirstAvailableScheduler) Name() string {
	return "first-available"
}

func (FirstAvailableScheduler) Assign(system *System) bool {
	requestID := firstPendingRequestID(system)
	if requestID == 0 || len(system.Elevators) == 0 {
		return false
	}

	firstElevator := &system.Elevators[0]
	if !canAcceptRequest(*firstElevator) {
		return false
	}

	system.assignRequestToElevator(requestID, 0)
	return true
}
