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
