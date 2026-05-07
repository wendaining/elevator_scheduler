package elevator

// FCFS - First Come First Served
// 当前策略：直接按照请求的顺序去处理
type FCFSScheduler struct{}

func (FCFSScheduler) Name() string {
	return "fcfs"
}

func (FCFSScheduler) Assign(s *System) bool {
	if len(s.PendingRequests) == 0 || len(s.Elevators) == 0 {
		return false
	}
	// 取出系统中 pending requests 里面第一个请求
	request := s.PendingRequests[0]
	for _, elevator := range s.Elevators {
		if !canAcceptRequest(elevator) {
			continue
		}
		s.PendingRequests = s.PendingRequests[1:]
		elevator.TargetFloors = append(elevator.TargetFloors, request.Floor)
		return true
	}
	return false
}
