<template>
  <div id="app-root">
    <pre v-if="error">{{ error }}</pre>
    <div v-else-if="state" class="layout">
      <div class="main-area">
        <BuildingView
          @update:selected-elevator-id="onSelectedChange"
        />
      </div>
      <div class="side-panel">
        <!-- 第 5 步替换为 ControlPanel -->
          <p>选中：{{ selectedElevatorId ? '#' + selectedElevatorId : '无' }}</p>
      </div>
    </div>
    <p v-else class="loading">加载中…</p>
  </div>
</template>

<script setup>
import { ref, provide, onMounted, onUnmounted } from 'vue'
import { fetchState } from './api.js'
import BuildingView from './components/BuildingView.vue'

const state = ref(null)
const selectedElevatorId = ref(null)
const error = ref(null)
let timer = null

function onSelectedChange(id) {
  selectedElevatorId.value = id
}

onMounted(() => {
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

provide('state', state)
provide('selectedElevatorId', selectedElevatorId)
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
  padding: 1rem;
}

.loading {
  padding: 2rem;
  color: #999;
}
</style>
