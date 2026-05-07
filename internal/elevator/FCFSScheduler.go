package elevator

// FCFS - First Come First Served
// 当前策略：直接按照请求的顺序去处理
type FCFSScheduler struct{}

func (FCFSScheduler) Name() string {
	return "fcfs"
}

func (FCFSScheduler) Assign(s *System) bool {
	requestIndex := firstPendingRequestIndex(s)
	if requestIndex == -1 || len(s.Elevators) == 0 {
		return false
	}
	// 取出系统中 pending requests 里面第一个请求
	for i, elevator := range s.Elevators {
		if !canAcceptRequest(elevator) {
			continue
		}
		s.assignRequestToElevator(requestIndex, i)
		return true
	}
	return false
}
