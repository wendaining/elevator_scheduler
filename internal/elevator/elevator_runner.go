package elevator

import (
	"context"
	"fmt"
)

// elevatorTickCommand 是发送给单部电梯 goroutine 的控制信号。
// done 用于把该电梯本次 tick 的执行结果传回 System.Step。
type elevatorTickCommand struct {
	done chan error
}

// StartElevatorRunners 为每部电梯启动一个 goroutine。
// 当前版本中，每个 goroutine 只负责在收到 tick 命令时推进自己对应的电梯一步。
func (s *System) StartElevatorRunners(ctx context.Context) {
	s.mu.Lock()
	if s.elevatorRunnersStarted {
		s.mu.Unlock()
		return
	}

	commands := make([]chan elevatorTickCommand, len(s.Elevators))
	for i := range commands {
		commands[i] = make(chan elevatorTickCommand)
	}
	s.elevatorCommands = commands
	s.elevatorRunnersStarted = true
	s.mu.Unlock()

	for elevatorIndex, commandChannel := range commands {
		go s.runElevator(ctx, elevatorIndex, commandChannel)
	}
}

// runElevator 是单部电梯 goroutine 的运行循环。
// 它不断等待 tick 命令，收到后只推进自己负责的那一部电梯。
func (s *System) runElevator(ctx context.Context, elevatorIndex int, commands <-chan elevatorTickCommand) {
	for {
		select {
		case <-ctx.Done():
			return
		case command := <-commands:
			command.done <- s.stepElevatorByIndex(elevatorIndex)
		}
	}
}

// stepElevatorByIndex 在 System 锁保护下推进指定下标的电梯。
func (s *System) stepElevatorByIndex(elevatorIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if elevatorIndex < 0 || elevatorIndex >= len(s.Elevators) {
		return fmt.Errorf("elevator index %d out of range", elevatorIndex)
	}

	return stepElevator(s, &s.Elevators[elevatorIndex])
}
