package elevator

import "fmt"

// NearestIdleScheduler 是留给你继续完成的“最近空闲电梯优先”骨架。
//
// 目标行为：
//   - 取最早的 PendingRequest。
//   - 在所有空闲且未紧急停止的电梯中，找到距离请求楼层最近的电梯。
//   - 把请求楼层加入那部电梯的 TargetFloors。
type NearestIdleScheduler struct{}

func (NearestIdleScheduler) Name() string {
	return "nearest-idle"
}

func (NearestIdleScheduler) Assign(system *System) bool {
	if len(system.PendingRequests) == 0 || len(system.Elevators) == 0 {
		return false
	}

	request := system.PendingRequests[0] // 取最早的 request
	bestIndex := -1                      // 目标电梯序号
	bestDistance := 0                    // 距离的最近值

	for i, elevator := range system.Elevators { // 下标和对象同时遍历
		if !canAcceptRequest(elevator) {
			continue
		}

		distance := floorDistance(elevator.CurrentFloor, request.Floor)

		if bestIndex == -1 || distance < bestDistance {
			bestIndex = i
			bestDistance = distance
		}
		if distance == bestDistance {
			numTargetNow := len(system.Elevators[bestIndex].TargetFloors)
			numTargetCandidate := len(system.Elevators[i].TargetFloors)
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

	system.PendingRequests = system.PendingRequests[1:]
	system.Elevators[bestIndex].TargetFloors = append(
		system.Elevators[bestIndex].TargetFloors,
		request.Floor,
	)
	return true
}

func NewScheduler(name string) (Scheduler, error) {
	switch name {
	case "first-available":
		return FirstAvailableScheduler{}, nil
	case "nearest-idle":
		return NearestIdleScheduler{}, nil
	default:
		return nil, fmt.Errorf("unknown scheduler %q", name)
	}
}
