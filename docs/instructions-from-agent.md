# 从零开始构建项目的 Checklist

这个 checklist 的目标不是一次性把项目做完，而是把项目拆成一组可以理解、可以运行、可以提交的小阶段。每完成一个阶段，都应该能说清楚：我新增了什么能力、学到了什么概念、下一步为什么要这样做。

## 0. 当前状态确认

- [x] 创建基础目录：`cmd/server`、`internal/elevator`、`internal/api`、`web`
- [x] 初始化 Go module：`go mod init os_sp26_proj1`
- [x] 写一个简短版 `AGENTS.md`，先说明本项目的协作原则
- [x] 在 `docs/record.md` 记录当前项目起点：目录已建立、Go module 已初始化、下一步准备做核心模型
- [x] 做一次小提交，提交类型可以是 `chore` 或 `docs`

## 1. 建立最小后端程序

目标：先让 Go 后端能启动，不急着实现电梯算法。

- [x] 创建 `cmd/server/main.go`
- [x] 在 `main.go` 中启动一个最简单的 HTTP 服务
- [x] 提供一个健康检查接口，例如 `GET /api/health`
- [x] 用浏览器或 `curl` 验证接口可以访问
- [x] 理解 `net/http` 中 `HandleFunc` 和 `ListenAndServe` 的基本作用
- [x] 记录到 `docs/record.md`：第一次启动后端服务的命令、访问地址、遇到的问题
- [x] 做一次小提交，例如 `feat: add minimal http server`

## 2. 设计电梯核心数据模型

目标：先把“系统状态”描述清楚，再写调度逻辑。

- [x] 创建 `internal/elevator/model.go`
- [x] 定义方向类型，例如 `DirectionUp`、`DirectionDown`、`DirectionIdle`
- [x] 定义请求类型，例如楼层请求和电梯内部目标楼层
- [x] 定义 `Elevator` 结构体，包含编号、当前楼层、方向、是否开门、目标楼层等字段
- [x] 定义 `System` 结构体，包含楼层数量、电梯列表、待处理请求等字段
- [x] 先不要加入 goroutine 和 channel，保持模型容易阅读
- [x] 写完后向 Agent 询问一次代码讲解，确认每个字段的含义
- [x] 做一次小提交，例如 `feat: define elevator state model`

## 3. 实现同步版本的系统逻辑

目标：先用普通函数模拟电梯移动，避免一开始就被并发复杂度干扰。

- [x] 创建 `internal/elevator/system.go`
- [x] 实现 `NewSystem(floors int, elevatorCount int)` 初始化函数
- [ ] 实现添加请求的方法，例如 `AddHallRequest(floor int, direction Direction)`
- [x] 实现查看状态的方法，例如 `Snapshot()`
- [x] 实现一个简单的 `Step()` 方法：每调用一次，系统向前推进一个时间片

  这里的“时间片”可以先理解为一次离散模拟步，而不是现实中的一整段真实时间。比如每调用一次 `Step()`，系统只做一小步事情：电梯向目标楼层移动一层、到达目标楼层后开门、或者处理一个待分配请求。这样做的好处是逻辑容易观察和测试。后续如果前端每隔 500ms 调用一次后端推进逻辑，用户看到的就是电梯一格一格移动；如果写测试，也可以连续调用多次 `Step()` 来验证电梯状态如何变化。

  示例：

  ```text
  初始状态：电梯在 1 楼，请求去 4 楼
  Step 1：电梯从 1 楼移动到 2 楼
  Step 2：电梯从 2 楼移动到 3 楼
  Step 3：电梯从 3 楼移动到 4 楼
  Step 4：电梯到达目标楼层，开门并移除该目标
  ```

- [x] 初始调度策略可以非常简单：把请求分配给最近的空闲电梯，或者先只用第一部电梯
- [x] 手动写一个很小的测试或临时调用，验证请求进入后电梯会移动
- [x] 做一次小提交，例如 `feat: add synchronous elevator simulation`

## 4. 暴露最小 HTTP API

目标：让前端能够通过 JSON 和后端交互。

- [x] 创建 `internal/api/handler.go`
- [x] 设计 `GET /api/state`，返回当前所有电梯状态
- [x] 设计 `POST /api/request`，接收楼层和方向
- [x] 在 `cmd/server/main.go` 中把 API handler 和 elevator system 接起来
- [x] 给非法楼层、非法方向返回清晰的错误响应
- [x] 用 `curl` 或浏览器验证 API
- [x] 记录请求和响应示例到 `docs/record.md`，以后可以整理进报告或 `docs/api.md`
- [x] 做一次小提交，例如 `feat: expose elevator state api`

## 5. 建立最小前端页面

目标：打通“按钮 -> HTTP 请求 -> 后端状态 -> 页面刷新”的闭环。

- [x] 创建 `web/index.html`
- [x] 创建 `web/app.js`
- [x] 创建 `web/style.css`
- [x] 页面显示 20 层楼的上行 / 下行按钮
- [x] 页面显示 5 部电梯的当前楼层和方向
- [x] 前端定时请求 `GET /api/state` 刷新状态
- [x] 点击楼层按钮时发送 `POST /api/request`
- [x] 暂时不追求动画，先保证交互链路可靠
- [x] 在 `docs/record.md` 记录第一次前后端打通的过程
- [x] 做一次小提交，例如 `feat: add basic elevator web ui`

## 6. 加入更明确的调度算法

目标：从“能跑”走向“可以解释调度策略”。

阶段顺序约束：在实现并发模型和几个合适的调度算法之前，不继续改前端。当前优先工作区域是 `internal/elevator` 包；先把调度算法和并发模型做扎实，再进入 API 和前端切换功能。

- [x] 创建 `internal/elevator/scheduler.go`
- [x] 先实现一个简单且容易解释的算法，例如最近空闲电梯优先
- [x] 再考虑实现 FCFS 或 SCAN
- [x] 为不同算法定义统一接口，方便后续切换
- [x] 在状态接口里暴露当前使用的调度算法名称
- [x] 在前端显示当前调度策略
- [x] 在 `docs/record.md` 记录算法思路、优点、缺点和一个具体例子
- [x] 在 `internal/elevator` 包中实现几个合适的调度算法，并为核心算法补测试
- [x] 做一次小提交，例如 `feat: add nearest elevator scheduler`

## 6.5 重构为高质量调度模型

目标：先把核心数据模型升级到能支撑真实调度，再继续实现更强的 SCAN/LOOK 和并发模型。这个阶段允许改 `internal/elevator/model.go`、`system.go`、scheduler、API handler 和测试，但仍然不要改前端展示逻辑。

设计原则：

- 不再把“学习版 / 简化版 / 更容易写对”作为核心算法目标。
- 小步提交，但每一步都必须通向最终的高质量模型。
- `PendingRequests` 不再作为系统事实来源；请求应保存在统一的 `Requests` 集合中，并通过状态区分 pending、assigned、done。
- 先完成模型重构和兼容测试，再继续写 cost 函数和高级调度策略。

### 6.5.1 全局时间片模型

目标：用离散时间片作为整个模拟系统的统一时间单位，而不是使用真实时间 `time.Time`。

- [x] 在 `System` 中增加全局时间片字段，例如 `CurrentTick int`
- [x] 约定系统启动时 `CurrentTick = 0`
- [x] 每调用一次 `Step()`，`CurrentTick += 1`
- [x] 所有请求时间都使用 tick 记录，例如 `CreatedTick`、`AssignedTick`、`CompletedTick`
- [x] 预留可配置时间参数，例如跨一层需要多少 tick、开门基础时间、每上 / 下一名乘客额外消耗多少 tick
- [x] 在 `Snapshot()` / `GET /api/state` 中暴露当前 tick，方便前端同步显示
- [ ] 做一次小提交，例如 `feat: add simulation tick clock`

### 6.5.2 请求模型重构

目标：让请求从“临时队列元素”升级为系统内可追踪的对象。

- [ ] 定义 `RequestStatus`，至少包含 `pending`、`assigned`、`done`
- [ ] 给 `Request` 增加唯一 ID，例如 `ID int64`
- [ ] 给 `Request` 增加 tick 字段，例如 `CreatedTick`、`AssignedTick`、`CompletedTick`
- [ ] 给 `Request` 增加 `AssignedElevatorID`，用来记录请求被分配给哪部电梯
- [ ] 在 `System` 中增加 `Requests []Request`
- [ ] 删除或停止使用 `PendingRequests []Request` 作为状态字段
- [ ] 如果需要 pending 请求，使用辅助函数从 `Requests` 中筛选 `Status == RequestPending`
- [ ] 修改 `AddRequest`：创建请求 ID，设置 `CreatedTick = CurrentTick`，初始状态为 `pending`
- [ ] 修改 `Snapshot()` 和 API 返回，确保可以观察所有请求及其状态
- [ ] 写测试验证请求创建、分配、完成时状态和 tick 正确变化
- [ ] 在 `docs/record.md` 记录：为什么请求不能在分配时从系统里消失
- [ ] 做一次小提交，例如 `feat: track requests by status`

### 6.5.3 楼层模型

目标：让楼层成为系统中的对象，而不是只有一个楼层编号。

- [ ] 定义 `Floor` 结构体
- [ ] 在 `System` 中增加 `Floors []Floor`
- [ ] 每个 `Floor` 至少记录楼层编号、等待上行人数、等待下行人数
- [ ] 设计 hall 请求和楼层人数的关系：楼层触发请求，请求记录方向和人数变化
- [ ] 明确当电梯到达楼层并完成接人时，如何减少对应楼层等待人数
- [ ] 暂时不让前端输入人数也可以，但后端模型要能表达人数
- [ ] 写测试验证创建 hall 请求会影响对应楼层等待人数
- [ ] 在 `docs/record.md` 记录：为什么楼层也需要建模
- [ ] 做一次小提交，例如 `feat: model floor waiting passengers`

### 6.5.4 电梯负载和停靠计划

目标：让电梯能表达乘客数量、容量限制和更真实的停靠计划。

- [ ] 给 `Elevator` 增加 `Capacity int`
- [ ] 给 `Elevator` 增加 `PassengerCount int`
- [ ] 根据需要增加 `DoorRemainingTicks` 或类似字段，用于表示开门 / 乘客上下电梯耗时
- [ ] 定义 `StopPlan` 结构体，用于替代 `TargetFloors []int`
- [ ] `StopPlan` 至少包含 `Floor`、`RequestIDs`、是否接上行乘客、是否接下行乘客、是否有乘客下电梯等信息
- [ ] 把 `Elevator.TargetFloors` 重构为 `Elevator.Stops []StopPlan`
- [ ] 修改电梯移动逻辑：根据 `Stops` 决定下一站
- [ ] 修改到站逻辑：开门、处理上下客、更新楼层人数、更新电梯人数、完成相关请求
- [ ] 写测试验证同一楼层上行和下行请求不会被错误合并
- [ ] 写测试验证电梯容量限制会影响接人
- [ ] 在 `docs/record.md` 记录：`StopPlan` 相比 `[]int` 解决了哪些问题
- [ ] 做一次小提交，例如 `feat: replace target floors with stop plans`

### 6.5.5 调度接口预留 cost 函数

目标：先把高级调度需要的接口边界留好，但不急着完成最终 cost 公式。

- [ ] 设计调度评分结构，例如 `AssignmentScore` 或 `AssignmentCandidate`
- [ ] 预留 cost 计算接口，例如 `EstimateCost(system *System, elevator Elevator, request Request) int`
- [ ] 让 SCAN/LOOK 调度器通过统一入口选择候选电梯
- [ ] cost 函数先返回可解释的基础分数，但结构上要能加入等待时间补偿、掉头惩罚、负载惩罚
- [ ] 写测试验证调度器能基于 cost 选择候选电梯
- [ ] 在 `docs/record.md` 记录：cost 函数当前预留了哪些维度，后续如何增强
- [ ] 做一次小提交，例如 `feat: add scheduler cost interface`

### 6.5.6 重构现有算法和 API

目标：让现有 FCFS、Nearest、SCAN/LOOK、HTTP API 都适配新模型。

- [ ] 重构 FCFS：从 `Requests` 中选择最早的 pending 请求
- [ ] 重构 Nearest：基于 pending 请求和电梯当前位置选择候选电梯
- [ ] 重构 SCAN/LOOK：基于 `Stops`、`ScanDirection` 和请求方向追加或分配停靠计划
- [ ] 重构 `POST /api/request`：创建请求，并更新楼层等待人数
- [ ] 重构 `GET /api/state`：返回 tick、floors、requests、elevators、schedulerName
- [ ] 保持 API 错误处理清晰，例如非法楼层、非法方向、人数不合法
- [ ] 用 `curl` 验证新请求模型能从 API 进入系统
- [ ] 跑通 `go test ./...`
- [ ] 在 `docs/record.md` 记录新状态 JSON 的关键字段
- [ ] 做一次小提交，例如 `feat: adapt api to request state model`

### 6.5 提交拆分要求

这个阶段禁止把所有重构塞进一个提交。推荐拆分顺序：

```text
commit 1: tick 时钟
commit 2: RequestStatus / Requests / 删除 PendingRequests
commit 3: Floor 模型和楼层人数
commit 4: Elevator 负载字段和 StopPlan
commit 5: Step 到站和上下客逻辑
commit 6: scheduler cost 接口
commit 7: FCFS / Nearest / SCAN 适配新模型
commit 8: API 适配新状态
commit 9: 测试和文档补充
```

每个提交都应满足：

- [ ] 能用一句话说明这次改动的边界
- [ ] 不混入前端 UI 改动
- [ ] 包含必要测试或至少保留明确的验证方式
- [ ] 提交前运行 `gofmt` 和 `go test ./...`
- [ ] 如果某个提交暂时导致 API 返回字段变化，必须同步更新 `docs/record.md` 说明

## 7. 引入并发模型

目标：满足课程中“每部电梯作为独立执行单元”的要求。

- [ ] 在 tick、Requests、Floors、Stops、基础调度器都稳定后，再开始加入 goroutine
- [ ] 设计每部电梯的运行循环
- [ ] 使用 channel 传递请求或控制信号
- [ ] 使用 mutex 或单线程事件循环保护共享状态，避免数据竞争
- [ ] 明确哪些状态由调度器维护，哪些状态由电梯维护
- [ ] 学习并运行 `go test -race` 或类似方式检查数据竞争
- [ ] 在 `docs/record.md` 记录：并发版本相比同步版本改变了什么
- [ ] 做一次小提交，例如 `feat: run elevators concurrently`

完成本阶段后，再回到 API 层，为调度算法切换提供 HTTP 路由。

## 7.5 调度算法切换 API

目标：在 `internal/api` 包中增加切换调度算法的接口，但前提是 `internal/elevator` 中已经有可切换的算法实现。

- [ ] 设计切换调度算法的 API，例如 `POST /api/scheduler`
- [ ] 请求体包含算法名称，例如 `{ "name": "nearest-idle" }`
- [ ] handler 调用 `System.SetScheduler(name)`
- [ ] 非法算法名称返回清晰的 `400 Bad Request`
- [ ] `GET /api/state` 继续返回当前 `schedulerName`
- [ ] 用 `curl` 验证调度算法可以切换
- [ ] 在 `docs/record.md` 记录 API 设计、请求示例和响应示例

## 8. 补充测试和调试能力

目标：让项目不是只能靠肉眼点页面验证。

- [ ] 给核心调度逻辑写单元测试
- [ ] 测试非法楼层、边界楼层、重复请求等情况
- [ ] 给 API handler 写基础测试，或者至少保留可复现的 `curl` 示例
- [ ] 增加必要日志，能看出请求进入、调度分配、电梯移动、到达楼层
- [ ] 确保 `go test ./...` 可以通过
- [ ] 做一次小提交，例如 `test: add scheduler tests`

## 9. 改善界面展示

目标：在最小功能可靠后，再改善展示效果。

前端改动顺序：只有在调度算法、并发模型、以及切换调度算法的 API 都完成之后，才继续修改前端。前端的目标是增加一个类似下拉菜单的控件，用于选择当前调度算法。

- [ ] 显示电梯开门 / 关门状态
- [ ] 显示每部电梯的目标楼层或任务队列
- [ ] 显示全局待处理请求
- [ ] 增加调度算法切换控件
- [ ] 调度算法切换控件使用类似下拉菜单的交互，调用切换算法 API
- [ ] 增加简单动画或楼层高亮
- [ ] 保证界面信息清楚，不为了好看牺牲可读性
- [ ] 做一次小提交，例如 `feat: improve elevator dashboard`

## 10. 准备课程提交材料

目标：把开发过程沉淀成最终可提交内容。

- [ ] 整理系统架构说明
- [ ] 整理调度算法说明
- [ ] 整理并发模型说明
- [ ] 整理运行方式
- [ ] 整理测试方式
- [ ] 准备截图或演示说明
- [ ] 补充 Docker 打包，如果时间允许
- [ ] 检查 README 是否能让别人从零运行项目
- [ ] 做最终提交，例如 `docs: prepare project report materials`

## 每个阶段都要坚持的习惯

- [ ] 每次只做一个小目标，不把模型、API、前端、并发混在一次提交里
- [ ] 先让代码能运行，再追求更漂亮的设计
- [ ] 遇到不懂的 Go 语法或 Web 概念，优先要求 Agent 结合当前代码解释
- [ ] 重要设计写入 `docs/record.md`，不要只留在聊天记录里
- [ ] 每次提交前运行必要命令，例如 `go test ./...`
- [ ] 提交信息使用 `feat`、`docs`、`chore`、`test` 等前缀
