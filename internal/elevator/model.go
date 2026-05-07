package elevator

// Direction 是一个自定义的字符串类型。
//
// Go 语法说明：
//   - "type Direction string" 创建一个名为 Direction 的新类型。
//   - 它的内部行为与 string 相同，但使用命名类型可以使代码更易读，
//     并防止意外混用不相关的字符串。
type Direction string

// 这些常量是系统第一版使用的唯一方向值。
const (
	DirectionIdle Direction = "idle"
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
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

// Request 表示一个乘客请求。
//
// Go 语法说明：
//   - "struct" 将多个字段组合成一个数据类型。
//   - 字段名以大写字母开头，以便 API 层后续将其编码为 JSON。
//   - 每个字段后面的文本，如 `json:"floor"`，是一个 struct tag。
//     它告诉 Go 的 JSON 编码器在返回 API 数据时使用哪个键名。
type Request struct {
	Floor     int         `json:"floor"`
	Direction Direction   `json:"direction"`
	Kind      RequestKind `json:"kind"`

	// CreatedTick 记录请求是在第几个系统时间片创建的。
	// 后续重构为完整 Requests 模型后，会继续使用 tick 记录分配和完成时间。
	CreatedTick   int `json:"createdTick"`
	AssignedTick  int `json:"assignedTick"`
	CompletedTick int `json:"completedTick"`
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
	// TargetFloors 是此电梯当前的简单任务列表。
	TargetFloors  []int `json:"targetFloors"`
	EmergencyStop bool  `json:"emergencyStop"`
}

// System 表示整个电梯调度系统。
//
// 在第一版模型中，此结构体仅描述状态。它尚未包含 goroutine、channel 或调度算法。
type System struct {
	FloorCount int `json:"floorCount"`

	// CurrentTick 是整个模拟系统的全局离散时钟。
	// 每调用一次 Step()，系统向前推进一个时间片。
	CurrentTick int `json:"currentTick"`

	// 下面三个字段是后续更真实时间模型的配置预留。
	// 当前 Step() 仍然是“一次移动一层”的简单模型，后续会改成跨楼层和开门都消耗多个 tick。
	TicksPerFloor    int `json:"ticksPerFloor"`
	DoorBaseTicks    int `json:"doorBaseTicks"`
	TickPerPassenger int `json:"tickPerPassenger"`

	// Elevators 存储大楼中的所有电梯轿厢。
	Elevators []Elevator `json:"elevators"`

	// PendingRequests 存储尚未完全服务的请求。
	PendingRequests []Request `json:"pendingRequests"`

	// SchedulerName 会暴露给 API 和前端，方便观察当前调度策略。
	SchedulerName string `json:"schedulerName"`

	// scheduler 是真正执行调度逻辑的对象。字段名小写，所以 JSON 不会暴露它。
	scheduler Scheduler
}
