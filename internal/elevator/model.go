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

	// TODO: 考虑一下请求是否应该记录创建时间。如果需要，应该使用什么类型？
}

// Elevator 表示一部电梯轿厢。
type Elevator struct {
	ID           int       `json:"id"`
	CurrentFloor int       `json:"currentFloor"`
	Direction    Direction `json:"direction"`
	DoorOpen     bool      `json:"doorOpen"`
	// TargetFloors 是此电梯当前的简单任务列表。
	TargetFloors  []int `json:"targetFloors"`
	EmergencyStop bool  `json:"emergencyStop"`
}

// System 表示整个电梯调度系统。
//
// 在第一版模型中，此结构体仅描述状态。它尚未包含 goroutine、channel 或调度算法。
type System struct {
	FloorCount int `json:"floorCount"`

	// Elevators 存储大楼中的所有电梯轿厢。
	Elevators []Elevator `json:"elevators"`

	// PendingRequests 存储尚未完全服务的请求。
	PendingRequests []Request `json:"pendingRequests"`

	// SchedulerName 会暴露给 API 和前端，方便观察当前调度策略。
	SchedulerName string `json:"schedulerName"`

	// scheduler 是真正执行调度逻辑的对象。字段名小写，所以 JSON 不会暴露它。
	scheduler Scheduler
}
