package elevator

import (
	"encoding/json"
	"fmt"
	"log"
)

// NewSystem 是 System 的构造函数，接收楼层数、电梯数量和基础时间参数，返回初始化后的系统。
// 如果参数不合法（小于 1），返回错误。
func NewSystem(sc SystemConfig) (*System, error) {
	// 数据合法性检查
	if sc.Floors < 1 {
		return nil, fmt.Errorf("floors must be at least 1, got %d", sc.Floors)
	}
	if sc.ElevatorCount < 1 {
		return nil, fmt.Errorf("elevator count must be at least 1, got %d", sc.ElevatorCount)
	}
	if sc.TicksPerFloor < 1 {
		return nil, fmt.Errorf("ticks per floor must be at least 1, got %d", sc.TicksPerFloor)
	}
	if sc.DoorBaseTicks < 0 {
		return nil, fmt.Errorf("door base ticks must be at least 0, got %d", sc.DoorBaseTicks)
	}
	if sc.TickPerPassenger < 0 {
		return nil, fmt.Errorf("tick per passenger must be at least 0, got %d", sc.TickPerPassenger)
	}

	requestStore, err := OpenRequestStore(sc.DatabasePath)
	if err != nil {
		return nil, err
	}

	elevators := make([]Elevator, sc.ElevatorCount)
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

	scheduler := SCANScheduler{}

	return &System{
		FloorCount:       sc.Floors,
		CurrentTick:      0,
		TicksPerFloor:    sc.TicksPerFloor,
		DoorBaseTicks:    sc.DoorBaseTicks,
		TickPerPassenger: sc.TickPerPassenger,
		Elevators:        elevators,
		Requests:         map[int64]*Request{},
		SchedulerName:    scheduler.Name(),
		scheduler:        scheduler,
		requestStore:     requestStore,
		nextRequestID:    1,
	}, nil
}

// AddRequest 向系统添加一个新的乘梯请求。校验参数后创建请求并写入 Requests map。
//
// elevatorID 仅对 cabin 请求有意义，表示请求来自哪部电梯（1-based）。
// hall 请求应传 0。
//
// cabin 请求不会进入 pending 状态，而是在此方法内直接分配给对应电梯。
// 这样所有调度器都永远看不到 cabin 请求，cabin 请求不可能被分配给错误的电梯。
func (s *System) AddRequest(floor int, direction Direction, kind RequestKind, elevatorID int) (*Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.addRequestLocked(floor, direction, kind, elevatorID)
}

func (s *System) addRequestLocked(floor int, direction Direction, kind RequestKind, elevatorID int) (*Request, error) {
	if floor < 1 || floor > s.FloorCount {
		return nil, fmt.Errorf("floor must be between 1 and %d, got %d", s.FloorCount, floor)
	}
	if !IsValidDirection(direction) {
		return nil, fmt.Errorf("direction must be up, down, or idle, got %s", direction)
	}
	if !IsValidRequestKind(kind) {
		return nil, fmt.Errorf("kind must be hall or cabin, got %s", kind)
	}
	if kind == RequestKindCabin {
		if elevatorID < 1 || elevatorID > len(s.Elevators) {
			return nil, fmt.Errorf("cabin request must specify a valid elevator ID between 1 and %d, got %d", len(s.Elevators), elevatorID)
		}
	} else {
		elevatorID = 0
	}

	request := Request{
		ID:                 s.nextRequestID,
		Floor:              floor,
		Direction:          direction,
		Kind:               kind,
		Status:             RequestPending,
		ElevatorID:         elevatorID,
		CreatedTick:        s.CurrentTick,
		AssignedTick:       0,
		CompletedTick:      0,
		AssignedElevatorID: 0,
	}
	s.Requests[request.ID] = &request
	s.nextRequestID++
	log.Printf(
		"request created: id=%d floor=%d direction=%s kind=%s elevator=%d tick=%d",
		request.ID,
		request.Floor,
		request.Direction,
		request.Kind,
		request.ElevatorID,
		request.CreatedTick,
	)

	// cabin 请求直接分配给指定电梯，不进入 pending。
	if kind == RequestKindCabin {
		s.assignRequestToElevator(request.ID, elevatorID-1)
		sortElevatorStops(&s.Elevators[elevatorID-1])
	}

	return s.Requests[request.ID], nil
}

// SetScheduler 根据名称切换调度算法。
func (s *System) SetScheduler(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduler, err := NewScheduler(name)
	if err != nil {
		return err
	}

	s.scheduler = scheduler
	s.SchedulerName = scheduler.Name()
	return nil
}

// Close 停止所有后台 goroutine 并释放外部资源。
func (s *System) Close() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.runnerCancel != nil {
		s.runnerCancel()
		s.runnerCancel = nil
	}

	return s.requestStore.Close()
}

// Snapshot 返回系统当前状态的 JSON 快照，带缩进格式，便于调试和 HTTP API 使用。
func (s *System) Snapshot() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return json.MarshalIndent(s, "", "  ")
}

// Step 推进系统一个离散时间片。
// 每次调用 Step()，系统先进入下一个 tick，然后在这个 tick 内完成：
// 1. 调度器尝试分配请求。调度是即时决策，不额外消耗 tick。
// 2. 每部电梯执行一个行动单位，例如移动倒计时、开门倒计时或空闲等待。
func (s *System) Step() error {
	s.stepMu.Lock()
	defer s.stepMu.Unlock()

	return s.stepWithElevatorRunners()
}

// stepWithElevatorRunners 在调度器完成分配后，把一个 tick 命令发送给每部电梯 goroutine。
func (s *System) stepWithElevatorRunners() error {
	s.mu.Lock()
	if !s.elevatorRunnersStarted {
		s.mu.Unlock()
		return fmt.Errorf("elevator runners are not started")
	}

	if len(s.Elevators) == 0 {
		s.mu.Unlock()
		return fmt.Errorf("system has no elevators")
	}

	if s.scheduler == nil {
		s.mu.Unlock()
		return fmt.Errorf("No valid scheduler.")
	}

	assigned := s.scheduler.Assign(s)
	_ = assigned
	// 深拷贝的写法，把后者切片展开然后拷贝进一个空切片
	commands := append([]chan elevatorTickCommand(nil), s.elevatorCommands...)
	doneSignal := s.elevatorRunnersDone
	if len(commands) != len(s.Elevators) {
		s.mu.Unlock()
		return fmt.Errorf("elevator runner count %d does not match elevator count %d", len(commands), len(s.Elevators))
	}

	elevators := make([]Elevator, len(s.Elevators))
	for i := range s.Elevators {
		elevators[i] = cloneElevator(s.Elevators[i])
	}
	currentTick := s.CurrentTick
	ticksPerFloor := s.TicksPerFloor
	doorBaseTicks := s.DoorBaseTicks
	s.mu.Unlock()

	doneChannels := make([]chan elevatorTickResult, len(commands))
	for i, commandChannel := range commands {
		done := make(chan elevatorTickResult, 1)
		doneChannels[i] = done
		command := elevatorTickCommand{
			elevator:      elevators[i],
			currentTick:   currentTick,
			ticksPerFloor: ticksPerFloor,
			doorBaseTicks: doorBaseTicks,
			done:          done,
		}
		select {
		case commandChannel <- command:
		case <-doneSignal:
			return fmt.Errorf("elevator runners stopped")
		}
	}

	results := make([]elevatorTickResult, len(doneChannels))
	for i, done := range doneChannels {
		select {
		case result := <-done:
			if result.err != nil {
				return result.err
			}
			results[i] = result
		case <-doneSignal:
			return fmt.Errorf("elevator runners stopped")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, result := range results {
		s.Elevators[i] = result.elevator
		for _, requestID := range result.completedRequestIDs {
			if err := s.completeRequest(requestID, currentTick); err != nil {
				return err
			}
		}
	}
	s.CurrentTick++
	return nil
}

func cloneElevator(e Elevator) Elevator {
	if len(e.Stops) == 0 {
		return e
	}

	e.Stops = append([]StopPlan(nil), e.Stops...)
	for i := range e.Stops {
		e.Stops[i].RequestIDs = append([]int64(nil), e.Stops[i].RequestIDs...)
	}
	return e
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
	logRequestAssigned(s, *request, elevator.ID)
}

// completeRequest 将指定 ID 的请求标记为完成，写入 SQLite，
// 然后将其从运行态 Requests 中删除。
func (s *System) completeRequest(requestID int64, completedTick int) error {
	req, ok := s.Requests[requestID]
	if !ok {
		return nil
	}

	req.Status = RequestDone
	req.CompletedTick = completedTick

	if err := s.requestStore.SaveCompletedRequest(*req); err != nil {
		return err
	}

	log.Printf(
		"request completed: id=%d elevator=%d floor=%d direction=%s kind=%s createdTick=%d assignedTick=%d completedTick=%d",
		req.ID,
		req.AssignedElevatorID,
		req.Floor,
		req.Direction,
		req.Kind,
		req.CreatedTick,
		req.AssignedTick,
		req.CompletedTick,
	)
	delete(s.Requests, requestID)
	return nil
}

func logRequestAssigned(s *System, request Request, elevatorID int) {
	log.Printf(
		"request assigned: id=%d elevator=%d floor=%d direction=%s kind=%s scheduler=%s tick=%d",
		request.ID,
		elevatorID,
		request.Floor,
		request.Direction,
		request.Kind,
		s.SchedulerName,
		s.CurrentTick,
	)
}
