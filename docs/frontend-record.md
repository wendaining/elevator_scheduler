# 前端开发记录

## 2026-05-12：搭建 Vue 3 + Element Plus 工程（计划第 1 步）

本阶段目标：删除旧的纯 HTML/CSS/JS 页面，用 Vue 3 + Vite + Element Plus 重新搭建前端工程骨架。

### 删除的旧文件

- `web/index.html`（旧原生 HTML 页面）
- `web/app.js`（旧原生 JS）
- `web/style.css`（旧样式）

### 新增文件

```text
web/
  package.json          ← 依赖声明（Vue 3、Element Plus、Vite）
  vite.config.js        ← Vite 配置 + dev proxy
  index.html            ← Vite 入口 HTML
  src/
    main.js             ← 挂载 Vue 应用 + 注册 Element Plus 插件
    App.vue             ← 根组件（当前只显示占位文字）
    components/         ← 后续组件的目录（暂为空）
```

### package.json 依赖

```json
{
  "dependencies": {
    "vue": "^3.4.0",
    "element-plus": "^2.8.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.0.0",
    "vite": "^5.0.0"
  }
}
```

### vite.config.js 关键配置

```js
server: {
  proxy: {
    '/api': 'http://localhost:8080',
  },
},
build: {
  outDir: 'dist',
}
```

开发时 Vite 把 `/api/*` 请求转发到 Go 后端（`localhost:8080`），避免跨域问题。构建产物输出到 `web/dist/`。

### Vue 3 项目结构

和旧 HTML 方式最大的区别：

```text
旧：一个 index.html + 一个 app.js + 一个 style.css，所有逻辑混在一起
新：
  index.html      只有挂载点 <div id="app">
  main.js         创建 Vue 应用 + 注册插件
  App.vue         根组件，包含 <template>（HTML）、<script setup>（逻辑）、<style>（样式）
  components/     每个 .vue 文件是一个自包含的组件
```

`.vue` 文件（单文件组件）是 Vue 的核心概念：

```vue
<template>
  <!-- HTML 模板，组件渲染的内容 -->
</template>

<script setup>
  // JavaScript 逻辑，组件的数据和行为
</script>

<style>
  /* CSS 样式，只影响当前组件 */
</style>
```

### 验证结果

```bash
cd web && npm install   # 安装 51 个包
npx vite                # Vite 在 localhost:5173 启动，224ms 就绪
```

Go 后端编译不受影响（`go build ./...` 通过）。

## 2026-05-12：建立 API 通信层（计划第 2 步）

本阶段目标：封装所有后端 API 调用，在 App.vue 中建立轮询机制和状态管理。

### 新增文件

- `web/src/api.js` — API 封装模块

### api.js 结构

```js
const BASE = '/api'

fetchState()          → GET  /api/state        // 轮询全量状态
createRequest(f, d, k) → POST /api/request      // 创建乘梯请求
setScheduler(name)    → POST /api/scheduler     // 切换调度算法
setFloorCount(n)      → POST /api/floor-count   // 设置楼层数
setElevatorCount(n)   → POST /api/elevator-count // 设置电梯数
```

每个函数返回 Promise，调用方用 `await` 获取结果。错误时抛出带描述信息的 Error。

### App.vue 的轮询机制

```js
import { ref, provide, onMounted, onUnmounted } from 'vue'

const state = ref(null)   // 响应式状态
let timer = null

onMounted(() => {
  tick()                        // 立即获取一次
  timer = setInterval(tick, 500) // 然后每 500ms 轮询
})

onUnmounted(() => {
  clearInterval(timer)  // 组件卸载时停止轮询
})

async function tick() {
  state.value = await fetchState()
}

provide('state', state)  // 后代组件 inject('state') 即可获取
```

### Vue 3 概念：ref、provide/inject、onMounted/onUnmounted

**`ref(value)`** — 创建一个响应式引用。`.value` 读写值。模板中自动解包（直接写 `state` 不用 `.value`）。

**`provide(key, value)` / `inject(key)`** — 祖先组件 provide 数据，任意后代组件 inject 获取。比 props 一层层传递更简洁，适合全局状态。

**`onMounted(fn)` / `onUnmounted(fn)`** — 组件挂载到 DOM 后执行 / 组件从 DOM 移除前执行。这里用于启动和停止轮询。

### 布局占位

App.vue 的 template 做了左右分栏：

```html
<div class="layout">
  <div class="main-area"><!-- 左侧 75%，第 3 步放电梯可视化 --></div>
  <div class="side-panel"><!-- 右侧 25%，第 5 步放配置面板 --></div>
</div>
```

当前 main-area 显示一行占位文字（楼层数 / 电梯数 / tick），证明 state 已经正常流入组件。

### 后端运行方式

由于 Vite dev server 配置了 proxy，开发时访问 `localhost:5173` 即可——`/api/*` 请求会自动转发到 `localhost:8080` 的 Go 后端。不需要手动处理 CORS。

### 验证结果

- `npx vite` 启动正常
- 启动 `go run ./cmd/server` 后，浏览器 Console 可看到 `fetchState()` 返回的 JSON 对象

## 2026-05-12：电梯可视化区域（计划第 3 步）

本阶段目标：创建电梯井道组件和整体大楼视图，楼层按钮可触发 hall 请求。

### 新增文件

- `web/src/components/ElevatorShaft.vue`
- `web/src/components/BuildingView.vue`

### 组件树

```text
App.vue
  └── BuildingView.vue            ← 楼层标签列 + k 台电梯井道
        └── ElevatorShaft.vue × k ← 单台电梯的 n 层方块 + 轿厢
```

### ElevatorShaft.vue

**Props：** `elevator`（电梯对象）、`floorCount`、`ticksPerFloor`、`isSelected`

**Emits：** `select`（点击顶部标签）、`hall-request(floor, direction)`（点击楼层按钮）

**楼层方块：** `v-for` 从 `floorCount` → 1 渲染。每个方块内有 ▲ 上行按钮和 ▼ 下行按钮。顶层（`floorCount`）的 ▲ 和底层（1）的 ▼ 设为 `disabled`（`opacity: 0.25`）。

**动态高度：** `blockHeight = trackRef.clientHeight / floorCount`，在 `onMounted` 和 `window resize` 时重新计算，保证楼层数变化后方块自适应。

**电梯轿厢定位：**

```js
// 含 tick 间平滑偏移
let effective = e.currentFloor
if (e.moveRemainingTicks > 0 && e.direction !== 'idle') {
  const progress = 1 - (e.moveRemainingTicks / ticksPerFloor)
  effective += e.direction === 'up' ? progress : -progress
}
effective = Math.max(1, Math.min(n, effective))

const topPct = ((n - effective) / n) * 100
```

这样当 `TicksPerFloor = 5` 时，轿厢不会在第 5 个 tick 才从 2 楼跳到 3 楼，而是在 5 个 tick 内逐步移动，视觉更平滑。

**Vue 语法要点：**

- `<div :class="{ selected: isSelected }">` — 对象绑定 class：`isSelected` 为 true 时添加 `.selected`
- `@click.stop` — `.stop` 修饰符阻止事件冒泡（避免点击按钮触发外层事件）
- `$emit('hall-request', floor, direction)` — 子组件向父组件发事件，携带两个参数
- `defineProps({ elevator: { type: Object, required: true } })` — 声明组件接受的 prop 及类型约束

### BuildingView.vue

**Inject：** `state`（从 App.vue 的 provide）

**职责：** 渲染左侧楼层标签列 + k 个 `ElevatorShaft`，管理选中电梯状态。

**选中逻辑：** 点击某台电梯的顶部标签选中，再次点击取消。选中后 `selectedElevatorId` 被写入 App.vue，同时通过 provide 提供给 ControlPanel（第 5 步用）。

```js
function onSelect(id) {
  selectedElevatorId.value = selectedElevatorId.value === id ? null : id
  emit('update:selectedElevatorId', next)
}
```

### App.vue 更新

- 引入 `BuildingView`
- 新增 `selectedElevatorId` 状态，同时 provide 给后代组件
- 右侧面板暂显示选中的电梯 ID（占位，第 5 步替换）

### 验证结果

- Vite dev server 启动正常
- 页面显示 20 层 × 5 台电梯的完整布局
- 楼层按钮 hover 有反馈，边缘按钮 disabled
- 点击电梯标签可选中/取消（蓝色高亮）
- 运行 `go run ./cmd/server` 后点击楼层按钮，Network 面板可看到 `POST /api/request` 请求

## 2026-05-12：电梯颜色与动效（计划第 4 步）

本阶段目标：电梯轿厢改为半透明 + 柔光效果，用颜色区分运行状态。

### 设计思路

之前是一个灰色实心方块，现在改为 **半透明玻璃质感**：
- `rgba` 颜色 + `box-shadow` 柔光晕
- 三种状态通过 computed class 切换
- 位置和颜色都有 `transition`，视觉连续

### 状态颜色

| 状态 | 条件 | 颜色 | 视觉 |
|------|------|------|------|
| 空闲 | `direction === "idle"` | `rgba(103,194,58, 0.25)` | 绿色半透明 + 绿色光晕 |
| 移动 | `direction !== "idle" && !doorOpen` | `rgba(230,162,60, 0.3)` | 黄色半透明 + 暖色光晕 |
| 开门 | `doorOpen === true` | `rgba(245,108,108, 0.3)` | 红色半透明 + 红色光晕 |

### 关键 CSS

```css
.elevator-car {
  transition: background 0.5s, box-shadow 0.5s, border-color 0.5s;
}

.car-wrapper {
  transition: top 0.45s ease;  /* 位置平滑过渡 */
}
```

颜色和位移有各自独立的 transition，换楼层时颜色渐变和滑动同时发生。

### 新增细节

- **方向箭头**：移动时轿厢内显示 ▲ 或 ▼，空闲/开门时隐藏
- **角标**：灰色圆形数字，位于轿厢右上角，显示剩余停靠计划数
- **wrapper 层**：`car-wrapper` 负责定位，`elevator-car` 负责外观，职责分离

## 2026-05-12：把轿厢动画改为当前楼层格子高亮

本次是对第 4 步视觉方案的重构。

原来的 `ElevatorShaft.vue` 使用两层结构：

```text
floor-block      楼层格子和按钮
car-wrapper      绝对定位的电梯轿厢
elevator-car     轿厢本体和颜色
```

轿厢位置通过 `currentFloor`、`moveRemainingTicks` 和 `ticksPerFloor` 计算百分比 `top`。这个设计理论上能表达 tick 内移动进度，但实际页面中会显得电梯在格子之间一点一点挪动，视觉上不够干净。

### 新方案

删除单独的轿厢方块，不再计算轿厢 `top`。

现在电梯位置直接由当前楼层格子表达：

```js
function isCurrentFloor(floor) {
  return floor === props.elevator.currentFloor
}

function floorStateClass(floor) {
  const e = props.elevator
  if (floor !== e.currentFloor) return ''
  if (e.doorOpen) return 'is-current state-open'
  if (e.direction !== 'idle') return 'is-current state-moving'
  return 'is-current state-idle'
}
```

也就是说：

```text
currentFloor 在哪一层，哪一层的方块就高亮。
```

### 状态颜色

| 状态 | 条件 | 显示 |
|------|------|------|
| 空闲 | `direction === "idle"` | 当前楼层格子绿色 |
| 移动 | `direction !== "idle" && !doorOpen` | 当前楼层格子黄色 |
| 开门 | `doorOpen === true` | 当前楼层格子红色 |

移动时，当前楼层格子内仍然显示 ▲ 或 ▼。如果电梯还有停靠计划，右上角继续显示 `stops.length` 角标。

### 响应式高度

旧方案需要：

```text
trackRef.clientHeight / floorCount
window resize 时重新计算
```

新方案改成 CSS Grid：

```js
const trackStyle = computed(() => {
  return {
    gridTemplateRows: `repeat(${props.floorCount}, minmax(0, 1fr))`,
  }
})
```

这样每个楼层格子的高度由浏览器自动平均分配，不再需要手动读取 DOM，也不需要监听 `resize`。

### 改动文件

```text
web/src/components/ElevatorShaft.vue
web/src/components/BuildingView.vue
docs/frontend-plan.md
docs/frontend-design.md
docs/frontend-record.md
```

### 验证

```bash
npm run build
```

构建已通过。


## 2026-05-12：完成响应式布局和细节（计划第 6 步）

本阶段目标：让页面在不同窗口尺寸、楼层数和电梯数下保持可读，不因为控件挤压导致布局失控。

### 黑灰主题

在 `App.vue` 的根节点上覆盖 Element Plus 的主题变量：

```css
#app-root {
  --el-color-primary: #111827;
  --el-color-primary-light-3: #374151;
  --el-color-primary-light-7: #d1d5db;
  --el-color-primary-light-9: #f3f4f6;
}
```

这样 `el-slider`、`el-button`、`el-select` 的主色会更接近当前黑灰界面，而不是默认蓝色。

### 主布局响应式

桌面端保持：

```text
左侧电梯区域 flex: 3
右侧配置面板 flex: 1
```

窄屏时改为上下布局：

```css
@media (max-width: 900px) {
  .layout {
    flex-direction: column;
  }
}
```

左侧电梯区域在窄屏时允许横向滚动，避免 5 到 10 台电梯被压得过窄。

### 电梯井道细节

`ElevatorShaft.vue` 中按钮尺寸改为：

```css
.floor-btn {
  width: clamp(18px, 2.4vh, 24px);
  height: clamp(18px, 2.4vh, 24px);
}
```

这表示：

```text
窗口高度足够时，按钮最多 24px。
楼层很多或高度很小时，按钮可以缩到 18px。
```

这样 30 到 40 层时按钮不会明显溢出楼层格子。

### 右侧面板

`ControlPanel.vue` 内部允许滚动：

```css
.control-panel {
  overflow: auto;
}
```

窄屏时右侧面板变成两列，再更窄时变成单列：

```css
@media (max-width: 900px) {
  .control-panel {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 620px) {
  .control-panel {
    grid-template-columns: 1fr;
  }
}
```

### 验证

```bash
npm run build
```

构建已通过。

## 2026-05-12：前端轮询间隔改为读取后端配置

本次目标：整体模拟节奏加快一倍，并避免前后端分别维护 tick 间隔。

### 问题

之前前端写死：

```js
timer = setInterval(tick, 500)
```

后端也有自己的自动推进间隔：

```go
defaultAutoStepInterval = 500 * time.Millisecond
```

如果以后只改了后端，例如改成 `250ms`，但忘记改前端，前端就会每 500ms 才拉一次状态，漏掉一半 tick。页面仍然能工作，但视觉反馈会滞后。

### 新方案

后端新增：

```text
GET /api/config
```

返回示例：

```json
{
  "autoStepIntervalMs": 250,
  "ticksPerFloor": 5,
  "doorBaseTicks": 2,
  "tickPerPassenger": 1
}
```

前端新增 `fetchConfig()`：

```js
export async function fetchConfig() {
  const response = await fetch(`${BASE}/config`)
  if (!response.ok) {
    throw new Error(`GET /api/config failed: ${response.status}`)
  }
  return response.json()
}
```

`App.vue` 启动时先读取配置，再启动轮询：

```js
async function startPolling() {
  config.value = await fetchConfig()
  await tick()

  const interval = config.value.autoStepIntervalMs
  timer = setInterval(tick, interval)
}
```

这样以后要修改整体系统节奏，只需要改后端 `defaultAutoStepInterval`。

### 本次速度调整

后端默认值从：

```go
defaultAutoStepInterval = 500 * time.Millisecond
```

改为：

```go
defaultAutoStepInterval = 250 * time.Millisecond
```

这表示整个模拟系统节奏加快一倍。移动、开门等待、tick 数增长都会一起变快。

### 改动文件

```text
cmd/server/main.go
internal/api/handler.go
internal/api/handler_test.go
web/src/api.js
web/src/App.vue
docs/frontend-plan.md
docs/frontend-record.md
```

### 验证

```bash
go test ./...
npm run build
```

两者均已通过。

## 2026-05-12：修正楼层数字对齐和重复显示

本次只调整前端楼层可视化的细节。

### 问题

页面左侧已经有一列楼层数字，但每台电梯井道的每个楼层格子左上角也显示了一遍楼层数字，信息重复。

同时左侧楼层数字列没有为电梯顶部标签栏预留高度，而电梯井道本身有 `#1`、`#2` 这样的顶部标签，所以左侧数字和右侧楼层格子会产生纵向偏差。

### 修改

`ElevatorShaft.vue`：

```text
删除每个 floor-block 内部的 floor-label。
```

现在楼层数字只由 `BuildingView.vue` 左侧统一显示。

`BuildingView.vue`：

```text
给左侧楼层数字列增加 floor-label-spacer。
spacer 高度和电梯顶部标签栏一致，都是 28px。
楼层数字区域改为 CSS Grid，和电梯井道一样按 floorCount 均分行高。
```

这样左侧楼层数字和每一层方块能在同一水平线上对齐。

### 验证

```bash
npm run build
```

构建已通过。

## 2026-05-12：完成右侧配置面板（计划第 5 步）

本阶段目标：把右侧占位文本替换成可操作的配置面板。

### 新增文件

```text
web/src/components/ControlPanel.vue
web/src/components/LogTerminal.vue
```

### ControlPanel.vue

`ControlPanel.vue` 负责右侧所有操作：

```text
楼层数调节
电梯数调节
当前 tick 显示
调度算法切换
cabin 请求
前端操作日志
```

它通过 `inject('state')` 获取后端轮询状态，通过 props 获取当前选中的电梯 ID：

```js
const stateRef = inject('state')

const props = defineProps({
  selectedElevatorId: { type: Number, default: null },
})
```

### 楼层数和电梯数

使用 Element Plus 的 `el-slider`：

```html
<el-slider
  v-model="floorDraft"
  :min="2"
  :max="40"
  :step="1"
  :show-tooltip="false"
  @change="submitFloorCount"
/>
```

这里用 `@change` 而不是每次拖动都请求后端，避免拖动过程中连续重启系统。

### 调度算法切换

使用 `el-select`，前端显示中文名称，提交给后端时仍然使用算法代号：

```text
first-available → 优先空闲算法
nearest-idle    → 最近优先算法
fcfs            → 先来先服务算法
scan            → SCAN 电梯算法
look            → LOOK 电梯算法
```

### Cabin 请求

`BuildingView.vue` 的选中状态现在由 `App.vue` 统一持有，再传给 `ControlPanel.vue`。

原因是：cabin 请求提交成功后，右侧面板需要清空当前选中电梯，同时左侧顶部标签高亮也要取消。选中状态如果只存在于 `BuildingView.vue` 内部，右侧组件无法同步修改它。

`App.vue` 现在这样连接两个组件：

```html
<BuildingView
  :selected-elevator-id="selectedElevatorId"
  @update:selected-elevator-id="onSelectedChange"
/>

<ControlPanel
  :selected-elevator-id="selectedElevatorId"
  @clear-selection="selectedElevatorId = null"
/>
```

### LogTerminal.vue

`LogTerminal.vue` 是一个小型深色日志面板，只负责展示日志列表。

当前日志主要记录前端操作结果：

```text
楼层数调整
电梯数调整
调度算法切换
cabin 请求提交
失败信息
```

日志最多保留 24 条。

### 验证

```bash
npm run build
```

构建已通过。

## 2026-05-12：增加页面标题栏

本次给 Vue 前端增加顶部标题：

```text
电梯调度算法可视化程序
```

改动位置：

```text
web/src/App.vue
```

实现方式：

```html
<header class="app-header">
  <h1>电梯调度算法可视化程序</h1>
</header>
```

为了避免标题栏挤压或覆盖原来的左右布局，`#app-root` 改为纵向 flex：

```css
#app-root {
  display: flex;
  flex-direction: column;
}

.layout {
  flex: 1;
  min-height: 0;
}
```

这样 header 固定占一行，下面的电梯可视化区域和右侧面板使用剩余高度。

### 验证

```bash
npm run build
```

构建已通过。

## 2026-05-12：补充请求创建和完成事件日志

本次修正右侧事件日志只显示部分操作的问题。

之前日志主要在 `ControlPanel.vue` 内部维护，所以右侧 Cabin 请求、楼层数调整、算法切换能记录，但左侧楼层按钮创建的 Hall 请求不会进入日志。

### 新方案

把日志状态提升到 `App.vue` 统一维护：

```text
App.vue
  logs
  appendLog()
  trackRequest()
```

然后通过 `provide` 提供给子组件：

```js
provide('logs', logs)
provide('appendLog', appendLog)
provide('trackRequest', trackRequest)
```

`BuildingView.vue` 在创建 Hall 请求成功后写入日志：

```text
请求 #id 创建：Hall 上行/下行 x 楼
```

`ControlPanel.vue` 在创建 Cabin 请求成功后写入日志：

```text
请求 #id 创建：Cabin #elevator x 楼
```

### 请求完成日志

后端的运行态 `requests` 只保存 active 请求。请求完成后会从 `requests` map 删除。

前端利用这一点，在每次轮询 `GET /api/state` 后比较：

```text
上一帧 active requests
当前帧 active requests
```

如果某个 request ID 从上一帧存在变成当前帧不存在，就记录：

```text
请求 #id 完成：Hall/Cabin #elevator x 楼
```

为了避免楼层数/电梯数调整导致系统重启时误报完成，前端会在检测到 `currentTick` 变小、楼层数变化或电梯数变化时重置追踪表，不写完成日志。

### 验证

```bash
npm run build
```

构建已通过。