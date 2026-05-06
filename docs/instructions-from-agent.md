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

- [ ] 创建 `internal/api/handler.go`
- [ ] 设计 `GET /api/state`，返回当前所有电梯状态
- [ ] 设计 `POST /api/request`，接收楼层和方向
- [ ] 在 `cmd/server/main.go` 中把 API handler 和 elevator system 接起来
- [ ] 给非法楼层、非法方向返回清晰的错误响应
- [ ] 用 `curl` 或浏览器验证 API
- [ ] 记录请求和响应示例到 `docs/record.md`，以后可以整理进报告或 `docs/api.md`
- [ ] 做一次小提交，例如 `feat: expose elevator state api`

## 5. 建立最小前端页面

目标：打通“按钮 -> HTTP 请求 -> 后端状态 -> 页面刷新”的闭环。

- [ ] 创建 `web/index.html`
- [ ] 创建 `web/app.js`
- [ ] 创建 `web/style.css`
- [ ] 页面显示 20 层楼的上行 / 下行按钮
- [ ] 页面显示 5 部电梯的当前楼层和方向
- [ ] 前端定时请求 `GET /api/state` 刷新状态
- [ ] 点击楼层按钮时发送 `POST /api/request`
- [ ] 暂时不追求动画，先保证交互链路可靠
- [ ] 在 `docs/record.md` 记录第一次前后端打通的过程
- [ ] 做一次小提交，例如 `feat: add basic elevator web ui`

## 6. 加入更明确的调度算法

目标：从“能跑”走向“可以解释调度策略”。

- [ ] 创建 `internal/elevator/scheduler.go`
- [ ] 先实现一个简单且容易解释的算法，例如最近空闲电梯优先
- [ ] 再考虑实现 FCFS 或 SCAN
- [ ] 为不同算法定义统一接口，方便后续切换
- [ ] 在状态接口里暴露当前使用的调度算法名称
- [ ] 在前端显示当前调度策略
- [ ] 在 `docs/record.md` 记录算法思路、优点、缺点和一个具体例子
- [ ] 做一次小提交，例如 `feat: add nearest elevator scheduler`

## 7. 引入并发模型

目标：满足课程中“每部电梯作为独立执行单元”的要求。

- [ ] 在同步版本已经稳定后，再开始加入 goroutine
- [ ] 设计每部电梯的运行循环
- [ ] 使用 channel 传递请求或控制信号
- [ ] 使用 mutex 或单线程事件循环保护共享状态，避免数据竞争
- [ ] 明确哪些状态由调度器维护，哪些状态由电梯维护
- [ ] 学习并运行 `go test -race` 或类似方式检查数据竞争
- [ ] 在 `docs/record.md` 记录：并发版本相比同步版本改变了什么
- [ ] 做一次小提交，例如 `feat: run elevators concurrently`

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

- [ ] 显示电梯开门 / 关门状态
- [ ] 显示每部电梯的目标楼层或任务队列
- [ ] 显示全局待处理请求
- [ ] 增加调度算法切换控件
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
