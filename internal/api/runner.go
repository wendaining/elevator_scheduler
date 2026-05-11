package api

import (
	"context"
	"log"
	"time"
)

// StartAutoStep 启动一个后台 goroutine，按固定间隔推进电梯系统。
// 系统时间只由这个 ticker 推进；HTTP API 不再提供手动 Step 入口。
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
