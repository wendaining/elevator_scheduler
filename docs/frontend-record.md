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
