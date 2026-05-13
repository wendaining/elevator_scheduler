package elevator

import (
	"context"
)

// elevatorTickCommand 是发送给单部电梯 goroutine 的控制信号。
// 命令中带的是这部电梯的状态副本和本次 tick 所需的只读配置。
type elevatorTickCommand struct {
	elevator      Elevator
	ticksPerFloor int
	doorBaseTicks int
	done          chan elevatorTickResult
}

// elevatorTickResult 是单部电梯 goroutine 完成一个 tick 后返回的结果。
// goroutine 不直接修改 System；System.Step 统一合并这些结果。
type elevatorTickResult struct {
	elevator            Elevator
	completedRequestIDs []int64
	err                 error
}

// StartElevatorRunners 为每部电梯启动一个 goroutine。
// 每个 goroutine 独立计算自己负责的电梯在一个 tick 内的状态变化。
func (s *System) StartElevatorRunners(ctx context.Context) {
	s.mu.Lock()
	if s.elevatorRunnersStarted {
		s.mu.Unlock()
		return
	}

	runnerCtx, runnerCancel := context.WithCancel(ctx)
	commands := make([]chan elevatorTickCommand, len(s.Elevators))
	for i := range commands {
		commands[i] = make(chan elevatorTickCommand, 1)
	}
	done := make(chan struct{})
	s.elevatorCommands = commands
	s.elevatorRunnersDone = done
	s.elevatorRunnersStarted = true
	s.runnerCancel = runnerCancel
	s.mu.Unlock()

	go func() {
		<-runnerCtx.Done()
		close(done)
	}()

	for _, commandChannel := range commands {
		go s.runElevator(runnerCtx, commandChannel)
	}
}

// runElevator 是单部电梯 goroutine 的运行循环。
// 它不断等待 tick 命令，收到后只推进自己负责的那一部电梯。
func (s *System) runElevator(ctx context.Context, commands <-chan elevatorTickCommand) {
	for {
		select {
		case <-ctx.Done():
			return
		case command := <-commands:
			elevator, completedRequestIDs, err := stepElevatorState(
				command.elevator,
				command.ticksPerFloor,
				command.doorBaseTicks,
			)
			result := elevatorTickResult{
				elevator:            elevator,
				completedRequestIDs: completedRequestIDs,
				err:                 err,
			}
			select {
			case command.done <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// moveOneTick 将电梯向目标方向移动一个 tick。floorDelta 应为 1（向上）或 -1（向下）。
func moveOneTick(e *Elevator, floorDelta int, ticksPerFloor int) {
	if e.MoveRemainingTicks == 0 {
		e.MoveRemainingTicks = ticksPerFloor
	}
	e.MoveRemainingTicks--
	if e.MoveRemainingTicks == 0 {
		e.CurrentFloor += floorDelta
	}
}

// stepElevatorState 是单部电梯的纯状态推进函数。
// 它只读输入参数，返回更新后的 Elevator 和本 tick 完成的请求 ID。
func stepElevatorState(e Elevator, ticksPerFloor int, doorBaseTicks int) (Elevator, []int64, error) {
	if e.EmergencyStop {
		e.Direction = DirectionIdle
		if e.EmergencyRemainingTicks > 0 {
			e.EmergencyRemainingTicks--
		}
		if e.EmergencyRemainingTicks == 0 {
			e.EmergencyStop = false
		}
		return e, nil, nil
	}

	if e.DoorOpen {
		if e.DoorRemainingTicks > 0 {
			e.DoorRemainingTicks--
		}
		if e.DoorRemainingTicks == 0 {
			e.DoorOpen = false
		}
		e.Direction = DirectionIdle
		return e, nil, nil
	}

	if len(e.Stops) == 0 {
		e.Direction = DirectionIdle
		return e, nil, nil
	}

	nextStop := e.Stops[0]
	targetFloor := nextStop.Floor

	if e.CurrentFloor == targetFloor {
		e.Direction = DirectionIdle
		e.DoorOpen = true
		e.DoorRemainingTicks = doorBaseTicks
		e.Stops = e.Stops[1:]
		if e.DoorRemainingTicks == 0 {
			e.DoorOpen = false
		}
		return e, append([]int64(nil), nextStop.RequestIDs...), nil
	}

	if e.CurrentFloor < targetFloor {
		e.Direction = DirectionUp
		moveOneTick(&e, 1, ticksPerFloor)
		return e, nil, nil
	}

	e.Direction = DirectionDown
	moveOneTick(&e, -1, ticksPerFloor)
	return e, nil, nil
}
