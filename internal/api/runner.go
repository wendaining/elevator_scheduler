package api

import (
	"context"
	"log"
	"time"
)

// StartAutoStep 启动后台自动 ticker。interval 控制系统时间片推进频率。
// 系统重启时内部会重新启动 auto-step，调用方不需要再次调用本方法。
func (s *Server) StartAutoStep(ctx context.Context, interval time.Duration) {
	s.mu.Lock()
	s.baseCtx = ctx
	s.autoStepInterval = interval
	s.autoStepStarted = true
	s.mu.Unlock()
	s.startAutoStepLocked()
}

// startAutoStepLocked 启动一个新的 auto-step goroutine。调用者必须持有 s.mu。
func (s *Server) startAutoStepLocked() {
	ctx, cancel := context.WithCancel(s.baseCtx)
	s.autoStepCancel = cancel

	go func() {
		ticker := time.NewTicker(s.autoStepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.mu.Lock()
				sys := s.System
				s.mu.Unlock()
				if err := sys.Step(); err != nil {
					log.Printf("auto step failed: %v", err)
				}
			}
		}
	}()
}
