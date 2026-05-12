<template>
  <div id="app-root">
    <pre v-if="error">{{ error }}</pre>
    <div v-else-if="state" class="layout">
      <div class="main-area">
        <!-- 第 3 步替换为 BuildingView -->
        <p style="padding: 2rem;">系统就绪：{{ state.floorCount }} 层 / {{ state.elevators.length }} 部电梯 / tick {{ state.currentTick }}</p>
      </div>
      <div class="side-panel">
        <!-- 第 5 步替换为 ControlPanel -->
      </div>
    </div>
    <p v-else style="padding: 2rem;">加载中…</p>
  </div>
</template>

<script setup>
import { ref, provide, onMounted, onUnmounted } from 'vue'
import { fetchState } from './api.js'

const state = ref(null)
const error = ref(null)
let timer = null

onMounted(() => {
  // 立即获取一次，然后每 500ms 轮询
  tick()
  timer = setInterval(tick, 500)
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
})

async function tick() {
  try {
    state.value = await fetchState()
    error.value = null
  } catch (err) {
    error.value = err.message
  }
}

// 把 state 和 API 方法提供给后代组件
provide('state', state)
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  width: 100%;
  overflow: hidden;
}

#app-root {
  height: 100%;
  font-family: 'PingFang SC', 'Microsoft YaHei', sans-serif;
  color: #333;
  background: #f5f5f5;
}

.layout {
  display: flex;
  height: 100%;
}

.main-area {
  flex: 3;
  overflow: hidden;
}

.side-panel {
  flex: 1;
  min-width: 280px;
  max-width: 360px;
  background: #fff;
  border-left: 1px solid #e0e0e0;
  overflow-y: auto;
}
</style>
