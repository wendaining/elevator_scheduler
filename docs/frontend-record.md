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
