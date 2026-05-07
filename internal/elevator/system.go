package elevator

import (
	"encoding/json"
	"fmt"
	"log"
)

// NewSystem 是 System 的构造函数，接收楼层数、电梯数量和基础时间参数，返回初始化后的系统。
// 如果参数不合法（小于 1），返回错误。
func NewSystem(
	floors int,
	elevatorCount int,
	ticksPerFloor int,
	doorBaseTicks int,
	tickPerPassenger int,
) (*System, error) {
	// 数据合法性检查
	if floors < 1 {
		return nil, fmt.Errorf("floors must be at least 1, got %d", floors)
	}
	if elevatorCount < 1 {
		return nil, fmt.Errorf("elevator count must be at least 1, got %d", elevatorCount)
	}
	if ticksPerFloor < 1 {
		return nil, fmt.Errorf("ticks per floor must be at least 1, got %d", ticksPerFloor)
	}
	if doorBaseTicks < 0 {
		return nil, fmt.Errorf("door base ticks must be at least 0, got %d", doorBaseTicks)
	}
	if tickPerPassenger < 0 {
		return nil, fmt.Errorf("tick per passenger must be at least 0, got %d", tickPerPassenger)
	}

	elevators := make([]Elevator, elevatorCount)
	for i := range elevators {
		elevators[i] = Elevator{
			ID:                 i + 1,
			CurrentFloor:       1,
			Direction:          DirectionIdle,
			ScanDirection:      DirectionUp,
			DoorOpen:           false,
			TargetFloors:       []int{},
			TargetRequestIDs:   []int64{},
			MoveRemainingTicks: 0,
			DoorRemainingTicks: 0,
			EmergencyStop:      false,
		}
	}

	scheduler := FirstAvailableScheduler{}

	return &System{
		FloorCount:       floors,
		CurrentTick:      0,
		TicksPerFloor:    ticksPerFloor,
		DoorBaseTicks:    doorBaseTicks,
		TickPerPassenger: tickPerPassenger,
		Elevators:        elevators,
		Requests:         []Request{},
		SchedulerName:    scheduler.Name(),
		scheduler:        scheduler,
		nextRequestID:    1,
	}, nil
}

// AddRequest 向系统添加一个新的乘梯请求。先校验参数是否合法，再将请求保存到
// Requests 中，并用 Status 标记它当前处于 pending 状态。
func (s *System) AddRequest(floor int, direction Direction, kind RequestKind) (*Request, error) {
	if floor < 1 || floor > s.FloorCount {
		return nil, fmt.Errorf("floor must be between 1 and %d, got %d", s.FloorCount, floor)
	}
	if !IsValidDirection(direction) {
		return nil, fmt.Errorf("direction must be up, down, or idle, got %s", direction)
	}
	if !IsValidRequestKind(kind) {
		return nil, fmt.Errorf("kind must be hall or cabin, got %s", kind)
	}

	request := Request{
		ID:                 s.nextRequestID,
		Floor:              floor,
		Direction:          direction,
		Kind:               kind,
		Status:             RequestPending, // 默认状态是 pending，等待调度器分配
		CreatedTick:        s.CurrentTick,
		AssignedTick:       0,
		CompletedTick:      0,
		AssignedElevatorID: 0,
	}
	s.nextRequestID++
	s.Requests = append(s.Requests, request)
	return &s.Requests[len(s.Requests)-1], nil
}

// SetScheduler 根据名称切换调度算法。
func (s *System) SetScheduler(name string) error {
	scheduler, err := NewScheduler(name)
	if err != nil {
		return err
	}

	s.scheduler = scheduler
	s.SchedulerName = scheduler.Name()
	return nil
}

// Snapshot 返回系统当前状态的 JSON 快照，带缩进格式，便于调试和 HTTP API 使用。
func (s *System) Snapshot() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// Step 推进系统一个离散时间片。
// 每次调用 Step()，系统先进入下一个 tick，然后在这个 tick 内完成：
// 1. 调度器尝试分配请求。调度是即时决策，不额外消耗 tick。
// 2. 每部电梯执行一个行动单位，例如移动倒计时、开门倒计时或空闲等待。
func (s *System) Step() error {
	if len(s.Elevators) == 0 {
		return fmt.Errorf("system has no elevators")
	}

	if s.scheduler == nil {
		return fmt.Errorf("No valid scheduler.")
	}

	assigned := s.scheduler.Assign(s)
	if assigned {
		log.Println("assigned one request")
	}
	for i := range s.Elevators {
		stepElevator(s, &s.Elevators[i])
	}
	s.CurrentTick++
	return nil
}

// stepElevator 推进单部电梯一个行动 tick。
// 跨越相邻两层需要消耗 s.TicksPerFloor 个 tick；
// 到达目标层后，再用 s.DoorBaseTicks 表示开门停靠时间。
func stepElevator(s *System, e *Elevator) {
	// 紧急停止状态下不移动
	if e.EmergencyStop {
		e.Direction = DirectionIdle
		return
	}

	// 门开着时，当前 tick 用于停靠，不移动。
	if e.DoorOpen {
		if e.DoorRemainingTicks > 0 {
			e.DoorRemainingTicks--
		}
		if e.DoorRemainingTicks == 0 {
			e.DoorOpen = false
		}
		e.Direction = DirectionIdle
		return
	}

	// 没有目标楼层，保持空闲
	if len(e.TargetFloors) == 0 {
		e.Direction = DirectionIdle
		return
	}

	targetFloor := e.TargetFloors[0]

	// 当前已经在目标楼层：本 tick 用于开门、完成请求、移除目标。
	if e.CurrentFloor == targetFloor {
		e.Direction = DirectionIdle
		e.DoorOpen = true
		e.DoorRemainingTicks = s.DoorBaseTicks
		if len(e.TargetRequestIDs) > 0 {
			s.completeRequest(e.TargetRequestIDs[0], s.CurrentTick)
			e.TargetRequestIDs = e.TargetRequestIDs[1:]
		}
		e.TargetFloors = e.TargetFloors[1:]
		if e.DoorRemainingTicks == 0 {
			e.DoorOpen = false
		}
		return
	}

	// 当前不在目标楼层：本 tick 用于向目标方向移动。
	if e.CurrentFloor < targetFloor {
		e.Direction = DirectionUp
		moveOneTick(e, 1, s.TicksPerFloor)
		return
	}

	e.Direction = DirectionDown
	moveOneTick(e, -1, s.TicksPerFloor)
}

// moveOneTick 是一个辅助函数，用于将电梯 e 向目标方向移动一个 tick。
// floorDelta 应该是 1（向上）或 -1（向下）。函数会更新电梯的 CurrentFloor 和 MoveRemainingTicks。
func moveOneTick(e *Elevator, floorDelta int, ticksPerFloor int) {
	if e.MoveRemainingTicks == 0 {
		e.MoveRemainingTicks = ticksPerFloor
	}
	e.MoveRemainingTicks--
	if e.MoveRemainingTicks == 0 {
		e.CurrentFloor += floorDelta
	}
}

// 给 System 添加一些辅助方法，方便调度器调用：

// 用于将某个请求分配给某部电梯，更新请求状态并将目标楼层添加到电梯的任务列表中。
func (s *System) assignRequestToElevator(requestIndex int, elevatorIndex int) {
	request := &s.Requests[requestIndex]
	elevator := &s.Elevators[elevatorIndex]

	request.Status = RequestAssigned
	request.AssignedTick = s.CurrentTick
	request.AssignedElevatorID = elevator.ID

	elevator.TargetFloors = append(elevator.TargetFloors, request.Floor)
	elevator.TargetRequestIDs = append(elevator.TargetRequestIDs, request.ID)
}

// 用于将某个请求标记为完成状态，并记录完成的时间片。
func (s *System) completeRequest(requestID int64, completedTick int) {
	for i := range s.Requests {
		if s.Requests[i].ID != requestID {
			continue
		}
		s.Requests[i].Status = RequestDone
		s.Requests[i].CompletedTick = completedTick
		return
	}
}
