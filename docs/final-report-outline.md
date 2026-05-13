# 最终报告提纲

> 对应课程 PDF 考核点：**实现(40%)、算法(30%)、可行性(20%)、界面(10%)**。
> 学习目的：学习调度算法、通过电梯调度体会 OS 调度过程、学习特定环境下多线程编程方法。
>
> **报告格式**：使用 HTML 文件编写，方便嵌入图表（调度对比图、流程图、截图等）和代码高亮。
> 算法执行流程、并发模型、系统架构等适合用图表/流程图辅助说明，直接在报告 HTML 中渲染。
>
> **关于"可行性"**：PDF 中可行性(20%)是一个考核维度，不要求报告中单列等长章节。
> 可行性论证已分散在前面各章：架构决策论证（2.1）、数据模型设计权衡（2.3）、算法对比验证（3.6）、
> 并发安全保证（4.4）——这些都是可行性分析的实质内容。因此不单独设可行性章节。

## 篇幅分配

| 章节 | 估计占比 |
|------|---------|
| 项目概述 | ~5% |
| 系统设计与实现 | ~30% |
| 调度算法 | ~35% |
| Go 并发模型 | ~25% |
| 总结 | ~5% |

---

## 写作和排版规则

- HTML 正文中，一个 `<p>` 或表达完整语义的 `<div>` 尽量写成单行，不在标签内部手动换行；代码块、表格、列表和复杂图形结构除外。
- 某个设计点如果单独使用 `<h4>` 展开，附近同等重要的设计点也应补充 `<h4>`，避免只有一个细节被突出。例如核心数据模型中不只讲紧急制动，也应讲 `StopPlan`、tick 时间模型、运行态请求表等。
- 前端章节的配图必须优先使用实际运行截图，不使用纯示意图替代。截图应说明具体展示什么交互状态，例如楼层请求、电梯井道、控制面板、算法切换、紧急制动、日志终端。
- 如果当前还没有截图文件，可以在 HTML 中用 `<blockquote class="quote">` 写明后续应补哪张实际截图、截图应放在哪里、用什么 `<figure>` 格式引用。

---

## 1. 项目概述

- 项目定位：操作系统课程项目——电梯调度算法可视化系统
- 背景：电梯调度是操作系统进程调度的重要类比；多部电梯共享楼层资源，调度器决定哪个电梯响应哪个请求
- 流程速览：Web 前端提交乘梯请求 → 调度器选择电梯 → 每部电梯在独立 goroutine 中运行 → HTTP API 推送状态 → 前端可视化
- 技术栈：Go（net/http）、SQLite、goroutine+channel、Vue 3 + Element Plus + Vite
- 项目启动方式（复制 README 的快速开始部分）
- 章节导读
- **建议配图**：系统运行截图（整体界面一张）

---

## 2. 系统设计与实现

> 包含架构、数据模型、API、持久化、前端、测试。其中架构决策、数据模型权衡、测试结果
> 同时也是"可行性"考核的实质论证——说明设计方案为什么可行、为什么这样选。

### 2.1 系统架构概览

- 分层图：`前端 ↔ HTTP API ↔ 调度层 ↔ 电梯运行层 ↔ 持久化层`
- 各层职责一句话
- 关键设计决策：
  - 为什么 Go：goroutine 天然适合"每部电梯独立运行"的并发模型，标准库 net/http 免框架依赖
  - 为什么 SQLite：单文件零运维，不需要单独启动数据库服务，方便助教直接跑
  - 前后端通信：REST + 轮询，Vite dev proxy 避免 CORS
- **建议配图**：分层架构示意图

### 2.2 项目结构

- 用 tree 展示目录结构（`cmd/server`、`internal/elevator`、`internal/api`、`web`、`docs`、`analysis`、`scripts`）
- 各目录职责一句话

### 2.3 核心数据模型

- 领域对象：`Direction`、`Request`（含 `RequestStatus` 三态）、`StopPlan`、`Elevator`、`System`
- 设计要点：
  - `StopPlan` 替代 `[]int`：同一楼层上/下行请求不会错误合并，区分 hall_up / hall_down / cabin
  - `Requests map[int64]*Request`：哈希表 O(1) 查找，运行态只保留 pending+assigned
  - `RequestHistory` → SQLite：完成后从运行态 map 删除，写入数据库，避免状态无限增长，方便后续数据分析
  - `CurrentTick` 全局离散时钟：所有时间以 tick 为单位
  - cabin 请求直接分配给指定电梯，不进入 pending（cabin 请求不需要调度）
  - `EmergencyStop` + `EmergencyRemainingTicks`：报警暂停固定 tick 数后自动恢复
- 建议 `<h4>` 小节：
  - 请求状态与运行态存储
  - hall 请求与 cabin 请求
  - 停靠计划 `StopPlan`
  - 时间片与运行节奏（`CurrentTick`、`TicksPerFloor`、`DoorBaseTicks`、`TickPerPassenger`）
  - 电梯状态与紧急制动
- **建议配图**：核心结构体关系图（Request → Elevator → System）

### 2.4 HTTP API 设计

- 路由表（所有端点 + 方法 + 功能说明）
- 请求/响应格式举例（挑 2-3 个典型的）
- 错误处理约定（400/405/500）

### 2.5 SQLite 持久化

- 表结构（和 `Request` 字段一一对应）
- 何时写入：`completeRequest` 时
- 读取用途：Python 脚本读取做算法对比分析、写 SQL 查询统计数据
- 为什么运行态和历史分离：运行态保持轻量，历史数据用于统计

### 2.6 前端实现（简要）

- 技术栈一句话：Vue 3 + Element Plus + Vite
- 核心交互流程（不展开组件树）：
  - 按 `/api/config` 返回的间隔轮询 GET /api/state 驱动全局状态
  - 楼层上/下行按钮 → hall 请求
  - 选中电梯 + 楼层滑块 → cabin 请求
  - 配置面板：楼层数滑块、电梯数滑块、调度算法下拉、cabin 请求、紧急停止按钮、日志终端
- **建议配图**：2~4 张实际运行截图。至少包含整体界面；如果截图暂未整理进仓库，在 HTML 中用 `<blockquote class="quote">` 标注后续应补的截图内容，例如“楼层 hall 请求按钮与电梯井道”、“算法切换与 cabin 请求控制面板”、“紧急制动后的电梯状态与日志终端”。

### 2.7 测试策略

- 单元测试覆盖：调度器、cost 函数、Step 行为、边界楼层、非法输入
- `go test ./...` 全部通过 + `go test -race` 数据竞争检查通过
- 挑一个典型测试展示

---

## 3. 调度算法

> 报告的**核心章节**。五种算法都列出，FCFS / First-Available / Nearest-Idle 简短说明；SCAN 和 LOOK 详细展开。

### 3.1 统一调度接口

- `Scheduler` 接口：`Name() string` + `Assign(*System) bool`
- 运行时通过 `SetScheduler(name)` 切换，前端下拉菜单即生效
- `NewScheduler(name)` 工厂函数

### 3.2 简单算法（简略）

- **First-Available**：第一条空闲电梯接最早请求。局限：不考虑距离
- **FCFS**：按创建顺序分配。公平但效率可能不佳（可能造成长等待）
- **Nearest-Idle**：最早 pending 分配给距离最近的空闲电梯。比 First-Available 多了空间距离考量

### 3.3 SCAN 算法（重点）

- **来源**：磁盘调度中的电梯算法（Elevator Algorithm / SCAN）
- **核心思想**：
  - 每部电梯维护 `ScanDirection`（长期扫描方向）
  - 沿方向持续服务，没更多请求时反向
  - 顺路的运行中电梯可以追加新请求（不止空闲电梯接单）
- **执行过程**（可分步骤写，配伪代码或流程图）：
  1. 枚举 pending 请求 × 所有电梯
  2. 过滤候选：空闲电梯全可选；运行中电梯需 `canAppendSCANRequest` 判断
  3. 对每个候选计算 cost
  4. 选 cost 最低的分配
- **关键判断函数**：`canAppendSCANRequest`、`isFloorAheadInSCAN`、`requestMatchesSCANDirection`
- **Stops 排序**：上行升序 / 下行降序
- **举例**：具体场景，展示 SCAN 决策过程
- **建议配图**：SCAN 执行流程图

### 3.4 LOOK 算法（重点）

- **与 SCAN 的区别**：LOOK 只走到当前方向上最后一个请求处就反向，不会空跑到底
- **现实对应**：真实电梯不会空跑到顶楼再回头
- **实现改动**：在 SCAN 基础上修改反向条件——扫描方向上没有更多 pending 请求时即反向
- **性能对比**：LOOK vs SCAN，减少无效移动次数

### 3.5 cost 函数

- 调度器的决策核心：`DistanceCost + TurnPenalty + StopPenalty - WaitCompensation`
- 四个维度含义：
  - `DistanceCost`：电梯当前楼层到请求楼层的距离 × TicksPerFloor
  - `TurnPenalty`：需要掉头时加惩罚（20）
  - `StopPenalty`：已有 Stops 越多越忙，加惩罚（每个 stop +10）
  - `WaitCompensation`：请求等待越久越优先（`currentTick - createdTick`）
- 为什么有等待补偿：避免饥饿
- **建议配图**：cost 函数四维度的分解示意图

### 3.6 算法对比

- 表格：五种算法的策略、适用场景、优缺点
- 引用 `scripts/` 下 Python 脚本和 `analysis/` 下的对比图表（bar 图、count 图、分布直方图等）
- 用同一测试用例跑五种算法，对比完成时间、平均等待 tick 等指标
- **可行性落脚点**：算法对比数据本身就是"这个调度方案可行且高效"的最强证据

---

## 4. Go 并发模型

> 对应 PDF 学习目的："学习特定环境下多线程编程方法"。兼作 Go 并发概念介绍。

### 4.1 为什么需要并发

- 课程要求"每部电梯作为独立执行单元"
- 现实对应：多部电梯并行运行
- 本项目并发入口：HTTP handler、自动 ticker、每部电梯独立 goroutine

### 4.2 Go 并发基础

- goroutine：`go f()` 启动，比 OS 线程轻量
- channel：`ch <- v` / `v := <-ch`，goroutine 间通信
- select：同时等待多个 channel
- mutex：`sync.Mutex`，不可重入
- context：控制 goroutine 生命周期，`ctx.Done()` 通知退出
- Goroutine vs OS Thread：点到关键区别即可

### 4.3 本项目的并发设计

- **两把锁**：
  - `mu` — 保护共享数据
  - `stepMu` — 保护 tick 边界
  - 为什么两把：Step 中间放 mu 让 API 和电梯 goroutine 并行，但 stepMu 确保不两个 Step 交错
- **电梯 goroutine 通信**：
  - Step → `elevatorTickCommand`（含状态副本）→ 电梯 goroutine
  - 电梯 goroutine → `elevatorTickResult`（新状态 + 完成请求 ID）→ Step 合并
- **核心原则**：goroutine 不直接写共享 System；状态拷贝 → 独立计算 → 回传 → 统一合并
- **`cloneElevator`**：Go 切片共享底层数组，并发必须深拷贝
- **建议配图**：完整 tick 执行流程图（调度 → 分发 → 并行计算 → 合并 → tick++）
- **可行性落脚点**：`go test -race` 全部通过验证了并发安全

### 4.4 数据竞争与 `go test -race`

- 什么是数据竞争，为什么并发代码必须检查
- 本项目 `go test -race ./...` 全部通过

---

## 5. 总结

- 完成了什么
- 学到了什么：调度算法、Go 并发编程、前后端通信、工程实践
- 后续改进方向

---

## 附录

- A：API 文档（全部路由 + 请求/响应示例）
- B：项目运行指南（Go + npm）
- C：项目目录结构（tree）
