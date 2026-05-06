package elevator

import (
	"encoding/json"
	"fmt"
)

// NewSystem 是 System 的构造函数，接收楼层数和电梯数量，返回初始化后的系统。
// 如果参数不合法（小于 1），返回错误而不是静默修正，让调用方知道传入了无效参数。
func NewSystem(floors int, elevatorCount int) (*System, error) {
	if floors < 1 {
		return nil, fmt.Errorf("floors must be at least 1, got %d", floors)
	}
	if elevatorCount < 1 {
		return nil, fmt.Errorf("elevator count must be at least 1, got %d", elevatorCount)
	}

	elevators := make([]Elevator, elevatorCount)
	for i := range elevators {
		elevators[i] = Elevator{
			ID:            i + 1,
			CurrentFloor:  1,
			Direction:     DirectionIdle,
			DoorOpen:      false,
			TargetFloors:  []int{},
			EmergencyStop: false,
		}
	}

	return &System{
		FloorCount:      floors,
		Elevators:       elevators,
		PendingRequests: []Request{},
	}, nil
}

// AddRequest 向系统添加一个新的乘梯请求。先校验参数是否合法，再将请求追加到
// PendingRequests 末尾。
func (s *System) AddRequest(floor int, direction Direction, kind RequestKind) error {
	if floor < 1 || floor > s.FloorCount {
		return fmt.Errorf("floor must be between 1 and %d, got %d", s.FloorCount, floor)
	}
	if direction != DirectionUp && direction != DirectionDown && direction != DirectionIdle {
		return fmt.Errorf("direction must be up, down, or idle, got %s", direction)
	}
	if kind != RequestKindHall && kind != RequestKindCabin {
		return fmt.Errorf("kind must be hall or cabin, got %s", kind)
	}
	s.PendingRequests = append(s.PendingRequests, Request{Floor: floor, Direction: direction, Kind: kind})
	return nil
}

// Snapshot 返回系统当前状态的 JSON 快照，带缩进格式，便于调试和 HTTP API 使用。
func (s *System) Snapshot() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// Step 推进系统一个离散时间片。每调用一次，先尝试将待处理请求分配给电梯，
// 然后让每部电梯向各自的目标楼层移动一层（或到达后开门）。
func (s *System) Step() error {
	if len(s.Elevators) == 0 {
		return fmt.Errorf("system has no elevators")
	}

	s.assignNextRequestToFirstElevator()

	for i := range s.Elevators {
		stepElevator(&s.Elevators[i])
	}

	return nil
}

// assignNextRequestToFirstElevator 是临时调度策略：取出最早的待处理请求，
// 分配给 1 号电梯。只有当 1 号电梯空闲（无目标楼层、未紧急停止）时才会分配。
func (s *System) assignNextRequestToFirstElevator() {
	if len(s.PendingRequests) == 0 {
		return
	}

	firstElevator := &s.Elevators[0]
	// 1 号电梯紧急停止或正在执行其他任务，暂不分配
	if firstElevator.EmergencyStop || len(firstElevator.TargetFloors) > 0 {
		return
	}

	// 取出最早的请求，分配给 1 号电梯
	request := s.PendingRequests[0]
	s.PendingRequests = s.PendingRequests[1:]
	firstElevator.TargetFloors = append(firstElevator.TargetFloors, request.Floor)
}

// stepElevator 推进单部电梯一个时间片：每次最多向目标楼层移动一层，
// 到达目标楼层后开门并移除该目标。如果电梯处于紧急停止状态，则保持不动。
func stepElevator(e *Elevator) {
	// 紧急停止状态下不移动
	if e.EmergencyStop {
		e.Direction = DirectionIdle
		return
	}

	// 如果门开着，先关门（下一次 Step 再移动）
	if e.DoorOpen {
		e.DoorOpen = false
	}

	// 没有目标楼层，保持空闲
	if len(e.TargetFloors) == 0 {
		e.Direction = DirectionIdle
		return
	}

	// 向第一个目标楼层移动一层
	targetFloor := e.TargetFloors[0]
	if e.CurrentFloor < targetFloor {
		e.Direction = DirectionUp
		e.CurrentFloor++
		return
	} else if e.CurrentFloor > targetFloor {
		e.Direction = DirectionDown
		e.CurrentFloor--
		return
	} else { // 已到达目标楼层：开门，移除该目标
		e.Direction = DirectionIdle
		e.DoorOpen = true
		e.TargetFloors = e.TargetFloors[1:]
	}
}
