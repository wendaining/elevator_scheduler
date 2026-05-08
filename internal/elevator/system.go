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
	return NewSystemWithDatabase(
		floors,
		elevatorCount,
		ticksPerFloor,
		doorBaseTicks,
		tickPerPassenger,
		":memory:",
	)
}

// NewSystemWithDatabase 创建电梯系统，并把已完成请求持久化到指定 SQLite 数据库。
func NewSystemWithDatabase(
	floors int,
	elevatorCount int,
	ticksPerFloor int,
	doorBaseTicks int,
	tickPerPassenger int,
	databasePath string,
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

	requestStore, err := OpenRequestStore(databasePath)
	if err != nil {
		return nil, err
	}
	maxCompletedRequestID, err := requestStore.MaxCompletedRequestID()
	if err != nil {
		requestStore.Close()
		return nil, err
	}

	elevators := make([]Elevator, elevatorCount)
	for i := range elevators {
		elevators[i] = Elevator{
			ID:                 i + 1,
			CurrentFloor:       1,
			Direction:          DirectionIdle,
			ScanDirection:      DirectionUp,
			DoorOpen:           false,
			Stops:              []StopPlan{},
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
		Requests:         map[int64]*Request{},
		SchedulerName:    scheduler.Name(),
		scheduler:        scheduler,
		requestStore:     requestStore,
		nextRequestID:    maxCompletedRequestID + 1,
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
	s.Requests[request.ID] = &request
	s.nextRequestID++
	return s.Requests[request.ID], nil
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

// Close 释放 System 持有的外部资源，例如 SQLite 数据库连接。
func (s *System) Close() error {
	if s == nil {
		return nil
	}
	return s.requestStore.Close()
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
		if err := stepElevator(s, &s.Elevators[i]); err != nil {
			return err
		}
	}
	s.CurrentTick++
	return nil
}

// stepElevator 推进单部电梯一个行动 tick。
// 跨越相邻两层需要消耗 s.TicksPerFloor 个 tick；
// 到达目标层后，再用 s.DoorBaseTicks 表示开门停靠时间。
func stepElevator(s *System, e *Elevator) error {
	// 紧急停止状态下不移动
	if e.EmergencyStop {
		e.Direction = DirectionIdle
		return nil
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
		return nil
	}

	// 没有停靠计划，保持空闲
	if len(e.Stops) == 0 {
		e.Direction = DirectionIdle
		return nil
	}

	nextStop := e.Stops[0]
	targetFloor := nextStop.Floor

	// 当前已经在目标楼层：本 tick 用于开门、完成请求、移除目标。
	if e.CurrentFloor == targetFloor {
		e.Direction = DirectionIdle
		e.DoorOpen = true
		e.DoorRemainingTicks = s.DoorBaseTicks
		for _, requestID := range nextStop.RequestIDs {
			if err := s.completeRequest(requestID, s.CurrentTick); err != nil {
				return err
			}
		}
		e.Stops = e.Stops[1:]
		if e.DoorRemainingTicks == 0 {
			e.DoorOpen = false
		}
		return nil
	}

	// 当前不在目标楼层：本 tick 用于向目标方向移动。
	if e.CurrentFloor < targetFloor {
		e.Direction = DirectionUp
		moveOneTick(e, 1, s.TicksPerFloor)
		return nil
	}

	e.Direction = DirectionDown
	moveOneTick(e, -1, s.TicksPerFloor)
	return nil
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

// 用于将某个请求分配给某部电梯，更新请求状态并将停靠计划添加到电梯任务列表中。
func (s *System) assignRequestToElevator(requestID int64, elevatorIndex int) {
	request := s.Requests[requestID]
	elevator := &s.Elevators[elevatorIndex]

	request.Status = RequestAssigned
	request.AssignedTick = s.CurrentTick
	request.AssignedElevatorID = elevator.ID

	addStopPlan(elevator, stopPlanFromRequest(*request))
}

// completeRequest 将指定 ID 的请求标记为完成，写入 SQLite，
// 然后将其从运行态 Requests 中删除。
func (s *System) completeRequest(requestID int64, completedTick int) error {
	req, ok := s.Requests[requestID]
	if !ok {
		return nil
	}

	completedRequest := *req
	completedRequest.Status = RequestDone
	completedRequest.CompletedTick = completedTick

	// 先写数据库，再从运行态 map 删除
	if err := s.requestStore.SaveCompletedRequest(completedRequest); err != nil {
		return err
	}

	*req = completedRequest
	delete(s.Requests, requestID)
	return nil
}
