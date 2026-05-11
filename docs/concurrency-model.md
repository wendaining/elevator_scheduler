# 电梯系统并发模型详解

本文专门解释本项目的并发设计，重点对应这次提交：

```text
7901ecca278304595a082c0f9b06467f2651419b
refactor: 重构电梯的并发模型
```

这份文档的目标不是只说明“代码改了什么”，而是从没有并发编程经验的视角，把这套设计为什么这样写、每个 goroutine 在干什么、channel 和锁分别负责什么讲清楚。

## 先建立直觉：为什么电梯系统适合并发

现实世界里的电梯不是这样运行的：

```text
先让 1 号电梯动一下
再让 2 号电梯动一下
再让 3 号电梯动一下
...
```

现实世界更接近这样：

```text
调度系统统一决定每部电梯要去哪
每部电梯各自运行
系统不断观察每部电梯的状态
```

所以在操作系统课程项目里，“每部电梯作为独立执行单元”是一个很自然的并发建模方式。

在 Go 里，独立执行单元通常用：

```go
go someFunction()
```

也就是 goroutine。

但是，并发程序最容易出问题的地方也在这里：

```text
多个 goroutine 如果同时读写同一份数据，就可能产生数据竞争。
```

例如：

```text
goroutine A 正在修改 s.Elevators[0].CurrentFloor
goroutine B 正在把 s.Elevators 编码成 JSON 返回给前端
```

这两个操作如果没有保护，就会变成数据竞争。

Go 的 race detector 可以检查这类问题：

```bash
go test -race ./...
```

本项目的并发模型要同时满足两件事：

```text
1. 每部电梯确实由自己的 goroutine 负责运行计算。
2. System 这种共享状态不能被多个 goroutine 随便同时写。
```

## Go 并发概念速览

### goroutine

goroutine 是 Go 的轻量级并发执行单元。

普通函数调用：

```go
runElevator()
```

含义是：

```text
当前代码必须等 runElevator() 执行完，才能继续往下走。
```

goroutine 调用：

```go
go runElevator()
```

含义是：

```text
启动一个新的并发执行流，让它在后台跑。
当前代码不会等待它结束，会继续往下执行。
```

本项目里，每部电梯会启动一个 goroutine：

```go
go s.runElevator(ctx, elevatorIndex, commandChannel)
```

如果默认有 5 部电梯，那么就会有 5 个电梯 goroutine。

### channel

channel 是 goroutine 之间传递消息的管道。

可以把它理解成：

```text
一个 goroutine 把任务放进 channel
另一个 goroutine 从 channel 里取任务
```

本项目里，`System.Step()` 给每部电梯发送一次 tick 命令：

```go
commandChannel <- command
```

电梯 goroutine 在自己的循环里接收命令：

```go
case command := <-commands:
```

这表示：

```text
System.Step() 不直接调用“移动电梯”的函数。
System.Step() 把命令发给电梯 goroutine。
电梯 goroutine 收到命令后自己计算。
```

### mutex

mutex 是互斥锁，用来保护共享数据。

Go 里常见写法是：

```go
s.mu.Lock()
defer s.mu.Unlock()
```

含义是：

```text
从 Lock 到 Unlock 之间，同一时间只能有一个 goroutine 进入这段代码。
```

本项目里 `System` 是共享状态：

```go
type System struct {
	mu     sync.Mutex
	stepMu sync.Mutex

	CurrentTick int
	Elevators   []Elevator
	Requests    map[int64]*Request
	...
}
```

这些字段会被 API、自动 tick runner、调度器、电梯运行逻辑访问，所以必须有明确的并发边界。

### context

`context.Context` 在这里主要用来通知后台 goroutine 退出。

`cmd/server/main.go` 里创建：

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

然后把 `ctx` 传给后台 runner：

```go
system.StartElevatorRunners(ctx)
server.StartAutoStep(ctx, defaultAutoStepInterval)
```

当程序结束、`cancel()` 被调用时，goroutine 里的：

```go
case <-ctx.Done():
	return
```

就会触发，后台 goroutine 可以退出。

## 当前系统里有哪些 goroutine

启动后，系统里主要有这些并发执行流。

### 1. HTTP server goroutine

`net/http` 会为请求处理提供并发能力。

比如前端同时发：

```text
GET /api/state
POST /api/request
```

这些请求可能由不同 goroutine 处理。

所以 API handler 不能直接随便读写 `System`，必须通过 `System` 自己提供的并发安全方法。

例如：

```go
s.System.AddRequestSnapshot(...)
s.System.Snapshot()
```

### 2. 自动 step runner goroutine

`internal/api/runner.go` 里：

```go
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
```

下面逐行拆开解释。

先看函数签名：

```go
func (s *Server) StartAutoStep(ctx context.Context, interval time.Duration)
```

这里的 `(s *Server)` 表示这是 `Server` 结构体的方法。

调用时写：

```go
server.StartAutoStep(ctx, defaultAutoStepInterval)
```

参数有两个：

```text
ctx
  用来控制这个后台 goroutine 什么时候退出。

interval
  自动推进系统的时间间隔。
  当前 main.go 里传入的是 defaultAutoStepInterval，也就是 500ms。
```

### `go func() { ... }()`

然后是：

```go
go func() {
	...
}()
```

这是 Go 里很常见的写法：启动一个匿名函数作为 goroutine。

拆开看：

```go
func() {
	...
}
```

这是一个没有名字的函数。

最后的：

```go
()
```

表示立刻调用这个函数。

前面的：

```go
go
```

表示不要在当前 goroutine 里执行它，而是新开一个 goroutine 在后台执行。

所以：

```go
go func() {
	...
}()
```

整体含义是：

```text
启动一个后台循环，让它负责系统时钟。
StartAutoStep() 自己不会卡住，会很快返回。
```

如果这里不加 `go`：

```go
func() {
	for {
		...
	}
}()
```

那么当前函数会进入无限循环，`main.go` 后面的 HTTP server 就没机会继续启动。

### `ticker := time.NewTicker(interval)`

goroutine 内部第一行：

```go
ticker := time.NewTicker(interval)
```

`time.NewTicker(interval)` 会创建一个定时器。

它每隔 `interval` 时间，就会向自己的 channel 发送一次信号。

这个 channel 是：

```go
ticker.C
```

如果 `interval` 是 500ms，可以理解成：

```text
每 500ms，ticker.C 上就会出现一次消息。
```

后面这段：

```go
case <-ticker.C:
	if err := s.System.Step(); err != nil {
		log.Printf("auto step failed: %v", err)
	}
```

就是在等待这个消息。

### `defer ticker.Stop()`

下一行：

```go
defer ticker.Stop()
```

`defer` 表示：

```text
等当前函数返回之前，再执行这句。
```

这里当前函数指的是这个后台匿名函数：

```go
go func() {
	...
}()
```

所以 `defer ticker.Stop()` 的意思是：

```text
当后台 goroutine 准备退出时，停止 ticker，释放 ticker 相关资源。
```

如果不 stop ticker，它可能继续在后台维护计时资源。

### `for { ... }`

然后是：

```go
for {
	...
}
```

Go 里的无限循环就是这样写。

这个 runner 的工作就是长期运行：

```text
等系统关闭
等自动 ticker
```

所以它需要一个无限循环。

### `select { ... }`

循环里面是：

```go
select {
case <-ctx.Done():
	return
case <-ticker.C:
	if err := s.System.Step(); err != nil {
		log.Printf("auto step failed: %v", err)
	}
}
```

`select` 是 Go 里专门用于等待多个 channel 的语法。

可以把它理解成：

```text
同时等好几个事件。
哪个事件先发生，就执行哪个 case。
```

这里同时等三个事件：

```text
1. ctx.Done()：系统要求退出
2. ticker.C：自动时间到了
```

### `case <-ctx.Done(): return`

第一种情况：

```go
case <-ctx.Done():
	return
```

`ctx.Done()` 是一个 channel。

当外部调用：

```go
cancel()
```

这个 channel 就会收到关闭信号。

这里的：

```go
<-ctx.Done()
```

表示等待退出信号。

收到后执行：

```go
return
```

也就是结束这个后台 goroutine。

在 `main.go` 里：

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

当 `main` 退出时，`defer cancel()` 会执行，从而通知后台 goroutine 退出。

### `case <-ticker.C`

第二种情况：

```go
case <-ticker.C:
	if err := s.System.Step(); err != nil {
		log.Printf("auto step failed: %v", err)
	}
```

这表示：

```text
ticker 到时间了，系统自动推进一个 tick。
```

也就是调用：

```go
s.System.Step()
```

如果 `Step()` 返回错误，runner 会记录日志：

```go
log.Printf("auto step failed: %v", err)
```

### 为什么不再支持手动 `POST /api/step`

系统现在采用更明确的语义：

```text
时间只由后端系统时钟推进。
客户端只能提交请求和读取状态。
```

也就是说，HTTP API 只负责：

```text
POST /api/request
  创建乘梯请求。

GET /api/state
  读取当前系统快照。
```

而不再暴露：

```text
POST /api/step
```

这样做减少了一层 API runner 的控制通道：

```text
不需要 stepCommand
不需要 stepCommands channel
不需要 RequestStep
不需要 done 回信 channel
```

系统的推进来源也更单一：

```text
time.Ticker -> StartAutoStep goroutine -> System.Step()
```

### 3. 每部电梯一个 goroutine

`internal/elevator/elevator_runner.go` 里：

```go
for elevatorIndex, commandChannel := range commands {
	go s.runElevator(ctx, elevatorIndex, commandChannel)
}
```

如果有 5 部电梯，就会启动 5 个这样的 goroutine。

每个电梯 goroutine 的主循环是：

```go
func (s *System) runElevator(ctx context.Context, elevatorIndex int, commands <-chan elevatorTickCommand) {
	for {
		select {
		case <-ctx.Done():
			return
		case command := <-commands:
			elevator, completedRequestIDs, err := stepElevatorState(...)
			result := elevatorTickResult{...}
			select {
			case command.done <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}
```

这个循环可以读成：

```text
只要系统没退出，我就一直等待命令。
收到 tick 命令后，我计算自己这部电梯的新状态。
算完后，我把结果发回去。
```

## 这次提交到底改了什么

这次提交涉及这些文件：

```text
internal/elevator/elevator_runner.go
internal/elevator/model.go
internal/elevator/system.go
internal/elevator/system_test.go
docs/record2.md
```

核心变化是：

```text
旧设计：
  每部电梯 goroutine 收到命令后，自己拿 System 锁，直接修改 s.Elevators[i]。

新设计：
  System.Step() 复制每部电梯状态，把副本发给对应 goroutine。
  电梯 goroutine 只计算副本，返回结果。
  System.Step() 统一合并结果。
```

这次改动的意义是：

```text
电梯 goroutine 不再直接写共享 System。
共享状态的写入集中在 System.Step() 的合并阶段。
channel 用来传递任务和结果。
mutex 用来保护真正共享的 System 状态。
```

这比“每个 goroutine 自己拿锁改 System”更清晰，因为状态所有权更明确。

## System 里的并发字段

`internal/elevator/model.go` 里：

```go
type System struct {
	mu     sync.Mutex
	stepMu sync.Mutex

	...

	elevatorCommands       []chan elevatorTickCommand
	elevatorRunnersDone    <-chan struct{}
	elevatorRunnersStarted bool

	nextRequestID int64
}
```

逐个解释。

### `mu sync.Mutex`

`mu` 保护 `System` 的共享状态。

主要包括：

```text
CurrentTick
Elevators
Requests
SchedulerName
scheduler
requestStore
nextRequestID
```

例如 `Snapshot()` 会这样：

```go
func (s *System) Snapshot() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return json.MarshalIndent(s, "", "  ")
}
```

这表示：

```text
JSON 编码 System 的时候，不允许其他 goroutine 同时修改 System。
```

### `stepMu sync.Mutex`

`stepMu` 只保护一件事：

```text
同一时间只能有一个完整的 Step() 正在执行。
```

为什么 `mu` 不够？

因为现在 `Step()` 中间会释放 `mu`：

```text
1. 拿 mu，调度并复制状态
2. 释放 mu
3. 把命令发给每部电梯 goroutine
4. 等待结果
5. 再拿 mu，合并结果
```

如果没有 `stepMu`，就可能出现：

```text
Step A 复制了第 10 tick 的状态，释放 mu，等待电梯结果
Step B 也进来复制第 10 tick 的状态，也开始等待电梯结果
两个 Step 交错，CurrentTick 和电梯状态就乱了
```

所以：

```go
func (s *System) Step() error {
	s.stepMu.Lock()
	defer s.stepMu.Unlock()
	...
}
```

保证：

```text
一个 tick 完整结束后，下一个 tick 才能开始。
```

### `elevatorCommands []chan elevatorTickCommand`

这是“发命令给电梯”的 channel 列表。

如果有 5 部电梯：

```text
elevatorCommands[0] -> 发命令给 1 号电梯 goroutine
elevatorCommands[1] -> 发命令给 2 号电梯 goroutine
elevatorCommands[2] -> 发命令给 3 号电梯 goroutine
elevatorCommands[3] -> 发命令给 4 号电梯 goroutine
elevatorCommands[4] -> 发命令给 5 号电梯 goroutine
```

类型是：

```go
[]chan elevatorTickCommand
```

读法是：

```text
一个切片，里面每个元素都是一个 channel，
每个 channel 传输 elevatorTickCommand。
```

### `elevatorRunnersDone <-chan struct{}`

这是一个只读 channel，用来表示：

```text
电梯 runner 所依赖的 context 已经结束。
```

如果系统正在关闭，而 `Step()` 还想往电梯 channel 里发送命令，就可能卡住。

所以发送命令时写成：

```go
select {
case commandChannel <- command:
case <-doneSignal:
	return fmt.Errorf("elevator runners stopped")
}
```

含义是：

```text
如果命令能发出去，就继续。
如果 runner 已经停止，就返回错误，不要死等。
```

### `elevatorRunnersStarted bool`

这个字段表示：

```text
是否已经启动了每部电梯的 goroutine。
```

`Step()` 会检查它：

```go
if !s.elevatorRunnersStarted {
	return fmt.Errorf("elevator runners are not started")
}

return s.stepWithElevatorRunners()
```

也就是说：

```text
启动了电梯 goroutine：推进一个系统 tick。
没启动电梯 goroutine：直接返回错误。
```

项目现在不再保留同步推进路径，避免出现两套 Step 语义。

## 启动流程

`cmd/server/main.go` 里：

```go
server := api.NewServer(system)
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
system.StartElevatorRunners(ctx)
server.StartAutoStep(ctx, defaultAutoStepInterval)
```

启动顺序是：

```text
1. 创建 System
2. 创建 API Server
3. 创建 context
4. 启动每部电梯的 goroutine
5. 启动自动 step runner
6. 启动 HTTP server
```

为什么先启动电梯 goroutine，再启动自动 step runner？

因为自动 step runner 一旦启动，就可能马上调用：

```go
System.Step()
```

而 `Step()` 在并发路径中需要把命令发给电梯 goroutine。

所以合理顺序是：

```text
先让电梯 goroutine 准备好接命令
再让自动 tick 开始推进系统
```

## 每个 tick 的完整执行过程

这是理解整个并发模型最关键的一节。

假设系统现在有 3 部电梯，并且自动 runner 触发了一次：

```go
s.System.Step()
```

完整过程如下。

### 第 1 步：进入 `System.Step()`

```go
func (s *System) Step() error {
	s.stepMu.Lock()
	defer s.stepMu.Unlock()

	s.mu.Lock()
	if !s.elevatorRunnersStarted {
		s.mu.Unlock()
		return fmt.Errorf("elevator runners are not started")
	}
	s.mu.Unlock()
	return s.stepWithElevatorRunners()
}
```

这里先拿：

```go
s.stepMu.Lock()
```

表示：

```text
现在开始执行一个完整 tick。
这个 tick 没结束前，别的 Step() 不能进来。
```

然后检查：

```go
if s.elevatorRunnersStarted
```

因为服务启动时已经调用了：

```go
system.StartElevatorRunners(ctx)
```

所以会进入：

```go
s.stepWithElevatorRunners()
```

### 第 2 步：调度器在锁内分配请求

`stepWithElevatorRunners()` 一开始：

```go
s.mu.Lock()
if len(s.Elevators) == 0 {
	...
}

if s.scheduler == nil {
	...
}

assigned := s.scheduler.Assign(s)
```

这里必须拿 `s.mu`。

因为调度器会读写：

```text
Requests
Elevators
Stops
AssignedTick
AssignedElevatorID
```

这些都属于共享状态。

调度器分配请求时，必须看到一个一致的系统状态，不能分配到一半被 API 或另一个 tick 改掉。

### 第 3 步：复制本 tick 的输入状态

调度完成后，代码会复制几个东西：

```go
commands := append([]chan elevatorTickCommand(nil), s.elevatorCommands...)
doneSignal := s.elevatorRunnersDone
```

`commands` 是每部电梯的命令 channel。

然后检查：

```go
if len(commands) != len(s.Elevators) {
	...
}
```

这保证：

```text
有几部电梯，就必须有几个电梯 goroutine 的 channel。
```

然后深拷贝电梯状态：

```go
elevators := make([]Elevator, len(s.Elevators))
for i := range s.Elevators {
	elevators[i] = cloneElevator(s.Elevators[i])
}
```

还会复制只读配置：

```go
currentTick := s.CurrentTick
ticksPerFloor := s.TicksPerFloor
doorBaseTicks := s.DoorBaseTicks
```

这一步的目的：

```text
把本 tick 需要的输入固定下来。
后面电梯 goroutine 计算时，不再直接读 System。
```

### 第 4 步：释放 `System.mu`

复制完本 tick 的输入后：

```go
s.mu.Unlock()
```

为什么这里释放锁？

因为接下来电梯 goroutine 只需要计算自己的 `Elevator` 副本，不需要读写共享 `System`。

释放锁后：

```text
API 可以读取 Snapshot
API 可以添加请求
电梯 goroutine 可以并行计算
```

但是注意：虽然释放了 `mu`，`stepMu` 还没释放。

所以：

```text
别的 API 操作可以进入 System。
但是另一个 Step() 不能开始。
```

这是两把锁分工的关键。

## 两把锁的分工

可以把两把锁理解成：

```text
mu：
  保护共享数据。

stepMu：
  保护 tick 边界。
```

`mu` 管的是“能不能读写 System 字段”。

`stepMu` 管的是“能不能开始另一次 Step”。

为什么不能只用一把锁？

如果 `Step()` 全程拿着 `mu`：

```text
调度
发送命令
等待所有电梯计算
合并结果
```

那么在等待电梯 goroutine 的过程中，`Snapshot()` 和 `AddRequest()` 也都被阻塞。

如果 `Step()` 不全程拿 `mu`，但又没有 `stepMu`：

```text
两个 Step 可能交错执行。
```

所以现在的结构是：

```text
stepMu 覆盖整个 Step，保证 tick 不交错。
mu 只覆盖真正读写共享 System 的片段。
```

## 第 5 步：给每部电梯发送 tick 命令

接下来：

```go
doneChannels := make([]chan elevatorTickResult, len(commands))
for i, commandChannel := range commands {
	done := make(chan elevatorTickResult, 1)
	doneChannels[i] = done
	command := elevatorTickCommand{
		elevator:      elevators[i],
		currentTick:   currentTick,
		ticksPerFloor: ticksPerFloor,
		doorBaseTicks: doorBaseTicks,
		done:          done,
	}
	select {
	case commandChannel <- command:
	case <-doneSignal:
		return fmt.Errorf("elevator runners stopped")
	}
}
```

这里有两个 channel：

```text
commandChannel
  Step 用它给某部电梯发送命令。

done
  这部电梯算完后，用它把结果发回来。
```

可以把一次命令理解成：

```text
发给 1 号电梯：
  这是你当前的状态副本。
  这是当前 tick。
  这是时间配置。
  算完之后，把结果放到 done 里。
```

每部电梯都会收到自己的 `elevatorTickCommand`。

## 第 6 步：电梯 goroutine 收到命令并计算

电梯 goroutine 一直在：

```go
select {
case <-ctx.Done():
	return
case command := <-commands:
	...
}
```

当命令来了，它会调用：

```go
elevator, completedRequestIDs, err := stepElevatorState(
	command.elevator,
	command.currentTick,
	command.ticksPerFloor,
	command.doorBaseTicks,
)
```

注意这里传入的是：

```go
command.elevator
```

这是电梯状态副本，不是 `s.Elevators[i]`。

所以这个计算不会直接修改共享 `System`。

### `stepElevatorState` 做什么

```go
func stepElevatorState(e Elevator, currentTick int, ticksPerFloor int, doorBaseTicks int) (Elevator, []int64, error)
```

它接收一个 `Elevator` 值。

在 Go 里，结构体按值传递时，会复制一份。

所以函数内部修改：

```go
e.Direction = DirectionUp
e.CurrentFloor++
```

修改的是局部副本。

函数最后返回新的 `Elevator`：

```go
return e, completedRequestIDs, nil
```

它处理几种情况。

### 情况 1：紧急停止

```go
if e.EmergencyStop {
	e.Direction = DirectionIdle
	return e, nil, nil
}
```

如果电梯紧急停止，本 tick 不移动。

### 情况 2：门开着

```go
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
```

如果门已经开着，本 tick 用来消耗停靠时间。

### 情况 3：没有停靠计划

```go
if len(e.Stops) == 0 {
	e.Direction = DirectionIdle
	return e, nil, nil
}
```

没有任务就保持 idle。

### 情况 4：已经到目标层

```go
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
```

如果当前楼层就是目标楼层：

```text
开门
设置开门剩余 tick
移除已经完成的 StopPlan
返回这次停靠完成的 Request ID
```

这里返回 `completedRequestIDs`，但不直接调用数据库。

原因是：

```text
数据库和 Requests map 属于 System 管理。
电梯 goroutine 只负责计算，不负责改全局状态。
```

### 情况 5：还没到目标层

```go
if e.CurrentFloor < targetFloor {
	e.Direction = DirectionUp
	moveOneTick(&e, 1, ticksPerFloor)
	return e, nil, nil
}

e.Direction = DirectionDown
moveOneTick(&e, -1, ticksPerFloor)
return e, nil, nil
```

如果目标层在上方，就向上移动一个行动 tick。

如果目标层在下方，就向下移动一个行动 tick。

真正控制“几 tick 移动一层”的是：

```go
moveOneTick(&e, floorDelta, ticksPerFloor)
```

## 第 7 步：电梯 goroutine 把结果发回 `Step`

计算完后：

```go
result := elevatorTickResult{
	elevator:            elevator,
	completedRequestIDs: completedRequestIDs,
	err:                 err,
}
```

然后：

```go
select {
case command.done <- result:
case <-ctx.Done():
	return
}
```

也就是说：

```text
如果系统还在运行，就把结果发回 Step。
如果系统正在关闭，就退出。
```

## 第 8 步：`Step` 收集所有电梯结果

`stepWithElevatorRunners()` 里：

```go
results := make([]elevatorTickResult, len(doneChannels))
for i, done := range doneChannels {
	select {
	case result := <-done:
		if result.err != nil {
			return result.err
		}
		results[i] = result
	case <-doneSignal:
		return fmt.Errorf("elevator runners stopped")
	}
}
```

这表示：

```text
等待每部电梯都返回结果。
任何一部电梯返回错误，整个 Step 返回错误。
如果 runner 停止，也返回错误。
```

为什么要等所有电梯？

因为一个 `Step()` 表示一个全局 tick。

在一个 tick 里，所有电梯都应该完成自己的行动单位，然后系统再统一进入下一个 tick。

所以：

```text
不能 1 号电梯算完就 CurrentTick++
必须所有电梯都算完，CurrentTick 才能 +1
```

## 第 9 步：合并结果

收集完结果后：

```go
s.mu.Lock()
defer s.mu.Unlock()

for i, result := range results {
	s.Elevators[i] = result.elevator
	for _, requestID := range result.completedRequestIDs {
		if err := s.completeRequest(requestID, currentTick); err != nil {
			return err
		}
	}
}
s.CurrentTick++
```

这里重新拿 `s.mu`，因为接下来要修改共享 `System`。

合并做了三件事：

```text
1. 把每部电梯的新状态写回 s.Elevators[i]
2. 把已完成的请求写入 SQLite，并从运行态 Requests 删除
3. CurrentTick++
```

这是一个很重要的边界：

```text
电梯 goroutine 只计算。
System.Step() 负责合并。
```

这样可以避免多个电梯 goroutine 同时写 `System`。

## 为什么需要 `cloneElevator`

`stepWithElevatorRunners()` 里：

```go
elevators := make([]Elevator, len(s.Elevators))
for i := range s.Elevators {
	elevators[i] = cloneElevator(s.Elevators[i])
}
```

看起来直接写：

```go
elevators[i] = s.Elevators[i]
```

似乎也可以。

但问题在于 `Elevator` 里有切片：

```go
Stops []StopPlan
```

`StopPlan` 里也有切片：

```go
RequestIDs []int64
```

Go 里的 slice 不是完整数组，它内部大致包含：

```text
指向底层数组的指针
长度
容量
```

所以如果只做普通赋值：

```go
copy := original
```

那么：

```text
copy.Stops 和 original.Stops 可能指向同一个底层数组。
copy.Stops[0].RequestIDs 和 original.Stops[0].RequestIDs 也可能指向同一个底层数组。
```

这会留下隐式共享。

在并发代码里，隐式共享很危险，因为你以为自己在改副本，实际上可能还在碰同一个底层数组。

所以这里写了：

```go
func cloneElevator(e Elevator) Elevator {
	if len(e.Stops) == 0 {
		return e
	}

	e.Stops = append([]StopPlan(nil), e.Stops...)
	for i := range e.Stops {
		e.Stops[i].RequestIDs = append([]int64(nil), e.Stops[i].RequestIDs...)
	}
	return e
}
```

它做了两层复制：

```text
复制 Stops 切片
复制每个 StopPlan 里的 RequestIDs 切片
```

这样电梯 goroutine 拿到的状态才是真正独立的本 tick 输入。

## 为什么 `stepElevatorState` 不直接写数据库

请求完成时，直觉上可能会想：

```text
电梯到站了，就直接在电梯 goroutine 里把 Request 标记完成并写数据库。
```

但这样会让电梯 goroutine 触碰：

```text
System.Requests
System.requestStore
SQLite connection
```

这会让状态所有权变复杂。

当前设计把职责拆开：

```text
电梯 goroutine：
  只判断“我这部电梯本 tick 完成了哪些 Request ID”。

System.Step()：
  在合并阶段调用 completeRequest，把请求写入 SQLite，并从 Requests 删除。
```

这样更容易保证：

```text
Requests map 只在 System.mu 保护下修改。
SQLite 写入也集中在 System 的状态合并阶段。
```

## 为什么删除同步路径 `stepLocked`

服务启动时一定会先调用：

```go
system.StartElevatorRunners(ctx)
```

然后自动时钟才会调用：

```go
server.StartAutoStep(ctx, defaultAutoStepInterval)
```

所以实际运行路径只有一条：

```text
StartElevatorRunners
  -> StartAutoStep
  -> System.Step()
  -> stepWithElevatorRunners()
```

如果保留 `stepLocked()`，读代码时会产生误解：

```text
好像系统同时支持“同步推进”和“并发推进”两种正式模型。
```

现在删除它后，`Step()` 的语义更单一：

```text
Step() 必须在电梯 goroutine 已启动后调用。
Step() 永远通过 stepWithElevatorRunners 推进系统。
```

测试也按真实服务启动顺序写：

```text
NewSystem
StartElevatorRunners
Step
```

## 自动时钟 runner 和电梯 runner 的关系

这里容易混淆，因为系统里有两层后台执行逻辑，但只有电梯层使用 channel。

### 自动时钟 runner

`internal/api/runner.go` 里的 `StartAutoStep` 负责：

```text
time.Ticker -> System.Step()
```

它不再接收手动 step 命令，也不再维护 `stepCommands` channel。

### 电梯 runner 的 channel

`internal/elevator/elevator_runner.go` 里的：

```go
elevatorCommands []chan elevatorTickCommand
```

负责的是：

```text
System.Step() -> 每部电梯 goroutine
```

### 两层合起来

一次自动 tick 的完整路径是：

```text
time.Ticker
  -> API runner goroutine
  -> System.Step()
  -> scheduler.Assign()
  -> elevatorCommands[0] / elevatorCommands[1] / ...
  -> 每部电梯 goroutine
  -> elevatorTickResult
	  -> System.Step() 合并结果
```

## 请求进入系统时会发生什么

`POST /api/request` 最终会调用：

```go
System.AddRequestSnapshot(...)
```

这个函数内部会拿 `System.mu`：

```go
func (s *System) AddRequestSnapshot(...) (Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	...
}
```

所以如果它和 `Step()` 同时发生，情况是安全的。

比如：

```text
Step 正在调度或合并：
  AddRequest 会等 System.mu。

Step 已经释放 mu，正在等电梯 goroutine 计算：
  AddRequest 可以进入，把新请求加入 Requests。
```

第二种情况里，新请求会不会被当前 tick 的调度器看到？

不会。

因为当前 tick 的调度阶段已经结束，并且已经复制了本 tick 的电梯输入。

这个新请求会在下一次 `Step()` 的调度阶段被看到。

这符合离散时间模拟的语义：

```text
一个 tick 开始时做调度。
tick 进行中来的请求，进入下一轮调度。
```

## Snapshot 为什么安全

`GET /api/state` 会调用：

```go
System.Snapshot()
```

内部：

```go
s.mu.Lock()
defer s.mu.Unlock()

return json.MarshalIndent(s, "", "  ")
```

所以 JSON 编码时，`System` 不会被同时修改。

如果 `Step()` 正在等电梯 goroutine 计算，此时它没有拿 `mu`。

那么 `Snapshot()` 可以读取当前已经提交的状态。

注意：

```text
它读到的是上一个完整 tick 合并后的状态。
不会读到“电梯 goroutine 算了一半”的中间状态。
```

因为电梯 goroutine 算的是副本，还没有写回 `System`。

这也是“副本计算、统一合并”的好处。

## 为什么 `CurrentTick` 在最后才加

`stepWithElevatorRunners()` 最后：

```go
s.CurrentTick++
```

含义是：

```text
当前 tick 的所有电梯行动已经完成。
系统进入下一个 tick。
```

如果在开头就加：

```text
调度、完成请求、写 CompletedTick 时语义会更绕。
```

现在的语义是：

```text
请求在 CurrentTick 这个时间点被创建或分配。
电梯在 CurrentTick 这个 tick 内行动。
如果本 tick 到站，CompletedTick 记录为 currentTick。
所有行动结束后，CurrentTick++。
```

## 这次测试验证了什么

`internal/elevator/system_test.go` 里新增或调整了：

```go
TestStepWithElevatorRunnersAdvancesEachElevator
```

测试核心是：

```go
system.Elevators[0].Stops = []StopPlan{
	{Floor: 2, Reason: StopReasonCabin, Direction: DirectionIdle},
}
system.Elevators[1].Stops = []StopPlan{
	{Floor: 3, Reason: StopReasonCabin, Direction: DirectionIdle},
}

ctx, cancel := context.WithCancel(context.Background())
defer cancel()
system.StartElevatorRunners(ctx)

if err := system.Step(); err != nil {
	t.Fatalf("Step returned error: %v", err)
}
```

它验证：

```text
1. 能启动每部电梯 goroutine。
2. Step 能给每部电梯发送 tick 命令。
3. 每部电梯能返回自己的新状态。
4. System 能把结果合并回 Elevators。
5. 一个全局 Step 只让 CurrentTick 增加 1。
```

测试里先设置 `Stops`，再启动 runner：

```text
先构造确定的初始状态
再启动并发执行单元
```

这样测试更清晰，也避免测试代码自己在 runner 启动后直接改共享状态。

## 如何自己验证并发安全

普通测试：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
```

race 检查：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

普通测试只能说明：

```text
功能结果大致正确。
```

race 检查更适合并发代码，因为它会尝试发现：

```text
一个 goroutine 正在写某个变量
另一个 goroutine 同时读或写同一个变量
中间没有锁或其他同步关系
```

并发代码修改后，应该优先跑：

```bash
go test -race ./...
```

## 如何阅读这部分代码

建议按这个顺序读。

### 第 1 步：看 `cmd/server/main.go`

重点看：

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
system.StartElevatorRunners(ctx)
server.StartAutoStep(ctx, defaultAutoStepInterval)
```

先理解系统启动了两类后台工作：

```text
每部电梯 runner
自动 step runner
```

### 第 2 步：看 `internal/api/runner.go`

重点看：

```go
StartAutoStep
```

理解：

```text
系统时钟什么时候调用 System.Step()
为什么 HTTP API 不再暴露手动 Step 入口
```

### 第 3 步：看 `internal/elevator/model.go`

重点看 `System` 结构体里的并发字段：

```go
mu
stepMu
elevatorCommands
elevatorRunnersDone
elevatorRunnersStarted
```

先不用急着看业务字段。

### 第 4 步：看 `internal/elevator/elevator_runner.go`

重点看：

```go
elevatorTickCommand
elevatorTickResult
StartElevatorRunners
runElevator
stepElevatorState
```

这一文件回答：

```text
每部电梯 goroutine 怎么启动？
它怎么收到命令？
它怎么计算？
它怎么返回结果？
```

### 第 5 步：看 `internal/elevator/system.go`

重点看：

```go
Step
stepWithElevatorRunners
cloneElevator
stepElevator
completeRequest
```

这一文件回答：

```text
System 怎么调度？
怎么把任务发给电梯？
怎么收集结果？
怎么把结果写回系统？
```

## 用一张文字图总结

```text
                         +----------------------+
                         |      HTTP 请求       |
                         | GET state / POST req |
                         +----------+-----------+
                                    |
                                    v
                         +----------------------+
                         |      API Server      |
                         +----------+-----------+
                                    |
                                    | auto ticker
                                    v
                    +-------------------+
                    | auto step runner  |
                    | goroutine         |
                    +---------+---------+
                              |
                              v
                    +-------------------+
                    |   System.Step()   |
                    +---------+---------+
                              |
              +---------------+----------------+
              |                                |
              v                                v
      lock System.mu                    scheduler.Assign()
      copy state                         update Requests/Stops
              |
              v
      unlock System.mu
              |
              v
      send elevatorTickCommand
              |
       +------+------+------+
       |      |      |      |
       v      v      v      v
    Elevator Elevator Elevator ...
    goroutine goroutine goroutine
       |      |      |
       v      v      v
  elevatorTickResult
       |      |      |
       +------+------+ 
              |
              v
      lock System.mu
      merge Elevators
      complete Requests
      CurrentTick++
      unlock System.mu
```

## 最核心的设计原则

这套并发模型可以浓缩成四句话：

```text
1. API 层不直接管理并发细节，System 自己保护自己的状态。
2. Step 是全局 tick 边界，同一时间只能有一个 Step。
3. 每部电梯用 goroutine 并行计算自己的下一状态。
4. goroutine 不直接写共享 System，结果由 System 统一合并。
```

如果你理解了这四句话，再回头读代码，很多看起来复杂的 channel、mutex、copy、result 都会变得更有因果关系。
