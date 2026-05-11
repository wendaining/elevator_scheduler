package elevator

import "sync"

// Direction 是一个自定义的字符串类型。
//
// Go 语法说明：
//   - "type Direction string" 创建一个名为 Direction 的新类型。
//   - 它的内部行为与 string 相同，但使用命名类型可以使代码更易读，
//     并防止意外混用不相关的字符串。
type Direction string

// 这些常量是系统使用的唯一方向值。
const (
	DirectionIdle Direction = "idle"
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

// 记录每个请求的状态
type RequestStatus string

const (
	RequestPending  RequestStatus = "pending"
	RequestAssigned RequestStatus = "assigned"
	RequestDone     RequestStatus = "done"
)

// RequestKind 描述请求的来源。
//
// Hall 请求在电梯外部创建，例如在 5 楼按下"上行"按钮。
// Cabin 请求在电梯内部创建，例如进入电梯后按下 12 楼按钮。
type RequestKind string

const (
	RequestKindHall  RequestKind = "hall"
	RequestKindCabin RequestKind = "cabin"
)

// StopReason 描述电梯为什么要在某层停靠。
// 同一楼层的 hall up、hall down、cabin target 不能简单合并成一个 int。
type StopReason string

const (
	StopReasonHallUp   StopReason = "hall_up"
	StopReasonHallDown StopReason = "hall_down"
	StopReasonCabin    StopReason = "cabin"
)

// Request 表示一个乘客请求。
//
// Go 语法说明：
//   - "struct" 将多个字段组合成一个数据类型。
//   - 字段名以大写字母开头，以便 API 层后续将其编码为 JSON。
//   - 每个字段后面的文本，如 `json:"floor"`，是一个 struct tag。
//     它告诉 Go 的 JSON 编码器在返回 API 数据时使用哪个键名。
type Request struct {
	ID        int64         `json:"id"`
	Floor     int           `json:"floor"`
	Direction Direction     `json:"direction"`
	Kind      RequestKind   `json:"kind"`
	Status    RequestStatus `json:"status"`

	// CreatedTick 记录请求是在第几个系统时间片创建的。
	CreatedTick        int `json:"createdTick"`
	AssignedTick       int `json:"assignedTick"`
	CompletedTick      int `json:"completedTick"`
	AssignedElevatorID int `json:"assignedElevatorId"`
}

// StopPlan 表示一部电梯的一次停靠计划。
//
// Floor 说明停在哪层。
// Reason 说明为什么停，例如接上行 hall 请求、接下行 hall 请求、或响应 cabin 目标。
// Direction 保留请求方向，方便调度器判断顺路关系。
// RequestIDs 记录这个停靠会完成哪些请求。
type StopPlan struct {
	Floor      int        `json:"floor"`
	Reason     StopReason `json:"reason"`
	Direction  Direction  `json:"direction"`
	RequestIDs []int64    `json:"requestIds"`
}

// Elevator 表示一部电梯轿厢。
type Elevator struct {
	ID           int       `json:"id"`
	CurrentFloor int       `json:"currentFloor"`
	Direction    Direction `json:"direction"`
	// ScanDirection 是 SCAN 算法使用的长期扫描方向。
	// 它和 Direction 不同：Direction 表示当前是否正在移动，
	// ScanDirection 表示空闲接单时优先沿哪个方向寻找请求。
	ScanDirection Direction `json:"scanDirection"`
	DoorOpen      bool      `json:"doorOpen"`
	// Stops 是此电梯当前的停靠计划列表。
	Stops []StopPlan `json:"stops"`
	// MoveRemainingTicks 表示当前跨越相邻两层还剩多少 tick。
	// 它让 TicksPerFloor 真正参与模拟，而不是每次 Step 都移动一整层。
	MoveRemainingTicks int `json:"moveRemainingTicks"`
	// DoorRemainingTicks 表示门保持打开还剩多少 tick。
	DoorRemainingTicks int  `json:"doorRemainingTicks"`
	EmergencyStop      bool `json:"emergencyStop"`
}

// System 表示整个电梯调度系统。
type System struct {
	mu     sync.Mutex
	stepMu sync.Mutex

	FloorCount int `json:"floorCount"`

	// CurrentTick 是整个模拟系统的全局离散时钟。
	// 每调用一次 Step()，系统向前推进一个时间片。
	CurrentTick int `json:"currentTick"`

	// 下面三个字段是后续更真实时间模型的配置预留。
	TicksPerFloor    int `json:"ticksPerFloor"`    // 移动一层需要的 tick 数
	DoorBaseTicks    int `json:"doorBaseTicks"`    // 开关门的基础 tick 数
	TickPerPassenger int `json:"tickPerPassenger"` // 每多一个乘客，开关门额外增加的 tick 数

	// Elevators 存储大楼中的所有电梯轿厢。
	Elevators []Elevator `json:"elevators"`

	// Requests 是运行态请求表，只保存 pending 和 assigned 状态的请求。
	// 请求完成后会写入 SQLite，并从本 map 中删除。
	// key 为请求 ID，value 为请求指针。
	Requests map[int64]*Request `json:"requests"`

	// SchedulerName 会暴露给 API 和前端，方便观察当前调度策略。
	SchedulerName string `json:"schedulerName"`

	// scheduler 是真正执行调度逻辑的对象。字段名小写，所以 JSON 不会暴露它。
	scheduler Scheduler

	// requestStore 负责把已完成请求写入 SQLite。
	// 字段名小写且 json:"-"，表示它属于后端内部实现，不出现在 API 状态里。
	requestStore *RequestStore `json:"-"`

	elevatorCommands       []chan elevatorTickCommand
	elevatorRunnersDone    <-chan struct{}
	elevatorRunnersStarted bool

	nextRequestID int64
}

type SystemConfig struct {
	Floors           int
	ElevatorCount    int
	TicksPerFloor    int
	DoorBaseTicks    int
	TickPerPassenger int
	DatabasePath     string
}
