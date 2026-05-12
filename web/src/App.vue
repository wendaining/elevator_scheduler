<template>
  <div id="app-root">
    <pre v-if="error">{{ error }}</pre>
    <template v-else-if="state">
      <header class="app-header">
        <h1>电梯调度算法可视化程序</h1>
      </header>
      <div class="layout">
        <div class="main-area">
          <BuildingView
            :selected-elevator-id="selectedElevatorId"
            @update:selected-elevator-id="onSelectedChange"
          />
        </div>
        <div class="side-panel">
          <ControlPanel
            :selected-elevator-id="selectedElevatorId"
            @clear-selection="selectedElevatorId = null"
          />
        </div>
      </div>
    </template>
    <p v-else class="loading">加载中…</p>
  </div>
</template>

<script setup>
import { ref, provide, onMounted, onUnmounted } from 'vue'
import { fetchConfig, fetchState } from './api.js'
import BuildingView from './components/BuildingView.vue'
import ControlPanel from './components/ControlPanel.vue'

const state = ref(null)
const selectedElevatorId = ref(null)
const error = ref(null)
const config = ref(null)
let timer = null

function onSelectedChange(id) {
  selectedElevatorId.value = id
}

onMounted(() => {
  startPolling()
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
})

async function startPolling() {
  try {
    config.value = await fetchConfig()
    await tick()

    const interval = config.value.autoStepIntervalMs
    if (!Number.isFinite(interval) || interval <= 0) {
      throw new Error(`invalid autoStepIntervalMs from backend: ${interval}`)
    }
    timer = setInterval(tick, interval)
  } catch (err) {
    error.value = err.message
  }
}

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
provide('config', config)
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
  --el-color-primary: #111827;
  --el-color-primary-light-3: #374151;
  --el-color-primary-light-5: #6b7280;
  --el-color-primary-light-7: #d1d5db;
  --el-color-primary-light-8: #e5e7eb;
  --el-color-primary-light-9: #f3f4f6;
  --el-color-primary-dark-2: #030712;
  --el-border-radius-base: 6px;
  --el-text-color-primary: #111827;
  --el-text-color-regular: #374151;
  --el-border-color: #d1d5db;
  --el-fill-color-light: #f3f4f6;

  height: 100%;
  font-family: 'PingFang SC', 'Microsoft YaHei', sans-serif;
  color: #111827;
  background: #f3f4f6;
  display: flex;
  flex-direction: column;
}

.app-header {
  height: 52px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding: 0 18px;
  background: #111827;
  color: #fff;
  border-bottom: 1px solid #030712;
}

.app-header h1 {
  font-size: 18px;
  font-weight: 700;
  line-height: 1;
  letter-spacing: 0;
}

.layout {
  display: flex;
  flex: 1;
  min-height: 0;
  min-width: 0;
}

.main-area {
  flex: 3;
  min-width: 0;
  overflow: auto;
}

.side-panel {
  flex: 1;
  min-width: 280px;
  max-width: 360px;
  background: #fff;
  border-left: 1px solid #e0e0e0;
  overflow: hidden;
  padding: 12px;
}

.loading {
  padding: 2rem;
  color: #999;
}

pre {
  padding: 16px;
  color: #991b1b;
  white-space: pre-wrap;
}

@media (max-width: 900px) {
  html, body, #app {
    overflow: auto;
  }

  .layout {
    min-height: calc(100vh - 52px);
    height: auto;
    flex-direction: column;
  }

  .main-area {
    height: 70vh;
    min-height: 520px;
  }

  .side-panel {
    flex: none;
    width: 100%;
    max-width: none;
    min-width: 0;
    border-left: 0;
    border-top: 1px solid #e0e0e0;
    overflow: visible;
  }
}
</style>
