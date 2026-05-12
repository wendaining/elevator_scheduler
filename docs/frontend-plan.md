# Vue 3 + Element Plus 前端实现计划

## Context

用 Vue 3 + Element Plus 替换当前原生 HTML/CSS/JS 的 web/ 目录。前端需要展示 k 台电梯在 n 层楼中运行的状态，右侧提供配置面板。所有状态通过轮询 `GET /api/state` 获取，操作通过 REST API 提交。

## 可用 API 清单（已存在，无需改后端）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/config` | 获取前端轮询所需的后端运行配置，例如 `autoStepIntervalMs` |
| GET | `/api/state` | 全量系统状态，JSON 返回 |
| POST | `/api/request` | 创建请求 `{floor, direction, kind}` |
| POST | `/api/scheduler` | 切换调度算法 `{name}` |
| POST | `/api/floor-count` | 设置楼层数 `{floorCount}` |
| POST | `/api/elevator-count` | 设置电梯数 `{elevatorCount}` |

后端 `GET /api/state` 返回的 JSON 字段参考 model.go 的 json tag：`floorCount`, `currentTick`, `elevators[]` (含 `id`, `currentFloor`, `direction`, `scanDirection`, `doorOpen`, `stops[]`, `doorRemainingTicks`), `requests{}`, `schedulerName`。

静态文件通过 `http.FileServer(http.Dir("web"))` 提供，Vue 构建产物放到 `web/` 目录即可。

## 逐步实现计划

### 第 1 步：搭建 Vue 3 + Element Plus 工程

- [x] 在 `web/` 目录手动创建最小 Vite + Vue 3 配置
- [x] 安装 `element-plus`
- [x] 配置 Vite 构建输出目录为 `web/dist`，dev proxy 转发 `/api` 到 `localhost:8080`
- [x] 创建一个最小 `App.vue` 验证能跑
- [x] 删除旧的 `index.html`、`app.js`、`style.css`
- [x] **验证**：`npm run dev` 能看到 Vue 页面

### 第 2 步：建立 API 通信层

- [x] 创建 `src/api.js`，封装所有后端 API 调用：
  - `fetchConfig()` → `GET /api/config`（读取 `autoStepIntervalMs`，避免前端硬编码轮询间隔）
  - `fetchState()` → `GET /api/state`（轮询用，间隔由 `fetchConfig()` 返回值决定）
  - `createRequest(floor, direction, kind)` → `POST /api/request`
  - `setScheduler(name)` → `POST /api/scheduler`
  - `setFloorCount(n)` → `POST /api/floor-count`
  - `setElevatorCount(n)` → `POST /api/elevator-count`
- [x] 在 `App.vue` 的 `onMounted` 中先读取 `fetchConfig()`，再按后端返回的 `autoStepIntervalMs` 轮询 `fetchState`
- [x] 整个应用的状态由 `fetchState` 的返回值驱动，通过 Vue 的 `provide` 向下传递
- [x] **验证**：浏览器 Console 能看到 state JSON

### 第 3 步：左侧电梯可视化区域

- [x] 创建 `ElevatorShaft.vue`（单台电梯井道组件）：
  - 竖向排列 n 个楼层方块（n = floorCount），紧密排列
  - 每个方块内有两个按钮：▲ 上行 / ▼ 下行
  - 按钮点击时调用 `createRequest(floor, "up"/"down", "hall")`
  - 顶部显示电梯序号标签，标签可点击选中（emit 事件到父组件）
- [x] 创建 `BuildingView.vue`（整体大楼可视化）：
  - 水平排列 k 台 `ElevatorShaft`，k = elevatorCount
  - 使用 CSS Flexbox，响应式均分宽度
- [x] 电梯位置直接由当前楼层方块高亮表示：
  - 不再渲染额外的绝对定位轿厢方块
  - 根据 `elevator.currentFloor` 给对应楼层格子添加状态颜色
  - 楼层 1 在最下方；视觉上不再使用 `MoveRemainingTicks` 做中间插值
- [x] **验证**：页面能看到电梯方块，点击楼层按钮能触发 API

### 第 4 步：电梯位置和颜色

- [x] 在 `ElevatorShaft.vue` 中用当前楼层格子表示电梯位置：
  - 取消额外的电梯轿厢方块组件，不再做 `top` 平滑移动
  - 当前楼层方块直接变色，视觉更稳定
  - 颜色逻辑：
    - `doorOpen === true` → 当前楼层格子红色（开门等待）
    - `doorOpen === false && direction !== "idle"` → 当前楼层格子黄色（移动中）
    - `direction === "idle"` → 当前楼层格子绿色（空闲待命）
  - 移动时当前楼层格子内显示 ▲/▼ 方向箭头
  - 右上角灰色圆形角标显示 `stops` 数量
- [x] **验证**：提交请求后，当前楼层格子按状态变黄/红/绿，不再出现轿厢滑动卡顿

### 第 5 步：右侧配置面板

- [x] 创建 `ControlPanel.vue`：
  - **楼层数调节**：`el-slider`，范围 [2, 40]，变化时调用 `setFloorCount`
  - **电梯数调节**：`el-slider`，范围 [1, 10]，变化时调用 `setElevatorCount`
  - **当前 Tick 显示**：大字数字，实时刷新
  - **调度算法切换**：`el-select` 下拉菜单：
    - 显示名称映射：`first-available` → "优先空闲算法"、`nearest-idle` → "最近优先算法"、`fcfs` → "先来先服务算法"、`scan` → "SCAN 电梯算法"、`look` → "LOOK 电梯算法"
    - 切换时调用 `setScheduler`
  - **Cabin 请求**：
    - 选中某台电梯后，显示选中标识（"已选中 #k 号电梯"）+ 楼层滑块 + 确认按钮
    - 滑块范围 `[1, floorCount]`，与当前楼层数联动
    - 滑块右侧实时显示当前选中的楼层数字
    - 确认按钮调用 `createRequest(floor, "idle", "cabin")`；确认成功后自动取消选中，滑块归位
    - 未选中电梯时，该区域显示浅灰色提示文字："点击电梯顶部标签以发起轿厢内请求"
    - 不限制目标楼层范围
  - **日志终端**：底部一个深色区域，显示最近的前端操作事件，最多保留最近 N 条
- [x] 创建 `LogTerminal.vue`，由 `ControlPanel.vue` 传入日志列表
- [x] 整体布局：`App.vue` 使用左侧 flex: 3（约 75%）+ 右侧 flex: 1（约 25%）
- [x] **验证**：所有控件可用，调节滑块/切换算法后 state 相应变化

### 第 6 步：响应式布局和细节

- [x] 电梯方块大小根据楼层数自适应：`grid-template-rows: repeat(floorCount, minmax(0, 1fr))`
- [x] 窗口 resize 时由 CSS Grid 自动重新分配高度，不再手动读取 DOM 高度
- [ ] Element Plus 主题色自定义为黑灰色系
- [x] 添加微动效：当前楼层格子的背景色、边框和阴影 transition
- [ ] **验证**：调整浏览器窗口大小，布局不乱；调节楼层数，电梯方块自适应

### 第 7 步：在 main.go 中配置静态文件路径

- [ ] Vite 构建后产物在 `web/dist/`，修改 `main.go` 中 `http.Dir("web")` → `http.Dir("web/dist")`
- [ ] **验证**：`go run ./cmd/server` 后浏览器访问 `localhost:8080` 看到新前端

## 关键设计决策

| 问题 | 方案 |
|------|------|
| 状态管理 | 直接用 Vue reactive + provide/inject，不引入 Pinia（项目规模小） |
| 轮询频率 | 前端读取 `GET /api/config` 的 `autoStepIntervalMs`，与后端 tick 间隔保持一致 |
| 电梯位置显示 | 不渲染额外轿厢方块；用 `currentFloor` 对应楼层格子高亮表示位置 |
| Vite dev proxy | 开发时 `vite.config.js` 配置 proxy 到 `localhost:8080`，避免 CORS |
| 删除旧文件 | index.html、app.js、style.css 全部删除 |

## 文件结构（最终）

```
web/
  vite.config.js
  package.json
  index.html          (Vite 入口)
  src/
    main.js
    App.vue
    api.js            (API 封装)
    components/
      BuildingView.vue
      ElevatorShaft.vue
      ControlPanel.vue
      LogTerminal.vue
```

## 验证方式

1. `cd web && npm run dev` → 开发服务器正常启动
2. `go run ./cmd/server` → 后端启动，前端 proxy 到后端
3. 点击楼层按钮 → 后端收到请求，电梯开始移动
4. 调节滑块 → 楼层数/电梯数变化
5. 切换算法下拉 → 调度器切换，页面显示当前算法
6. 选中电梯 → 右侧面板显示 cabin 请求入口
