## 2026-05-10：SCAN / LOOK 是否必须实现 cost 函数

先给结论：

```text
SCAN 和 LOOK 算法本身不一定需要 cost 函数。
但在多电梯系统里，如果要决定“哪个电梯接这个请求”，cost 函数会很有用。
```

也就是说，`cost` 不是 SCAN / LOOK 的定义本身，而是多电梯调度时的一层评分机制。

### SCAN / LOOK 本身是什么

SCAN 和 LOOK 更像是在回答：

```text
一部电梯已经有了方向和停靠队列之后，它应该按什么顺序服务这些请求？
```

SCAN 的思想类似磁盘调度里的电梯算法：

```text
沿一个方向持续服务
到边界或没有更多请求后再反向
```

LOOK 和 SCAN 很像，但它不一定真的走到最顶层或最底层，而是：

```text
只走到当前方向上最后一个需要服务的请求处
然后反向
```

所以 SCAN / LOOK 的核心是：

```text
维护方向
维护停靠计划
顺路请求插入
什么时候反向
```

这些可以不用 cost 函数，也能写出一个完整算法。

### cost 函数解决的是什么问题

cost 函数更像是在回答另一个问题：

```text
当前有一个 pending request，应该分给哪部电梯？
```

例如现在有 5 部电梯：

```text
1 号电梯：在 3 楼，上行，Stops=[6, 9]
2 号电梯：在 10 楼，下行，Stops=[7]
3 号电梯：在 1 楼，空闲
4 号电梯：在 15 楼，上行，Stops=[18]
5 号电梯：在 5 楼，空闲
```

新请求是：

```text
6 楼，上行 hall request
```

这时候 SCAN / LOOK 可以告诉我们：

```text
某部电梯如果接了这个请求，应该把这个 StopPlan 插到哪里
```

但还需要决定：

```text
到底哪部电梯最适合接？
```

这就是 cost 函数的价值。

### cost 可以理解成“接这个请求的代价”

可以把 cost 函数想成：

```go
EstimateCost(system, elevator, request) int
```

它给“某部电梯接某个请求”打一个分数。

分数越低，说明越适合。

cost 可以考虑很多因素：

```text
电梯当前楼层到请求楼层的距离
请求是否顺路
电梯是否需要掉头
电梯当前已有多少停靠计划
请求已经等待了多久
电梯是否空闲
```

例如一个简单 cost 可以是：

```text
基础距离
+ 掉头惩罚
+ 已有 Stops 数量惩罚
- 等待时间补偿
```

这不是 SCAN / LOOK 的核心规则，而是多电梯分配时的评分规则。

### 没有 cost 函数也可以实现 SCAN / LOOK 吗

可以。

比如可以先写一个规则式 SCAN：

```text
1. 优先找能顺路接这个请求的电梯
2. 如果有多部顺路电梯，选距离最近的
3. 如果没有顺路电梯，找空闲电梯
4. 如果有多部空闲电梯，选距离最近的
5. 如果都不满足，暂时不分配
```

这个算法没有显式 `cost` 函数，但本质上仍然在做选择，只是规则写死在 if/else 里。

问题是：随着规则增加，if/else 会越来越难读。

### 有 cost 函数的好处

cost 函数的好处是把“评分逻辑”集中起来。

没有 cost 函数时，逻辑可能散落在：

```text
assignAlongTheWay()
assignToIdleElevator()
scanRequestPriority()
各种 if/else
```

有 cost 函数后，可以更统一：

```text
枚举所有 elevator + request 候选组合
对每个候选组合计算 cost
选择 cost 最低的候选
把请求插入对应电梯的 Stops
```

这样后续要改策略时，不一定要重写整个调度器，只需要调整 cost 公式。

### 什么时候必须上 cost

如果目标只是实现一个能解释的单电梯 SCAN / LOOK：

```text
不必须。
```

如果目标是多电梯、可比较、可扩展的调度系统：

```text
应该上。
```

因为多电梯调度不只是“电梯内部怎么排序”，还包括：

```text
请求分配给哪部电梯
是否抢占 / 重分配
等待太久的请求是否要补偿
是否避免某些请求长期饥饿
```

这些都很适合通过 cost 或 score 表达。

### 对当前项目的建议

当前可以直接重写 SCAN，甚至删除旧的 `SCANScheduler.go` 后重新实现。

但建议把问题拆成两层：

```text
第一层：LOOK / SCAN 的停靠顺序规则
  负责方向、顺路插入、反向、Stops 排序

第二层：多电梯候选评分
  负责决定哪个电梯接哪个请求
  这里可以使用 cost 函数
```

也就是说：

```text
LOOK / SCAN 不等于 cost
cost 是帮助 LOOK / SCAN 在多电梯场景下做选择的工具
```

所以如果后面重写 SCAN / LOOK，可以这样设计：

```text
先定义候选：
  elevator + request + 插入后的 StopPlan 位置

再定义 cost：
  这个候选会带来多少代价

最后执行：
  选择 cost 最低的候选并应用
```

### 一个比较清晰的实现方向

可以先不要急着写复杂公式，但要预留结构：

```go
type AssignmentCandidate struct {
	RequestID     int64
	ElevatorIndex int
	Cost          int
}
```

然后调度器做：

```text
1. 枚举所有 pending request
2. 枚举所有 elevator
3. 判断这个 elevator 是否可以接这个 request
4. 计算 cost
5. 选 cost 最低者
6. 插入 StopPlan
```

这样即使一开始 cost 很简单，例如：

```text
距离 + 掉头惩罚 + Stops 数量惩罚
```

后续也可以扩展：

```text
等待时间补偿
长期饥饿避免
不同算法对比
负载人数惩罚
```

### 最终回答

你的判断是合理的：当前 SCAN 可以重写，后面也可以加入 LOOK。

对于问题“SCAN / LOOK 需不需要 cost 函数”，答案是：

```text
算法定义本身不需要。
多电梯调度实现中很推荐需要。
```

如果只是写一个单电梯或规则式版本，可以不用 cost。

如果要做课程里更有说服力的多电梯调度，那么 cost 函数应该作为候选分配层存在，而不是混在 SCAN / LOOK 的方向规则里。

## 2026-05-10：实现调度 cost 函数骨架

本阶段完成 `docs/instructions-from-agent.md` 的 6.5.6：先把多电梯候选评分层搭出来。

这轮没有重写 `SCANScheduler.go`。原因是当前目标不是改 SCAN，而是先把 cost 函数作为独立能力建立起来。后续如果重写 SCAN 或新增 LOOK，可以复用这一层。

### 新增文件

新增：

```text
internal/elevator/cost.go
internal/elevator/cost_test.go
```

`cost.go` 负责回答一个问题：

```text
如果把某个 request 分配给某部 elevator，这个候选方案有多合适？
```

### 新增 `AssignmentScore`

```go
type AssignmentScore struct {
	DistanceCost     int
	TurnPenalty      int
	StopPenalty      int
	WaitCompensation int
	Total            int
}
```

它不是只返回一个黑盒数字，而是把 cost 拆成几个部分：

```text
DistanceCost
  电梯当前楼层到请求楼层的距离成本。

TurnPenalty
  如果电梯需要掉头，增加惩罚。

StopPenalty
  电梯已有 Stops 越多，说明它越忙，增加惩罚。

WaitCompensation
  请求等待越久，扣掉一部分成本，避免长期饥饿。

Total
  最终用于比较候选的总分。
```

当前公式是：

```text
Total = DistanceCost + TurnPenalty + StopPenalty - WaitCompensation
```

如果结果小于 0，会压到 0，避免出现负成本。

### 新增 `AssignmentCandidate`

```go
type AssignmentCandidate struct {
	RequestID     int64
	ElevatorIndex int
	Score         AssignmentScore
}
```

一个 candidate 表示：

```text
把 RequestID 这个请求
分配给 Elevators[ElevatorIndex] 这部电梯
对应的评分是 Score
```

这一步很重要，因为调度器真正要比较的不是单独的 request，也不是单独的 elevator，而是：

```text
request + elevator 的组合
```

### `EstimateCost`

```go
func EstimateCost(system *System, elevator Elevator, request Request) int
```

这是 6.5.6 里预留的简单入口。

它内部调用：

```go
EstimateAssignmentScore(system, elevator, request).Total
```

如果只关心最终分数，用 `EstimateCost`。

如果想看评分细节，用：

```go
EstimateAssignmentScore(...)
```

### 当前 cost 维度

当前已经实现的维度：

```text
距离
  floorDistance(elevator.CurrentFloor, request.Floor) * system.TicksPerFloor

掉头惩罚
  电梯当前 Direction 不是 idle，并且请求楼层不在当前方向上，就增加 turnPenaltyCost

已有停靠惩罚
  len(elevator.Stops) * stopPenaltyCost

等待时间补偿
  system.CurrentTick - request.CreatedTick
```

其中常量暂时是：

```go
turnPenaltyCost = 20
stopPenaltyCost = 10
```

这两个值不是最终调优结果，只是先让公式有清楚的结构。

### 为什么先让 `nearest-idle` 使用 cost

这轮没有大改 SCAN。

为了让 cost 层不是“写了但没人用”，把 `NearestIdleScheduler` 改成调用：

```go
BestIdleAssignmentCandidate(s, requestID)
```

也就是说，`nearest-idle` 现在不再手写一套距离比较逻辑，而是：

```text
1. 取最早 pending request
2. 枚举所有空闲电梯
3. 为每个候选计算 AssignmentScore
4. 选择 Total 最低的 candidate
5. 调用 assignRequestToElevator
```

这样 cost 函数已经进入真实调度路径。

### 为什么暂时只选空闲电梯

`BestIdleAssignmentCandidate` 当前只考虑：

```go
canAcceptRequest(elevator)
```

也就是：

```text
未紧急停止
没有 Stops
```

这是为了和 `nearest-idle` 的算法语义保持一致。

后续 LOOK / SCAN 要做的是另一类候选：

```text
运行中的电梯顺路追加请求
把 StopPlan 插入某个位置
计算插入后对总路线的影响
```

这比空闲电梯候选复杂，所以本轮先不混进去。

### 测试覆盖

新增测试覆盖：

```text
EstimateAssignmentScore 会返回距离、掉头、Stops、等待补偿明细
请求在反方向时会产生掉头惩罚
BestIdleAssignmentCandidate 会选择 cost 最低的空闲电梯
NearestIdleScheduler 会通过 cost 候选选择电梯
```

测试文件：

```text
internal/elevator/cost_test.go
```

### 后续如何增强

现在 cost 函数只是骨架，但结构已经能继续加维度：

```text
更精确估算已有 Stops 的等待时间
顺路插入请求的增量成本
LOOK / SCAN 的方向优先级
请求等待时间补偿的权重
避免某个请求长期 pending 的饥饿保护
电梯负载人数惩罚
```

后续如果重写 SCAN / LOOK，不应该把所有判断写成散落的 if/else，而应该尽量走：

```text
生成候选
计算 cost
选择最优候选
应用 StopPlan
```

### 本次验证

已运行：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
```

结果通过。

## 2026-05-10：引入最小并发模型

本阶段开始做 `docs/instructions-from-agent.md` 的第 7 部分：引入并发模型。

这次不是直接实现“每部电梯一个 goroutine”的完整版本，而是先做一个很小、能读懂、能验证的数据竞争安全版本：

```text
一个后台 goroutine 定时调用 System.Step()
API handler 仍然处理 HTTP 请求
后台 goroutine 和 API handler 通过同一把 mutex 保护 System
```

也就是说，系统现在第一次真的有了两个并发执行路径：

```text
路径 1：HTTP API
  用户提交请求、查看状态、手动 step

路径 2：后台自动 step
  定时推进电梯运行
```

### 先补 Go 并发最小概念

Go 并发里最常见的几个词：

```text
goroutine
  Go 的轻量并发执行单元。
  用 go f() 启动，表示让 f 在后台并发运行。

channel
  goroutine 之间传消息的管道。
  本轮暂时没有引入 channel，后续“每部电梯一个 goroutine”时再考虑。

mutex
  互斥锁，用来保护共享变量。
  同一时间只能有一个 goroutine 拿到这把锁。

select
  等待多个 channel 事件。
  本轮在后台循环里用它同时等待 ticker 和 ctx.Done()。

context
  用来控制 goroutine 退出。
  本轮用 context 让后台自动 step 循环可以停止。
```

如果和 POSIX 线程类比：

```text
goroutine 类似更轻量的线程
mutex 和 pthread_mutex 类似
channel 是 Go 更强调的消息传递工具
context 是 Go 服务端常用的取消机制
```

如果和 JavaScript 异步类比：

```text
JS async/await 多数时候还是单线程事件循环
Go goroutine 可以真的并发执行
所以 Go 里共享变量如果不加锁，会出现数据竞争
```

### 新增 `internal/api/runner.go`

新增文件：

```text
internal/api/runner.go
```

核心代码：

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
				s.mu.Lock()
				if err := s.System.Step(); err != nil {
					log.Printf("auto step failed: %v", err)
				}
				s.mu.Unlock()
			}
		}
	}()
}
```

逐段解释。

### `go func() { ... }()`

```go
go func() {
	...
}()
```

这表示启动一个新的 goroutine。

如果没有 `go`：

```go
func() {
	...
}()
```

这段代码会在当前 goroutine 里执行，`StartAutoStep` 会卡在无限循环里，后端无法继续启动 HTTP 服务。

加上 `go` 后：

```text
StartAutoStep 启动后台循环
然后立刻返回
main.go 继续注册路由并启动 HTTP server
```

### `time.NewTicker`

```go
ticker := time.NewTicker(interval)
defer ticker.Stop()
```

`Ticker` 表示一个固定间隔的时钟。

如果 interval 是：

```go
500 * time.Millisecond
```

那么大约每 500ms，`ticker.C` 这个 channel 会收到一次事件。

也就是说：

```text
每过一个 interval
后台 goroutine 就有机会调用一次 System.Step()
```

`defer ticker.Stop()` 表示函数退出时停止 ticker，避免资源泄漏。

### `select`

```go
select {
case <-ctx.Done():
	return
case <-ticker.C:
	...
}
```

`select` 用来等待多个 channel。

这里有两个事件：

```text
ctx.Done()
  外部要求后台 goroutine 停止

ticker.C
  到了下一次自动 step 时间
```

如果收到 `ctx.Done()`：

```go
return
```

后台 goroutine 结束。

如果收到 `ticker.C`：

```go
s.System.Step()
```

推进系统一个时间片。

### 为什么要加 mutex

`System` 是共享状态。

它里面有：

```text
CurrentTick
Elevators
Requests
Stops
scheduler
```

现在有多个地方会访问它：

```text
GET /api/state
  读取 System，生成 JSON 快照

POST /api/request
  修改 Requests，创建新请求

POST /api/step
  手动调用 Step，修改电梯状态

后台 goroutine
  定时调用 Step，修改电梯状态
```

如果没有锁，可能出现这种情况：

```text
后台 goroutine 正在修改 Elevators
同时 GET /api/state 正在读取 Elevators 并编码 JSON
```

这就是数据竞争。

Go 里数据竞争不是“偶尔结果不准”这么简单，它可能导致非常隐蔽的 bug。并发代码必须明确共享状态由谁保护。

### `Server` 里新增 mutex

`internal/api/handler.go` 里：

```go
type Server struct {
	System *elevator.System
	mu     sync.Mutex
}
```

`mu` 是一把互斥锁。

约定是：

```text
凡是 handler 或后台 goroutine 要读写 System，都先 Lock
读写结束后 Unlock
```

### `GET /api/state` 如何加锁

```go
s.mu.Lock()
data, err := s.System.Snapshot()
s.mu.Unlock()
```

`Snapshot()` 会读取整个 `System` 并编码成 JSON。

所以它必须在锁内完成，避免编码到一半时另一个 goroutine 修改了 `System`。

### `POST /api/request` 如何加锁

```go
s.mu.Lock()
createdRequest, err := s.System.AddRequest(...)
...
createdRequestSnapshot := *createdRequest
currentTick := s.System.CurrentTick
s.mu.Unlock()
```

`AddRequest()` 会修改：

```text
Requests map
nextRequestID
```

所以要加锁。

这里还有一个细节：

```go
createdRequestSnapshot := *createdRequest
```

`createdRequest` 是指向运行态请求的指针。如果解锁后再把这个指针交给 JSON 编码器，后台 goroutine 可能同时修改这个请求。

所以这里在锁内复制一份普通值：

```text
拿到 request 指针
复制成 request 快照
解锁
编码复制出来的快照
```

这样响应编码就不再读共享对象。

### `POST /api/step` 如何加锁

```go
s.mu.Lock()
if err := s.System.Step(); err != nil {
	s.mu.Unlock()
	...
}

data, err := s.System.Snapshot()
s.mu.Unlock()
```

手动 step 会修改 `System`，所以要加锁。

手动 step 后立刻返回 state，这个 `Snapshot()` 也在同一把锁里完成，保证返回的是 step 后的一致状态。

### 后台 goroutine 如何加锁

`runner.go` 里：

```go
s.mu.Lock()
if err := s.System.Step(); err != nil {
	log.Printf("auto step failed: %v", err)
}
s.mu.Unlock()
```

后台自动 step 和 API 手动 step 使用同一把锁。

所以同一时间不会出现：

```text
两个 Step 同时运行
Step 和 AddRequest 同时运行
Step 和 Snapshot 同时运行
```

这是当前最小并发版本的核心安全边界。

### `main.go` 如何启动后台 step

`cmd/server/main.go` 里新增：

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
server.StartAutoStep(ctx, defaultAutoStepInterval)
```

`context.Background()` 创建一个根 context。

`context.WithCancel(...)` 返回：

```text
ctx
  传给后台 goroutine，用于接收取消信号

cancel
  调用它就会关闭 ctx.Done()
```

当前 `main.go` 里用：

```go
defer cancel()
```

表示 main 退出时通知后台 goroutine 停止。

自动 step 间隔配置为：

```go
defaultAutoStepInterval = 500 * time.Millisecond
```

也就是说，服务启动后，大约每 500ms 后台自动推进一次 `System.Step()`。

### 这个版本带来的行为变化

之前：

```text
POST /api/request
  只创建请求

POST /api/step
  手动推进电梯
```

现在：

```text
POST /api/request
  创建请求

后台 goroutine
  自动每 500ms 调用 Step
```

所以前端或 curl 提交请求后，即使不手动调用 `/api/step`，电梯也会随着后台 ticker 自动移动。

`POST /api/step` 目前仍然保留，方便调试。

### 这还不是最终并发模型

课程要求里更理想的方向是：

```text
每部电梯作为独立执行单元
```

最终可能会设计成：

```text
每部电梯一个 goroutine
调度器通过 channel 给电梯发送 StopPlan
System 通过集中状态管理或事件循环汇总状态
API 只读受保护的快照
```

但本轮还没有做这些。

当前版本是：

```text
一个后台 goroutine 统一调用 Step
System 仍然是中心化状态
mutex 保护所有 API 和后台访问
```

这样做的目的不是最终形态，而是先把 Go 并发的最小可运行模型搭起来。

### 当前状态边界

本轮明确的边界：

```text
调度器仍然维护：
  请求分配
  Stops 插入
  AssignedTick
  AssignedElevatorID

stepElevator 仍然维护：
  单部电梯移动
  门状态
  到站完成请求

API 层 mutex 维护：
  谁可以在同一时刻访问 System

后台 goroutine 负责：
  定时调用 System.Step()
```

也就是说：

```text
业务逻辑仍在 elevator 包
并发保护暂时在 api.Server 外围
```

### 为什么 mutex 放在 API 层

这不是唯一设计。

也可以把 mutex 放进 `System` 里面，让 `System` 自己保护自己。

但当前先放在 `api.Server` 里，有两个原因：

```text
1. 对已有 elevator 包改动小
2. 同步版本的 System 测试不用全部重写
```

当前约定是：

```text
只要通过 HTTP 服务运行，就由 Server.mu 保护 System
```

后续如果每部电梯都有自己的 goroutine，可能需要重新设计，把并发控制进一步下沉到 elevator 包或单线程事件循环里。

### 新增测试

`internal/api/handler_test.go` 新增：

```go
TestStartAutoStepAdvancesSystemTick
```

它会：

```text
1. 创建测试 Server
2. 用很短的 interval 启动 StartAutoStep
3. 等待 CurrentTick 变大
4. 如果 200ms 内没有变化，就测试失败
```

这个测试证明：

```text
后台 goroutine 确实在推进 System.Step()
```

### race 检查

普通测试：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
```

并发数据竞争检查：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

结果都通过。



## 2026-05-10：把并发锁从 API 层下沉到 System 层

这次只做第 7 阶段并发重构的第一步：

```text
把 mutex 从 api.Server 下沉到 elevator.System
```

不做：

```text
channel
每部电梯一个 goroutine
每部电梯自己的运行循环
```

这些留给后续步骤。

### 为什么要下沉锁

上一版最小并发模型里，锁放在：

```go
type Server struct {
	System *elevator.System
	mu     sync.Mutex
}
```

也就是说，是 API 层负责保护 `System`。

这个方案能跑，也能通过 `go test -race`，但设计边界不够好。

原因是：`System` 不只会被 API 使用。

后续可能还有：

```text
后台 goroutine
每部电梯 goroutine
调度器 goroutine
测试代码
命令行调试工具
```

如果锁只放在 API 层，那么只有 HTTP handler 通过这把锁访问 `System` 才是安全的。其他地方如果直接调用：

```go
system.Step()
system.AddRequest(...)
system.Snapshot()
```

就可能绕过锁。

所以更合理的边界是：

```text
System 自己保护自己的内部状态
外部调用者不需要知道锁的存在
```

### 新的 System 结构

现在 `System` 里增加了：

```go
type System struct {
	mu sync.Mutex

	FloorCount int
	CurrentTick int
	...
}
```

`mu` 是小写字段：

```text
外部包不能访问
JSON 不会暴露
只有 elevator 包内部方法能使用
```

这表示锁是 `System` 的内部实现细节。

### 哪些方法现在自己加锁

当前这些公开方法会自己加锁：

```text
AddRequest
AddRequestSnapshot
SetScheduler
Close
Snapshot
Step
```

也就是说，外部代码调用这些方法时，不需要再手动加锁。

例如 API 层现在只需要：

```go
data, err := s.System.Snapshot()
```

而不是：

```go
s.mu.Lock()
data, err := s.System.Snapshot()
s.mu.Unlock()
```

### 为什么保留内部 locked 方法

`AddRequest` 现在大致是：

```go
func (s *System) AddRequest(...) (*Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.addRequestLocked(...)
}
```

真正创建请求的逻辑被拆到：

```go
func (s *System) addRequestLocked(...) (*Request, error)
```

命名里的 `Locked` 是一个约定：

```text
调用它之前，调用者必须已经持有 s.mu
```

这样做的好处是：如果后续某个公开方法需要在同一把锁里做多件事，就可以复用内部逻辑，避免重复加锁或死锁。

### 为什么新增 `AddRequestSnapshot`

`AddRequest` 仍然返回：

```go
*Request
```

这是指向 `System.Requests` 内部对象的指针。

在并发环境中，直接把这个指针交给 API 层继续使用并不理想。因为锁释放后，后台 goroutine 可能调用 `Step()`，修改这个请求的状态。

所以新增：

```go
func (s *System) AddRequestSnapshot(...) (Request, error)
```

它会：

```text
1. 加锁
2. 创建请求
3. 复制一份 Request 值
4. 解锁
5. 返回值拷贝
```

API 层拿到的是普通值，不是内部指针。

所以 `POST /api/request` 现在写成：

```go
createdRequest, err := s.System.AddRequestSnapshot(...)
```

而不是：

```go
createdRequest, err := s.System.AddRequest(...)
```

这能避免 API 在锁外编码一个仍可能被后台修改的内部指针。

### API 层现在更简单

`api.Server` 现在不再有：

```go
mu sync.Mutex
```

handler 也不再手动加锁。

例如 `GET /api/state`：

```go
data, err := s.System.Snapshot()
```

`POST /api/request`：

```go
createdRequest, err := s.System.AddRequestSnapshot(...)
```

`POST /api/step`：

```go
if err := s.System.Step(); err != nil {
	...
}
data, err := s.System.Snapshot()
```

API 层只表达 HTTP 逻辑：

```text
解析请求
调用 System
返回响应
```

并发保护交给 `System` 自己。

### 后台 runner 也更简单

`internal/api/runner.go` 里之前是：

```go
s.mu.Lock()
if err := s.System.Step(); err != nil {
	...
}
s.mu.Unlock()
```

现在改成：

```go
if err := s.System.Step(); err != nil {
	log.Printf("auto step failed: %v", err)
}
```

因为 `System.Step()` 内部已经会加锁。

### 当前锁保护的状态

这把锁保护的是整个 `System` 的共享状态：

```text
CurrentTick
Elevators
Requests
Stops
scheduler
nextRequestID
requestStore 写入时的调用顺序
```

这里记录的是当时“锁下沉到 System 层”阶段的粗粒度锁设计：

```text
一次只有一个 goroutine 能执行 AddRequest / Step / Snapshot 等 System 操作
```

后面的每电梯 goroutine 实现已经把电梯运行计算从这把粗粒度锁里拆出来：

```text
调度和状态合并仍然由 System.mu 保护
每部电梯的 tick 计算通过 channel 交给各自 goroutine 并行执行
API Snapshot 仍然在 System.mu 下读取一致状态
```

### 这个设计和后续 channel / 每电梯 goroutine 的关系

这一步不是最终形态。

它只是把并发安全边界先放到正确位置：

```text
System 是共享状态，所以 System 自己负责保护共享状态。
```

下一步如果引入 channel，可以让：

```text
API 把请求发送到 channel
后台调度循环从 channel 读取请求
System 仍然作为受保护的状态中心
```

再下一步如果每部电梯一个 goroutine，可能会继续演进成：

```text
每部电梯 goroutine 维护自己的运行循环
调度器通过 channel 下发 StopPlan
System 维护只读快照或集中事件日志
```

但这些都建立在当前这一步之上。

### 本次验证

普通测试：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
```

race 检查：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

结果都通过。

### 为什么有 `AddRequest` 和 `addRequestLocked`

这里看起来确实有点绕：

```go
func (s *System) AddRequest(...) (*Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.addRequestLocked(...)
}

func (s *System) addRequestLocked(...) (*Request, error) {
	...
}
```

如果只有 `AddRequest()` 一个函数，当然可以直接写成：

```go
func (s *System) AddRequest(...) (*Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 直接写创建请求的完整逻辑
}
```

这没有问题。

当前之所以拆出 `addRequestLocked()`，主要是因为现在有两个公开方法都需要“创建请求”这段逻辑：

```go
AddRequest(...)
AddRequestSnapshot(...)
```

它们的区别是：

```text
AddRequest
  返回内部 *Request 指针
  主要给测试或 elevator 包内部逻辑使用

AddRequestSnapshot
  返回 Request 值拷贝
  适合 API 层返回给客户端
```

但它们创建请求的核心逻辑完全一样：

```text
检查楼层
检查方向
检查请求类型
生成 ID
设置 CreatedTick
写入 Requests map
nextRequestID++
```

所以把共同逻辑放进：

```go
addRequestLocked(...)
```

可以避免复制两份创建请求代码。

#### 为什么不让 `AddRequestSnapshot` 直接调用 `AddRequest`

可能会想到这样写：

```go
func (s *System) AddRequestSnapshot(...) (Request, error) {
	request, err := s.AddRequest(...)
	if err != nil {
		return Request{}, err
	}
	return *request, nil
}
```

这个版本看起来更简单，但有一个并发细节：`AddRequest()` 自己会加锁并解锁。

流程会变成：

```text
AddRequestSnapshot 调用 AddRequest
AddRequest 加锁
AddRequest 创建请求
AddRequest 解锁
AddRequestSnapshot 拿到 *Request
AddRequestSnapshot 再复制 Request
```

问题在于：

```text
AddRequest 解锁之后
AddRequestSnapshot 复制之前
后台 goroutine 可能调用 Step()
Step 可能修改这个 Request
```

这样 `AddRequestSnapshot` 返回的就不一定是“刚创建时”的快照。

所以现在写成：

```go
func (s *System) AddRequestSnapshot(...) (Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, err := s.addRequestLocked(...)
	if err != nil {
		return Request{}, err
	}
	return *request, nil
}
```

这样创建和复制都发生在同一把锁里：

```text
加锁
创建请求
复制快照
解锁
```

API 层拿到的是稳定的值拷贝。

#### 为什么不能在持锁时再调用会加锁的方法

还有一个重要原因：Go 的 `sync.Mutex` 不是可重入锁。

也就是说，如果同一个 goroutine 已经拿到了 `s.mu`，然后又调用另一个也会 `s.mu.Lock()` 的方法，就会把自己卡死。

例如如果写成：

```go
func (s *System) AddRequestSnapshot(...) (Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, err := s.AddRequest(...)
	...
}
```

会发生：

```text
AddRequestSnapshot 拿到 s.mu
AddRequestSnapshot 调用 AddRequest
AddRequest 也想拿 s.mu
但 s.mu 已经被 AddRequestSnapshot 持有
于是永远等下去
```

这就是死锁。

所以并发代码里常见一种命名约定：

```text
PublicMethod()
  公开方法，负责加锁

privateMethodLocked()
  内部方法，假定调用者已经持有锁
```

`Locked` 后缀就是提醒：

```text
调用这个函数之前，必须已经拿到对应的锁。
```

#### 这个设计是不是必须的

不是必须。

如果觉得现在学习阶段可读性更重要，也可以把设计改得更简单：

```text
只保留 AddRequestSnapshot
API 和测试都使用它
删除 AddRequest
或者让 AddRequest 直接返回值拷贝
```

但在当前代码里，`AddRequest + addRequestLocked + AddRequestSnapshot` 的理由是：

```text
避免重复创建请求逻辑
保证快照复制发生在锁内
避免持锁时调用另一个会加锁的方法导致死锁
保留返回 *Request 的旧测试/内部使用方式
```

所以它不是为了炫技，也不是一定要这样写；它是锁下沉之后，为了同时满足“复用逻辑”和“并发安全快照”产生的折中。

`go test -race` 会启用 Go 的 race detector。它可以发现常见的数据竞争，例如：

```text
一个 goroutine 在写变量
另一个 goroutine 同时读同一个变量
中间没有锁或其他同步机制
```

这一步对于并发代码很重要。普通 `go test` 通过，不代表没有数据竞争；`go test -race` 更适合检查这类问题。

## 2026-05-10：使用 channel 传递 step 控制信号

这次完成第 7 阶段并发路线的第二步：

```text
使用 channel 传递请求或控制信号
```

### Go channel 基础机制

channel 可以先理解成：

```text
goroutine 之间传递消息的管道
```

如果 mutex 的思路是：

```text
多个 goroutine 共享同一份变量
访问变量前先抢锁
```

那么 channel 的思路更像是：

```text
一个 goroutine 把消息发出去
另一个 goroutine 收到消息后再处理
```

Go 里创建 channel 的语法是：

```go
ch := make(chan int)
```

这表示创建一个传递 `int` 的 channel。

如果要传递自定义结构体，也可以写：

```go
type command struct {
	name string
}

ch := make(chan command)
```

本项目里这次用的就是类似这种方式：

```go
stepCommands chan stepCommand
```

意思是：

```text
stepCommands 这个 channel 只能传 stepCommand 类型的消息
```

#### 发送和接收

发送消息：

```go
ch <- value
```

意思是：

```text
把 value 发送到 ch 这个 channel 里
```

接收消息：

```go
value := <-ch
```

意思是：

```text
从 ch 这个 channel 里取出一个 value
```

箭头 `<-` 的方向可以帮助记忆：

```text
ch <- value
  value 流向 channel

value := <-ch
  value 从 channel 流出
```

#### 无缓冲 channel 会阻塞

默认创建的是无缓冲 channel：

```go
ch := make(chan int)
```

无缓冲 channel 的特点是：

```text
发送方和接收方必须同时准备好
```

如果只有发送方：

```go
ch <- 1
```

但没有任何 goroutine 正在接收：

```go
value := <-ch
```

发送方就会卡住。

反过来，如果只有接收方在等，但没人发送，接收方也会卡住。

这和普通数组或队列不一样。无缓冲 channel 更像一次“交接”：

```text
发送方把东西递出去
接收方必须同时伸手接
交接完成后双方才继续运行
```

#### 有缓冲 channel

也可以创建有缓冲 channel：

```go
ch := make(chan int, 1)
```

第二个参数 `1` 表示容量。

有缓冲 channel 可以先暂存消息。

例如容量为 1 时：

```text
channel 为空时，发送一个值通常不会阻塞
channel 满了以后，继续发送才会阻塞
```

本轮代码里：

```go
done := make(chan error, 1)
```

就是一个容量为 1 的 channel。

它的作用是：runner 执行完 step 后，把 `error` 或 `nil` 放进去。即使发送方因为 HTTP 请求取消已经返回了，runner 往 `done` 里放一个结果也不容易卡住。

#### channel 常用于“请求-响应”

channel 不只能传简单数据，也可以传一个带“回信 channel”的结构体。

例如：

```go
type stepCommand struct {
	done chan error
}
```

这表示：

```text
发送方发出 stepCommand
stepCommand 里带着一个 done channel
接收方执行完任务后，把结果写回 done
```

整体像这样：

```text
发送方：
  1. 创建 done channel
  2. 把 done 放进 command
  3. 把 command 发给 runner
  4. 等待 done 返回结果

runner：
  1. 从 command channel 收到 command
  2. 执行 System.Step()
  3. 把结果发送到 command.done
```

这就是本轮 `POST /api/step` 使用的模式。

#### select 是等待多个 channel

如果一个 goroutine 只等一个 channel，可以写：

```go
command := <-s.stepCommands
```

但 runner 同时要等三种事件：

```text
ctx.Done()       退出信号
ticker.C         自动 step 时间到了
stepCommands     手动 step 命令来了
```

所以使用：

```go
select {
case <-ctx.Done():
	return
case <-ticker.C:
	...
case command := <-s.stepCommands:
	...
}
```

`select` 的含义是：

```text
等待多个 channel
哪个 channel 先准备好，就执行哪个 case
```

这就是为什么后面的 runner 可以同时支持：

```text
自动 ticker
手动 step 命令
退出信号
```

#### channel 和 mutex 不是互相替代

这里容易误解：用了 channel，是不是就不需要 mutex？

不一定。

当前项目里：

```text
channel
  用来传递控制信号，例如“请执行一次 Step”

mutex
  仍然在 System 内部保护共享状态
```

也就是说：

```text
channel 负责通信
mutex 负责保护共享数据
```

后续如果改成更彻底的单线程事件循环，可能减少 mutex 的使用。但当前阶段两者同时存在是合理的。

本轮选择先传递“控制信号”，也就是：

```text
手动 step 请求
```

暂时不通过 channel 传递乘梯请求，也不做每部电梯一个 goroutine。

### 为什么先传控制信号

当前项目里已经有后台 goroutine 自动调用：

```go
System.Step()
```

同时也保留了手动 API：

```text
POST /api/step
```

如果两边都直接调用 `System.Step()`，虽然 `System` 内部已经有 mutex，数据竞争问题可以避免，但控制流仍然比较分散：

```text
后台 ticker 直接 Step
HTTP handler 也直接 Step
```

这次引入 channel 后，手动 step 会先变成一个控制信号，发送给后台 runner：

```text
POST /api/step
  -> RequestStep(...)
  -> stepCommands channel
  -> runner goroutine
  -> System.Step()
```

这样做的意义是：先建立“通过 channel 给后台循环发送控制命令”的模式。

### `stepCommand`

`internal/api/runner.go` 新增：

```go
type stepCommand struct {
	done chan error
}
```

`stepCommand` 表示一次“请执行 Step”的控制信号。

里面的：

```go
done chan error
```

用于把执行结果传回发送方。

也就是说，这不是一个单向通知，而是一次请求-响应：

```text
发送方：请你 step 一次
runner：执行 System.Step()
runner：把 error 或 nil 发回 done
发送方：收到结果后继续返回 HTTP 响应
```

### `Server` 里新增 channel

`internal/api/handler.go` 里：

```go
type Server struct {
	System            *elevator.System
	stepCommands      chan stepCommand
	stepRunnerStarted bool
}
```

字段含义：

```text
System
  电梯系统核心状态。

stepCommands
  发送 step 控制信号的 channel。

stepRunnerStarted
  标记后台 runner 是否已经启动。
```

### 为什么新增 `NewServer`

新增：

```go
func NewServer(system *elevator.System) *Server {
	return &Server{
		System: system,
	}
}
```

现在 `main.go` 使用：

```go
server := api.NewServer(system)
```

而不是：

```go
server := &api.Server{System: system}
```

这样后续如果 `Server` 需要初始化更多内部字段，可以集中放在 `NewServer` 里，不让 `main.go` 知道太多内部细节。

当前 `stepCommands` 仍然在 `StartAutoStep` 里初始化，这样测试中不启动 runner 时也可以直接走同步 fallback。

### `StartAutoStep` 的新结构

现在 `StartAutoStep` 做两件事：

```go
s.ensureStepCommands()
s.stepRunnerStarted = true
```

然后启动后台 goroutine：

```go
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
```

这个 `select` 现在等待三个事件：

```text
ctx.Done()
  外部要求 runner 停止。

ticker.C
  到了自动 step 的时间。

s.stepCommands
  收到一次手动 step 控制信号。
```

也就是说，后台 runner 现在不只是定时器，也接收外部命令。

### 自动 step 和手动 step 的区别

自动 step：

```go
case <-ticker.C:
	s.runStepCommand(nil)
```

传入 `nil`，表示不需要把结果传给某个等待者。

手动 step：

```go
case command := <-s.stepCommands:
	s.runStepCommand(command.done)
```

传入 `command.done`，表示执行完以后要把结果发回去。

### `runStepCommand`

```go
func (s *Server) runStepCommand(done chan<- error) {
	err := s.System.Step()
	if err != nil {
		log.Printf("step failed: %v", err)
	}
	if done != nil {
		done <- err
	}
}
```

这个函数统一执行一次 step。

如果是自动 ticker 触发：

```go
s.runStepCommand(nil)
```

执行完只记录日志，不通知别人。

如果是手动 API 触发：

```go
s.runStepCommand(command.done)
```

执行完会通过 `done` channel 把结果传回发送方。

### `RequestStep`

`POST /api/step` 现在不直接调用 `System.Step()`，而是调用：

```go
func (s *Server) RequestStep(ctx context.Context) error
```

它的核心流程：

```go
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
```

第一段 `select`：

```text
尝试把 command 发给 runner
如果 HTTP 请求被取消，就返回 ctx.Err()
```

第二段 `select`：

```text
等待 runner 执行完 Step 并返回结果
如果 HTTP 请求中途取消，也返回 ctx.Err()
```

这里使用 `ctx` 的原因是：HTTP 请求本身可能断开或超时。后端不应该在客户端已经离开后还永远等着。

### 为什么 `done` 是 buffered channel

这里写的是：

```go
done := make(chan error, 1)
```

容量是 1。

这样 runner 执行完以后：

```go
done <- err
```

不会因为发送方刚好已经因为 `ctx.Done()` 返回而永久阻塞。

这是一个小的防御性设计。

### `handleStep` 现在怎么走

`internal/api/handler.go` 里：

```go
if err := s.RequestStep(r.Context()); err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}

data, err := s.System.Snapshot()
```

也就是说：

```text
HTTP handler
  不直接 Step
  通过 RequestStep 发 channel 控制信号
  等 runner 执行完
  再读取 Snapshot 返回
```

### 为什么保留 fallback

`RequestStep` 里有：

```go
if !s.stepRunnerStarted {
	return s.System.Step()
}
```

这是为了测试和渐进式重构。

有些测试只创建 `Server`，但没有启动 `StartAutoStep`。如果没有 fallback，`RequestStep` 会往一个没人接收的 channel 发送命令，然后卡住。

所以当前语义是：

```text
runner 已启动：
  通过 channel 发 step 控制信号

runner 未启动：
  直接同步调用 System.Step()
```

正式服务里，`main.go` 会启动：

```go
server.StartAutoStep(ctx, defaultAutoStepInterval)
```

所以正式运行时会走 channel。

### 当前并发边界

现在的边界是：

```text
System
  仍然用 mutex 保护内部状态。

runner goroutine
  负责接收 ticker 和 stepCommands。

POST /api/step
  通过 channel 请求 runner 执行一次 Step。

POST /api/request
  仍然直接调用 System.AddRequestSnapshot。
```

也就是说，这一步只把“控制信号”接入 channel。

乘梯请求本身还没有通过 channel 进入后台循环。后续如果继续推进，可以把：

```text
POST /api/request
```

也改成发送：

```text
AddRequestCommand
```

给后台事件循环处理。

### 这和每部电梯 goroutine 的关系

这一步仍然不是最终模型。

它只是先建立了：

```text
API -> channel -> runner goroutine -> System
```

下一步设计每部电梯 goroutine 时，可以继续扩展成：

```text
API -> request channel -> scheduler/event loop
scheduler -> elevator control channel -> elevator goroutine
elevator goroutine -> state update channel -> System snapshot
```

但如果一开始就做这一整套，会很难阅读和调试。

### 新增测试

新增测试：

```go
TestRequestStepUsesControlChannel
```

测试流程：

```text
1. 创建 Server
2. 启动 StartAutoStep，但 interval 设置成很长
3. 调用 RequestStep
4. 检查 CurrentTick 变成 1
```

interval 设置成 `time.Hour` 是为了避免自动 ticker 干扰测试，让 tick 增加只来自手动 channel 控制信号。

### 本次验证

普通测试：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
```

race 检查：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

结果都通过。

## 2026-05-11：为每部电梯启动运行 goroutine

这次完成第 7 阶段并发路线的第三步：

```text
设计每部电梯的运行循环，并逐步给每部电梯分配 goroutine
```

这次把电梯运行部分改成更规范的 Go 并发结构：

```text
每部电梯都有自己的 goroutine
每部电梯 goroutine 通过 channel 接收 tick 控制信号
System.Step() 负责调度、分发 tick、收集结果、合并状态
电梯 goroutine 不直接写共享 System，而是返回自己的计算结果
```

这个设计遵守 Go 并发里很重要的一条原则：

```text
不要让多个 goroutine 同时随便写同一份共享内存。
能用 channel 传递任务和结果，就用 channel 表达所有权转移。
必须共享的全局状态，再用 mutex 保护。
```

### 新增 `internal/elevator/elevator_runner.go`

新增文件：

```text
internal/elevator/elevator_runner.go
```

这个文件负责和“每部电梯 goroutine”有关的逻辑。

核心结构：

```go
type elevatorTickCommand struct {
	elevator      Elevator
	currentTick   int
	ticksPerFloor int
	doorBaseTicks int
	done          chan elevatorTickResult
}

type elevatorTickResult struct {
	elevator            Elevator
	completedRequestIDs []int64
	err                 error
}
```

它表示发给某一部电梯的一次 tick 命令。

这里不是让电梯 goroutine 直接去改：

```go
s.Elevators[i]
```

而是把这部电梯的状态副本放进命令里：

```go
elevator Elevator
```

电梯 goroutine 拿到副本后独立计算，最后通过 `done` 返回 `elevatorTickResult`。

完整数据流是：

```text
System.Step() 复制每部电梯的状态
System.Step() 把状态副本和只读配置发给对应电梯 goroutine
电梯 goroutine 独立计算本 tick 后的新状态
电梯 goroutine 把新 Elevator 和完成的 Request ID 发回
System.Step() 统一合并所有结果
System.Step() 给 CurrentTick +1
```

### `System` 新增字段

`System` 里新增：

```go
elevatorCommands       []chan elevatorTickCommand
elevatorRunnersDone    <-chan struct{}
elevatorRunnersStarted bool
```

含义：

```text
elevatorCommands
  每部电梯一个 channel。
  elevatorCommands[i] 用来给第 i 部电梯发送 tick 命令。

elevatorRunnersDone
  当外部 context 被取消时关闭。
  Step() 可以用它判断电梯 goroutine 是否已经停止，避免向没人接收的 channel 发送命令。

elevatorRunnersStarted
  标记每部电梯的 goroutine 是否已经启动。
```

这些字段没有 JSON tag，也不是导出字段。

它们属于并发运行时内部实现，不应该出现在 `/api/state` 里。

此外，`System` 里现在有两把锁：

```go
mu     sync.Mutex
stepMu sync.Mutex
```

它们分工不同：

```text
mu
  保护 System 里的共享状态，例如 Elevators、Requests、CurrentTick。

stepMu
  保护一次完整的 System.Step() 调用。
```

为什么还需要 `stepMu`？

因为在电梯 goroutine 版本里，`Step()` 会：

```text
1. 调度器分配请求
2. 解锁
3. 发送每部电梯的状态副本
4. 等待所有电梯返回计算结果
5. 重新加锁，合并结果
6. CurrentTick +1
```

中间不持有 `mu`，所以电梯 goroutine 可以并行计算。

但如果没有 `stepMu`，两个外部 goroutine 可能同时调用 `Step()`，导致两个全局 tick 交错执行。

所以 `stepMu` 保证：

```text
一次完整 Step 结束之后，下一次 Step 才能开始。
```

`mu` 保护状态，`stepMu` 保护 tick 边界。

### `StartElevatorRunners`

新增公开方法：

```go
func (s *System) StartElevatorRunners(ctx context.Context)
```

它的流程是：

```text
1. 给 System 加锁
2. 如果已经启动过，直接返回
3. 为每部电梯创建一个 channel
4. 保存到 s.elevatorCommands
5. 建立 elevatorRunnersDone
6. 标记 elevatorRunnersStarted = true
7. 解锁
8. 为每部电梯启动一个 goroutine
```

代码结构大致是：

```go
commands := make([]chan elevatorTickCommand, len(s.Elevators))
for i := range commands {
	commands[i] = make(chan elevatorTickCommand, 1)
}
s.elevatorCommands = commands
s.elevatorRunnersStarted = true
```

这里的 channel 是容量为 1 的 buffered channel：

```go
make(chan elevatorTickCommand, 1)
```

因为每部电梯在同一个全局 tick 内只会收到一个命令，容量 1 可以避免发送方和接收方在调度瞬间产生不必要的阻塞。

然后：

```go
for elevatorIndex, commandChannel := range commands {
	go s.runElevator(ctx, elevatorIndex, commandChannel)
}
```

这里的：

```go
go s.runElevator(...)
```

就是给每部电梯启动独立 goroutine。

### 单部电梯的运行循环

每个电梯 goroutine 执行：

```go
func (s *System) runElevator(ctx context.Context, elevatorIndex int, commands <-chan elevatorTickCommand) {
	for {
		select {
		case <-ctx.Done():
			return
		case command := <-commands:
			elevator, completedRequestIDs, err := stepElevatorState(...)
			command.done <- elevatorTickResult{...}
		}
	}
}
```

它一直等待两类事件：

```text
ctx.Done()
  系统要求退出，goroutine 结束。

commands
  收到一次 tick 命令，推进自己负责的电梯。
```

这里的关键不是 `elevatorIndex` 本身，而是所有权边界：

```text
System.Step() 拥有 System 共享状态
电梯 goroutine 拥有本次命令里的 Elevator 副本
电梯 goroutine 只返回结果，不直接写 System
```

### `stepElevatorState`

单部电梯的运动逻辑被抽成纯状态函数：

```go
func stepElevatorState(
	e Elevator,
	currentTick int,
	ticksPerFloor int,
	doorBaseTicks int,
) (Elevator, []int64, error)
```

它的输入是：

```text
一部电梯的状态副本
当前 tick
移动一层需要几个 tick
开门停靠需要几个 tick
```

它的输出是：

```text
更新后的电梯状态
本 tick 完成的请求 ID 列表
错误
```

这个函数不读写 `System`，因此适合放进 goroutine 并行执行。

### `System.Step()` 的变化

之前 `Step()` 永远走同步逻辑：

```text
调度器分配请求
for 循环推进所有电梯
CurrentTick++
```

现在 `Step()` 会先检查电梯 goroutine 是否已经启动：

```go
if !s.elevatorRunnersStarted {
	return fmt.Errorf("elevator runners are not started")
}

return s.stepWithElevatorRunners()
```

也就是说，`Step()` 只有一条正式推进路径：`stepWithElevatorRunners()`。

### `stepWithElevatorRunners`

这是每部电梯 goroutine 的主路径。

流程是：

```text
1. System 加锁
2. 检查电梯和调度器是否合法
3. 调度器 Assign
4. 拷贝 elevatorCommands
5. 深拷贝每部 Elevator，形成本 tick 的计算输入
6. 记录 currentTick、ticksPerFloor、doorBaseTicks 等只读配置
7. System 解锁
8. 给每个电梯 channel 发送 tick 命令
9. 等待每个电梯 goroutine 返回 result
10. System 重新加锁
11. 合并每部电梯的新状态
12. 把完成的 Request 写入 SQLite，并从运行态 Requests 删除
13. CurrentTick++
```

这里特意做了深拷贝：

```go
elevators[i] = cloneElevator(s.Elevators[i])
```

原因是 `Elevator` 里有切片字段：

```go
Stops []StopPlan
RequestIDs []int64
```

Go 的切片不是完整数组本身，而是指向底层数组的视图。

如果只是普通赋值：

```go
copy := s.Elevators[i]
```

那么 `copy.Stops` 仍然可能和 `s.Elevators[i].Stops` 指向同一个底层数组。

并发程序里不要留下这种隐式共享，所以这里用 `cloneElevator` 明确复制切片。

为什么调度器仍然在 `System` 锁里运行？

因为调度器会读写：

```text
Requests
Elevators
Stops
AssignedTick
AssignedElevatorID
```

这些都是共享状态。

所以调度阶段由 `System` 持有锁完成，保证调度看到的是同一个 tick 边界上的一致状态。

电梯运行阶段则不再持有 `System` 锁，而是让各电梯 goroutine 并行计算自己的结果。

为什么推进每部电梯时不在 `Step()` 里直接 for 循环？

因为现在要让每部电梯拥有自己的 goroutine：

```text
Step 发送命令
电梯 goroutine 执行
Step 等待结果
```

这就是从“一个函数循环所有电梯”变成“每部电梯独立执行单元”。

### 为什么不是每部电梯 goroutine 直接写 `System`

如果每个电梯 goroutine 直接同时写：

```go
s.Elevators[i]
```

同时 API 又在执行：

```go
GET /api/state
```

那么 `/api/state` 可能正在 JSON 编码 `s.Elevators`，电梯 goroutine 正在写 `s.Elevators[i]`。

这就是典型数据竞争。

现在的结构是：

```text
电梯 goroutine 并行计算
System.Step() 串行合并
API Snapshot 在 System.mu 下读取
```

所以它同时满足：

```text
电梯运行计算可以并行
共享状态读写有明确锁保护
tick 边界由 stepMu 保证不会交错
```

### `main.go` 的变化

服务启动时现在会先启动电梯 goroutine：

```go
system.StartElevatorRunners(ctx)
server.StartAutoStep(ctx, defaultAutoStepInterval)
```

顺序是：

```text
先启动每部电梯的 goroutine
再启动自动 step runner
```

这样后台自动 step 触发时，`System.Step()` 已经可以把 tick 命令发送给每部电梯。

### 新增测试

新增测试：

```go
TestStepWithElevatorRunnersAdvancesEachElevator
```

测试做了这些事：

```text
1. 创建 2 部电梯
2. 手动给两部电梯设置 Stops
3. 启动 StartElevatorRunners
4. 调用 System.Step()
5. 验证两部电梯都推进了一个行动 tick
6. 验证 CurrentTick 只增加 1
```

这个测试证明：

```text
Step 会向每部电梯 goroutine 发送 tick 命令
每部电梯 goroutine 会推进自己负责的电梯
整个系统仍然保持一个全局 tick
```

### 这次并发模型的代码阅读顺序

建议按这个顺序读：

```text
1. internal/elevator/model.go
   看 System 里的 mu、stepMu、elevatorCommands、elevatorRunnersDone。

2. internal/elevator/elevator_runner.go
   看 elevatorTickCommand、elevatorTickResult、StartElevatorRunners、runElevator、stepElevatorState。

3. internal/elevator/system.go
   看 Step、stepWithElevatorRunners、cloneElevator、stepElevator。

4. cmd/server/main.go
   看服务启动时如何启动每部电梯 goroutine。

5. internal/elevator/system_test.go
   看 TestStepWithElevatorRunnersAdvancesEachElevator。
```

### 本次验证

普通测试：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
```

race 检查：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

结果都通过。


## 2026-05-11：移除手动 Step API，系统时间统一由后端时钟推进

这次把 `POST /api/step` 和 API 层的手动 step 控制通道删除了。

新的系统语义是：

```text
客户端只能创建请求和读取状态。
系统时间只由 Go 后端的自动 ticker 推进。
```

也就是说，保留的核心 API 是：

```text
GET  /api/state
POST /api/request
```

不再提供：

```text
POST /api/step
```

### 为什么删除手动 Step

之前的设计里，`StartAutoStep` 同时支持：

```text
自动 ticker 触发 Step
手动 POST /api/step 触发 Step
```

为了支持手动 Step，API 层需要维护：

```text
stepCommand
stepCommands channel
stepRunnerStarted
RequestStep
runStepCommand
```

但真实运行时，前端并不会展示“手动推进一个时间片”的按钮。

电梯系统更自然的语义是：

```text
系统时钟一直运行。
用户只负责发出乘梯请求。
前端只负责轮询状态并渲染。
```

所以删除手动 Step 后，职责更清楚：

```text
internal/api/runner.go
  只负责后台 ticker。

internal/api/handler.go
  只负责 health、state、request 和静态文件。

web/app.js
  定时 GET /api/state，不再 POST /api/step。
```

### `StartAutoStep` 简化后的逻辑

现在 `StartAutoStep` 只做一件事：

```text
每隔 interval 调用一次 s.System.Step()
```

核心代码结构是：

```go
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
```

这里没有手动 step channel。

`select` 只等待两件事：

```text
ctx.Done()
  系统关闭，后台 goroutine 退出。

ticker.C
  到了下一个系统时间片，调用 System.Step()。
```

### 前端的变化

之前前端的 `tick()` 做两件事：

```text
1. POST /api/step
2. GET /api/state
```

现在改成：

```text
1. GET /api/state
```

原因是后端已经自己按固定间隔推进系统。

前端只需要定时读取状态：

```js
setInterval(tick, 800);
```

这里的 `tick()` 只是前端刷新 UI 的节奏，不再等价于后端的系统时间片。

后端真正的系统时间片由：

```go
defaultAutoStepInterval = 500 * time.Millisecond
```

控制。

### 并发模型的简化

删除手动 Step 后，系统推进路径变成：

```text
time.Ticker
  -> StartAutoStep goroutine
  -> System.Step()
  -> scheduler.Assign()
  -> elevatorCommands
  -> 每部电梯 goroutine
  -> elevatorTickResult
  -> System.Step() 合并结果
```

请求进入系统的路径是：

```text
POST /api/request
  -> System.AddRequestSnapshot()
  -> 等下一次自动 Step 调度
```

状态读取路径是：

```text
GET /api/state
  -> System.Snapshot()
```

这样推进时间的入口只有一个，API 表面也更接近真实电梯系统。

## 2026-05-11：删除 Step 的同步推进路径

这次删除了 `internal/elevator/system.go` 里的 `stepLocked()` 和同步用的 `stepElevator()` 封装。

原因是当前服务启动顺序已经固定为：

```text
NewSystem
StartElevatorRunners
StartAutoStep
```

也就是说，真实运行时调用 `System.Step()` 之前，每部电梯的 goroutine 已经启动。

因此 `Step()` 不再需要维护两套路径：

```text
旧路径 1：没启动 elevator runners 时，走 stepLocked()
旧路径 2：启动 elevator runners 后，走 stepWithElevatorRunners()
```

现在只保留正式路径：

```text
System.Step()
  -> stepWithElevatorRunners()
```

如果有人在没有调用 `StartElevatorRunners()` 的情况下直接调用 `Step()`，会返回错误：

```text
elevator runners are not started
```

这样做的好处是：

```text
代码里只有一种系统推进模型
测试和服务启动流程保持一致
读 Step() 时不会误以为同步路径也是正式运行模式
```

对应地，测试里凡是需要调用 `Step()` 的地方，也都先调用：

```go
system.StartElevatorRunners(ctx)
```

测试辅助函数 `startElevatorRunnersForTest` 用来减少重复代码。

## 2026-05-11：补充第 8 阶段测试

这次完成 `docs/instructions-from-agent.md` 第 8 部分里除 git 提交外的测试任务。

新增测试主要覆盖三类风险。

### 核心调度逻辑测试

新增文件：

```text
internal/elevator/scheduler_test.go
```

覆盖内容：

```text
FCFS 会选择 ID 最小的 pending 请求，而不是依赖 map 遍历顺序
FCFS 会跳过已有 Stops 的忙碌电梯和 EmergencyStop 电梯
FirstAvailable 在 1 号电梯忙碌时不会错误分配给其他电梯
同楼层、同方向的重复请求可以合并进同一个 StopPlan，但保留不同 Request ID
```

这里的重点是验证调度器对“请求顺序”“电梯可接单状态”“重复请求”的处理。

### 核心模型边界测试

在：

```text
internal/elevator/system_test.go
```

补充：

```text
非法楼层 0、21 会被 AddRequest 拒绝
边界楼层 1、20 可以创建请求
```

这些测试保证后端核心模型自身有边界检查，而不是只依赖 API handler。

### API handler 测试

在：

```text
internal/api/handler_test.go
```

补充：

```text
POST /api/request 拒绝非法楼层
POST /api/request 接受边界楼层
重复 POST /api/request 会创建不同 ID 的请求
POST /api/scheduler 可以切换合法调度算法
POST /api/scheduler 会拒绝未知算法
```

现在第 8 阶段的“API handler 写基础测试”不再只依赖 curl 示例，而是有可自动运行的 Go 测试。

### 验证命令

普通测试：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
```

并发数据竞争检查：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

两者均已通过。

## 2026-05-11：补充最小运行日志

这次使用 Go 标准库 `log.Printf` 补充最小运行日志，不引入第三方日志库。

日志覆盖四类事件：

```text
request created
  请求进入系统。

request assigned
  请求被调度器分配给某部电梯。

request completed
  请求完成，写入 SQLite，并从运行态 Requests 删除。

auto step failed
  后台自动 Step 出错。
```

其中 `auto step failed` 已经存在于：

```text
internal/api/runner.go
```

本次新增的位置是：

```text
internal/elevator/system.go
  addRequestLocked       记录 request created
  assignRequestToElevator 记录普通调度器的 request assigned
  completeRequest        记录 request completed

internal/elevator/SCANScheduler.go
  assignSCANRequest      记录 SCAN 调度器的 request assigned
```

目前没有记录每一个电梯移动 tick。

原因是移动日志会非常密集，尤其当前后端会持续自动 Step。先记录请求生命周期和错误日志，能覆盖大多数调试需求；如果后续要调试电梯运动细节，再单独增加移动日志或调试开关。

示例日志形态：

```text
request created: id=1 floor=8 direction=up kind=hall tick=12
request assigned: id=1 elevator=2 floor=8 direction=up kind=hall scheduler=nearest-idle tick=13
request completed: id=1 elevator=2 floor=8 direction=up kind=hall createdTick=12 assignedTick=13 completedTick=17
auto step failed: elevator runners are not started
```

这些日志和 SQLite 的职责不同：

```text
SQLite completed_requests
  保存已经完成的请求历史，适合统计和报告。

log.Printf
  显示运行过程中的关键事件，适合开发调试。
```

本次验证：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

两者均已通过。

## 2026-05-11：去掉两处代码冗余

本阶段清理了两处真正的冗余——不是并发模型的复杂度，而是实现了相同功能但代码绕弯的写法。

### 冗余 1：`completeRequest` 的无意义 copy/copy-back

**旧代码：**

```go
completedRequest := *req       // ① 值拷贝
completedRequest.Status = RequestDone
completedRequest.CompletedTick = completedTick
s.requestStore.SaveCompletedRequest(completedRequest)  // ② 写副本到 DB
*req = completedRequest        // ③ 把副本写回原指针
delete(s.Requests, requestID)  // ④ 删除 map 条目
```

③ 更新了 `req` 指向的值（为了让外部持有该指针的测试代码能读到更新后的状态），然后 ④ 立刻从 map 删了。这个流程绕了两层拷贝，中间多了一个 `completedRequest` 变量。

**新代码：**

```go
req.Status = RequestDone
req.CompletedTick = completedTick
s.requestStore.SaveCompletedRequest(*req)
delete(s.Requests, requestID)
```

直接在指针上改字段，写 DB 时传 `*req` 即可。两行中间变量和一整次结构体拷贝被去掉。

### 冗余 2：`AddRequest` + `AddRequestSnapshot` 两个公开方法做同一件事

之前：

```text
AddRequest        返回 *Request → 测试用
AddRequestSnapshot  返回 Request →  handler 用
```

底层 `addRequestLocked` 已经封装了全部创建逻辑，两个公开方法只是返回类型不同（指针 vs 值）。`AddRequestSnapshot` 的动机是"锁内创建 + 锁内拷贝，返回安全的值给 API 层"，但 `AddRequest` 也是锁内创建（`defer s.mu.Unlock()`），调用方在解锁后立即 `*req` 取值拷贝，效果完全一样。

**改动：**

- 删除 `AddRequestSnapshot`
- handler 改用 `AddRequest`，需要值拷贝时直接用 `*req`（Go 的语法：取指针指向的值）

```go
// 旧
createdRequest, err := s.System.AddRequestSnapshot(...)
response := map[string]any{"request": createdRequest}

// 新
req, err := s.System.AddRequest(...)
response := map[string]any{"request": *req}
```

### 顺便移动：`moveOneTick` 从 system.go 到 elevator_runner.go

`moveOneTick` 只在 `stepElevatorState`（elavator_runner.go）中被调用，但定义在 system.go 里。移到调用的文件里，减少跨文件搜索。

### 本次验证

```bash
go build ./...
go test ./...
```

全部 20 个测试通过。

## 2026-05-12：实现 SCAN 调度算法

这次根据 `docs/instructions-from-agent.md` 第 10 部分，在：

```text
internal/elevator/SCANScheduler.go
```

实现 SCAN 调度算法。

### 当前 SCAN 的调度顺序

`Assign` 的入口逻辑是：

```text
1. 如果没有 pending 请求，直接返回 false
2. 优先尝试把请求追加给顺路运行中的电梯
3. 如果没有顺路电梯，再找最近的空闲电梯
```

顺路电梯需要满足：

```text
电梯没有 EmergencyStop
电梯已经有 Stops，也就是正在服务某个方向
请求楼层在电梯当前 ScanDirection 的前方
hall 请求方向和 ScanDirection 一致
cabin 请求只要求楼层在前方
```

### 停靠计划排序

SCAN 的停靠计划按扫描方向排序：

```text
ScanDirection = up
  Stops 按楼层升序

ScanDirection = down
  Stops 按楼层降序
```

这样电梯会沿一个方向持续服务前方请求，而不是每次加入新请求后乱序掉头。

### 空闲电梯如何接单

如果没有正在顺路的电梯，调度器会在空闲电梯中选择候选：

```text
优先选择符合当前 ScanDirection 的请求
如果没有，就允许反向接单
同优先级下选距离最近的电梯
```

当空闲电梯接到反方向请求时，会调整自己的 `ScanDirection`。

### 测试覆盖

新增：

```text
internal/elevator/SCANScheduler_test.go
```

覆盖：

```text
运行中上行电梯可以追加顺路上行请求
没有顺路电梯时选择最近空闲电梯
下行扫描时 Stops 按楼层降序排列
```

验证命令：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

两者均已通过。

## 2026-05-12：让 SCAN 调度复用 cost 函数

这次重构 `internal/elevator/SCANScheduler.go`，让 SCAN 不再自己手写“距离最近 / 优先级”排序，而是复用之前预留的 cost 基础设施：

```text
EstimateAssignmentScore
AssignmentCandidate
```

### 候选电梯范围

SCAN 的候选现在包括：

```text
空闲电梯
正在运行但可以顺路追加该请求的电梯
```

仍然排除：

```text
EmergencyStop 电梯
正在运行但请求不顺路的电梯
```

这里保留“忙碌电梯必须顺路”的硬约束，是为了不破坏 SCAN 的核心方向语义。候选内部再交给 cost 排序。

### cost 在 SCAN 中发挥的作用

每个候选都会调用：

```go
EstimateAssignmentScore(s, scoreElevator, *request)
```

它会综合：

```text
DistanceCost
  请求楼层离电梯当前位置越远，成本越高。

TurnPenalty
  如果电梯当前方向和请求楼层不一致，成本增加。

StopPenalty
  已有停靠越多，成本越高，避免把任务都压给同一部电梯。

WaitCompensation
  请求等待越久，总成本越低，减少饥饿风险。
```

对已有 `Stops` 但 `Direction == idle` 的测试/边界状态，SCAN 会用 `ScanDirection` 作为 cost 的运行方向：

```go
if e.Direction == DirectionIdle && len(e.Stops) > 0 {
	e.Direction = e.ScanDirection
}
```

这样 cost 的 TurnPenalty 能正确理解这部电梯的扫描方向。

### 新增测试

在 `internal/elevator/SCANScheduler_test.go` 里补充：

```text
TestSCANSchedulerUsesCostStopPenalty
```

这个测试构造：

```text
一部顺路但已有多个 stop 的电梯
一部距离稍远但空闲的电梯
```

期望 SCAN 选择空闲电梯，说明 `StopPenalty` 确实参与了 SCAN 选择。

### 验证

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

两者均已通过。

## 2026-05-12：实现 LOOK 调度算法

这次在 `internal/elevator/LOOKScheduler.go` 里补全了 LOOK 算法。

LOOK 和 SCAN 的代码结构很接近：它们都会维护电梯的 `ScanDirection`，都会允许运行中的电梯顺路追加请求，也都会使用 `EstimateAssignmentScore()` 这个 cost 函数选择候选电梯。

二者真正的区别在方向切换：

```text
SCAN：
  理论上沿当前方向走到物理边界，例如 1 楼或顶楼，再反向。

LOOK：
  不等物理边界。
  如果当前方向前方已经没有停靠计划，但反方向还有停靠计划，就立即反向。
```

所以 LOOK 新增了这个核心判断：

```go
func prepareLOOKDirection(e *Elevator) {
	normalizeLOOKDirection(e)
	if len(e.Stops) == 0 {
		return
	}

	if hasLOOKStopAhead(*e, e.ScanDirection) {
		return
	}

	oppositeDirection := oppositeLOOKDirection(e.ScanDirection)
	if hasLOOKStopAhead(*e, oppositeDirection) {
		e.ScanDirection = oppositeDirection
	}
}
```

这段代码的意思是：

```text
1. 先保证 ScanDirection 是 up 或 down。
2. 如果电梯没有停靠计划，就不需要提前反向；后续接单时会按请求楼层重新对齐方向。
3. 如果当前扫描方向前方还有 stop，就继续保持方向。
4. 如果当前方向前方没有 stop，但反方向还有 stop，就把 ScanDirection 切到反方向。
```

分配请求时，LOOK 的候选电梯规则是：

```text
空闲电梯可以接单
运行中电梯只有在请求楼层位于当前 ScanDirection 前方时可以接单
hall 请求还要求请求方向和 ScanDirection 一致
cabin 请求只看目标楼层是否顺路
EmergencyStop 电梯不能接单
```

候选电梯之间仍然使用 cost 函数：

```go
score := EstimateAssignmentScore(s, scoreElevator, *request)
```

这意味着 LOOK 也会考虑：

```text
DistanceCost
TurnPenalty
StopPenalty
WaitCompensation
```

### 新增测试

新增文件：

```text
internal/elevator/LOOKScheduler_test.go
```

覆盖了三个行为：

```text
TestLOOKSchedulerAppendsRequestAlongTheWay
  验证 LOOK 可以给运行中顺路电梯追加请求。

TestLOOKSchedulerTurnsAroundWhenNoStopAhead
  验证当前方向前方没有 stop、反方向有 stop 时，LOOK 会提前反向。

TestLOOKSchedulerUsesCostStopPenalty
  验证 LOOK 和 SCAN 一样复用 cost 函数，已有多个 stop 的电梯会因为 StopPenalty 变贵。
```

### 验证

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
GOCACHE=/tmp/os_sp26_proj1-go-build go test -race ./...
```

两者均已通过。

## 2026-05-12：重构配置变更为系统级重启

上一版的 `POST /api/floor-count` 和 `POST /api/elevator-count` 试图在运行时微调 System 的字段，导致 goroutine 生命周期管理逻辑（`runnerCtx`、`runnerCancel`、`stopElevatorRunners`、`restartElevatorRunners`）渗入了 `model.go` 和 `elevator_runner.go`。

这次重构用一个更简单的方案替代：**改配置 = 重建整个 System 对象**。

### 核心思路

```text
旧方案：修改 System 内部字段 → 停止 goroutine → 重建 channel → 重启 goroutine
新方案：用新参数 NewSystem() → 启动新 goroutine → 替换指针 → 关闭旧 System
```

好处：

```text
1. System 层完全不需要知道"配置可以在运行时被修改"
2. model.go 的 runnerCtx / runnerCancel 删除，context import 删除
3. elevator_runner.go 的 stopElevatorRunners / restartElevatorRunners 删除
4. system.go 的 SetFloorCount / SetElevatorCount 删除
5. 所有重启逻辑集中在 api.Server.RestartSystem() 一个方法里
```

### 代码改动

**model.go：**
- 删除 `runnerCtx context.Context` 和 `runnerCancel context.CancelFunc` 字段
- 删除 `context` import

**elevator_runner.go：**
- `StartElevatorRunners` 还原为直接使用传入的 context，不再内部派生和存储
- 删除 `stopElevatorRunners()` 和 `restartElevatorRunners()`

**system.go：**
- 删除 `SetFloorCount()` 和 `SetElevatorCount()` 两个方法

**handler.go（合并原 runner.go）：**
- `Server` 新增字段：`mu`、`config`、`baseCtx`、`autoStepCancel`、`autoStepInterval`、`autoStepStarted`
- `NewServer` 签名改为 `NewServer(system, config)`，保存重建系统所需的配置
- 新增 `RestartSystem(floorCount, elevatorCount int) error`：
  1. 校验范围
  2. 取消当前 auto-step goroutine
  3. 用新参数调用 `elevator.NewSystem()`
  4. 为新 System 启动 elevator runners
  5. 原子替换 `s.System` 指针
  6. 关闭旧 System（释放 DB 连接）
  7. 重新启动 auto-step goroutine
- `StartAutoStep` 现在存储 `baseCtx` 和 `interval` 以备重启使用
- 新增 `startAutoStepLocked()` 内部方法，auto-step goroutine 每次 tick 通过 `s.mu` 读取当前 System 指针
- `handleFloorCount` / `handleElevatorCount` 改为读取另一个参数的当前值，然后调用 `RestartSystem`
- 所有 handler 通过 `s.mu` 保护对 `s.System` 的访问

**main.go：**
- 将 `SystemConfig` 提取为变量，传给 `NewServer`

**runner.go：**
- 删除，内容已并入 handler.go

### auto-step goroutine 的处理

auto-step goroutine 内部会通过 `s.mu` 读取最新的 `s.System` 指针：

```go
s.mu.Lock()
sys := s.System
s.mu.Unlock()
sys.Step()
```

这样当 `RestartSystem` 替换 `s.System` 后，下一次 tick 自动使用新系统，不需要外部协调。

`RestartSystem` 在替换前先取消旧的 auto-step context，确保旧 goroutine 退出，再启动新的。

### 验证

```bash
go build ./...
go test ./...
go test -race ./...
```

三者均已通过。

## 2026-05-12：新增前端运行配置 API，并把系统节奏加快一倍

这次新增了：

```text
GET /api/config
```

它用于把后端的运行节奏暴露给前端，避免前端硬编码轮询间隔。

返回字段包括：

```text
autoStepIntervalMs
  后端自动 Step 的间隔，单位毫秒。

ticksPerFloor
  电梯移动一层需要几个 tick。

doorBaseTicks
  开门等待的基础 tick 数。

tickPerPassenger
  每个乘客额外增加的开门等待 tick 数。
```

本次同时把后端默认自动推进间隔从：

```go
defaultAutoStepInterval = 500 * time.Millisecond
```

改成：

```go
defaultAutoStepInterval = 250 * time.Millisecond
```

这表示整个系统 tick 节奏加快一倍。因为前端现在会读取 `/api/config`，所以以后如果继续调整后端 tick 间隔，不需要再同步修改前端的 `setInterval`。

### 验证

新增测试：

```text
TestHandleConfigReturnsAutoStepInterval
```

验证命令：

```bash
go test ./...
```

已通过。
