package api

import (
	"context"
	"log"
	"time"
)

// stepCommand 是发送给后台 runner 的控制信号。
// done 用来把 Step 的执行结果传回发送方。
type stepCommand struct {
	done chan error
}

// StartAutoStep 启动一个后台 goroutine，按固定间隔推进电梯系统。
// 自动 ticker 和手动 step 请求都会进入同一个 runner goroutine。
func (s *Server) StartAutoStep(ctx context.Context, interval time.Duration) {
	s.ensureStepCommands()
	s.stepRunnerStarted = true

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runStepCommand(nil)
			case command := <-s.stepCommands:
				s.runStepCommand(command.done)
			}
		}
	}()
}

// RequestStep 通过 channel 向后台 runner 发送一次手动 step 控制信号。
func (s *Server) RequestStep(ctx context.Context) error {
	if !s.stepRunnerStarted {
		return s.System.Step()
	}

	s.ensureStepCommands()

	done := make(chan error, 1)
	command := stepCommand{done: done}

	select {
	case s.stepCommands <- command:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) runStepCommand(done chan<- error) {
	err := s.System.Step()
	if err != nil {
		log.Printf("step failed: %v", err)
	}
	if done != nil {
		done <- err
	}
}

func (s *Server) ensureStepCommands() {
	if s.stepCommands != nil {
		return
	}
	s.stepCommands = make(chan stepCommand)
}
