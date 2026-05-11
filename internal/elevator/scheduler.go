package elevator

import "fmt"

// Scheduler 是所有调度算法都要实现的统一接口。
//
// Go 语法说明：
//   - interface 描述“一个类型必须有哪些方法”。
//   - 只要某个类型实现了 Name 和 Assign，它就是 Scheduler。
//   - System 不需要关心具体算法类型，只需要调用 Scheduler.Assign。
type Scheduler interface {
	Name() string
	Assign(system *System) bool
}

func NewScheduler(name string) (Scheduler, error) {
	switch name {
	case "first-available":
		return FirstAvailableScheduler{}, nil
	case "nearest-idle":
		return NearestIdleScheduler{}, nil
	case "fcfs":
		return FCFSScheduler{}, nil
	case "scan":
		return SCANScheduler{}, nil
	default:
		return nil, fmt.Errorf("unknown scheduler %q", name)
	}
}
