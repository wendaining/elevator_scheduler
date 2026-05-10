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

当前是粗粒度锁：

```text
一次只有一个 goroutine 能执行 AddRequest / Step / Snapshot 等 System 操作
```

它不是最高性能方案，但非常适合当前阶段，因为：

```text
逻辑简单
容易验证
不容易写出数据竞争
后续可以再拆细
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