# 电梯系统并发模型

本文档只描述当前系统中实际在用的并发结构，而不是历史演进过程。

## 启动时有哪些 goroutine

`main.go` 的启动顺序：

```text
NewSystem
    ↓
StartElevatorRunners(ctx)    → 启动每部电梯的 goroutine（默认 5 个）
    ↓
StartAutoStep(ctx, 500ms)   → 启动一个后台 ticker goroutine
    ↓
ListenAndServe(:8080)        → net/http 自行为每个请求分配 goroutine
```

所以运行时至少有：5 个电梯 goroutine + 1 个 ticker goroutine + HTTP 请求 goroutine。

## System 的两把锁

```go
type System struct {
    mu     sync.Mutex   // 保护共享数据
    stepMu sync.Mutex   // 保护 tick 边界
    ...
}
```

**`mu`** — 保护所有共享字段：`Elevators`、`Requests`、`CurrentTick`、`nextRequestID` 等。

**`stepMu`** — 保证同一时间只有一个 `Step()` 在执行。不是冗余——`Step()` 中间会释放 `mu`，如果没有 `stepMu`，两个 `Step()` 可能交错：

```text
Step A：拿 mu → 调度 → 复制 → 放 mu → 等 goroutine 结果 ...
Step B：                               拿 mu → 调度 → 复制 → 放 mu  ← 交错！
```

所以 `stepMu` 管 tick 不交错，`mu` 管数据不竞争。两把锁各管各的。

## 一个完整 tick 的执行过程

每次 `StartAutoStep` 的 ticker 触发一次 `System.Step()`，流程如下：

### 第 1 段：调度（持有 mu）

```go
s.scheduler.Assign(s)              // 给 pending 请求分配电梯
commands := copy(s.elevatorCommands) // 复制 channel 列表
elevators := clone(s.Elevators)     // 深拷贝电梯状态
ticksPerFloor := s.TicksPerFloor    // 只读配置
// ... 释放 mu
```

### 第 2 段：分发 + 等待（不持 mu）

```go
for i := range elevators {
    commands[i] <- elevatorTickCommand{
        elevator:      elevators[i],   // 状态副本
        ticksPerFloor: ticksPerFloor,
        done:          resultCh,        // 回信 channel
    }
}
for i := range elevators {
    result := <-resultCh               // 等每部电梯返回
}
```

### 第 3 段：合并（重新拿 mu）

```go
for i, result := range results {
    s.Elevators[i] = result.elevator            // 写回新状态
    s.completeRequest(result.completedRequestIDs) // 完成请求 → 写 DB + 删除
}
s.CurrentTick++   // 整个 tick 结束
```

**关键设计：第 2 段不持锁。** 这样电梯 goroutine 可以并行计算，`GET /api/state` 和 `POST /api/request` 也可以正常执行。

## 电梯 goroutine 做什么

每部电梯 goroutine 的主循环很简单（`elevator_runner.go`）：

```go
for {
    select {
    case <-ctx.Done(): return
    case cmd := <-commands:
        newElevator, completedIDs, err := stepElevatorState(cmd.elevator, ...)
        cmd.done <- elevatorTickResult{newElevator, completedIDs, err}
    }
}
```

**它不直接写 `s.Elevators[i]`。** 只接收状态副本、计算新状态、通过 `done` channel 回传结果。共享状态的写回由 `stepWithElevatorRunners` 统一在锁内完成。

`stepElevatorState` 是纯函数：入参是 `Elevator` 值拷贝，出参是新 `Elevator` 和完成的请求 ID。它不读任何 `System` 字段。

## cloneElevator 为什么必须

```go
func cloneElevator(e Elevator) Elevator {
    e.Stops = append([]StopPlan(nil), e.Stops...)        // 复制 Stops 切片
    for i := range e.Stops {
        e.Stops[i].RequestIDs = append([]int64(nil), ...) // 复制 RequestIDs
    }
    return e
}
```

Go 的切片 `Stops []StopPlan` 只存指针/长度/容量，不存底层数组。普通赋值 `copy := s.Elevators[i]` 会让 `copy.Stops` 和原切片共享底层数组。并发时一个 goroutine 在读副本、另一个在写原数组，就是数据竞争。必须深拷贝两层。

## API handler 怎么保证安全

`Snapshot` 和 `AddRequest` 内部各自加 `mu`：

```go
func (s *System) AddRequest(...) (*Request, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.addRequestLocked(...)
}

func (s *System) Snapshot() ([]byte, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    return json.MarshalIndent(s, "", "  ")
}
```

handler 层不再自己拿锁，直接调 `System` 方法即可。

**并发安全边界：**

```text
POST /api/request → AddRequest()  → 拿 mu，创建请求，放 mu
                     (Step 持 mu 时，AddRequest 排队等)
                     (Step 放 mu 等 goroutine 时，AddRequest 可以进入)

GET /api/state   → Snapshot()  → 拿 mu，编码全量 JSON，放 mu
                     (读到的永远是一个完整 tick 合并后的状态)
```

## 为什么不把 AddRequest 也改成 channel

`POST /api/step` 之前存在，后来删了——因为真实电梯的时钟应该自动推进，不是前端手动触发。

但 `AddRequest` **没走** channel。原因是：

```text
AddRequest 只是往 map 里放一个 pending 请求。
这个操作很快（O(1)），不需要排队等 goroutine 处理。
下次 Step 调度时自然会看到这个新请求。
```

如果将来请求量很大，可以在 `internal/elevator` 里引入一个请求 channel 和专门的调度循环。当前规模不需要。

## 核心设计原则（四句话）

```text
1. 两把锁分工：mu 保护数据，stepMu 保证 tick 不交错
2. 电梯 goroutine 不写共享状态，只拿副本、算结果、回传
3. Step 中间释放 mu，让 API 可以并发访问
4. 时间只由后端 ticker 推进，客户端只提交请求和读状态
```
