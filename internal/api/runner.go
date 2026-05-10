package api

import (
	"context"
	"log"
	"time"
)

// StartAutoStep 启动一个后台 goroutine，按固定间隔推进电梯系统。
// 它是当前项目的最小并发版本：只有一个后台循环调用 Step，
func (s *Server) StartAutoStep(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.System.Step(); err != nil {
					log.Printf("auto step failed: %v", err)
				}
			}
		}
	}()
}
