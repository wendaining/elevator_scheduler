<template>
  <aside class="control-panel">
    <section class="panel-section overview">
      <div>
        <div class="section-label">当前 Tick</div>
        <div class="tick-value">{{ state.currentTick }}</div>
      </div>
      <div class="meta-grid">
        <div>
          <span>楼层</span>
          <strong>{{ state.floorCount }}</strong>
        </div>
        <div>
          <span>电梯</span>
          <strong>{{ state.elevators.length }}</strong>
        </div>
      </div>
    </section>

    <section class="panel-section">
      <div class="section-title">系统配置</div>

      <div class="field">
        <div class="field-header">
          <span>楼层数</span>
          <strong>{{ floorDraft }}</strong>
        </div>
        <el-slider
          v-model="floorDraft"
          :min="2"
          :max="40"
          :step="1"
          :show-tooltip="false"
          @change="submitFloorCount"
        />
      </div>

      <div class="field">
        <div class="field-header">
          <span>电梯数</span>
          <strong>{{ elevatorDraft }}</strong>
        </div>
        <el-slider
          v-model="elevatorDraft"
          :min="1"
          :max="10"
          :step="1"
          :show-tooltip="false"
          @change="submitElevatorCount"
        />
      </div>
    </section>

    <section class="panel-section">
      <div class="section-title">调度算法</div>
      <el-select
        v-model="schedulerDraft"
        class="full-width"
        @change="submitScheduler"
      >
        <el-option
          v-for="item in schedulerOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </el-select>
    </section>

    <section class="panel-section">
      <div class="section-title">轿厢内请求</div>
      <div v-if="selectedElevator" class="cabin-card">
        <div class="cabin-selected">
          <span>已选中</span>
          <strong>#{{ selectedElevator.id }}</strong>
        </div>
        <div class="field">
          <div class="field-header">
            <span>目标楼层</span>
            <strong>{{ cabinFloor }}</strong>
          </div>
          <el-slider
            v-model="cabinFloor"
            :min="1"
            :max="state.floorCount"
            :step="1"
            :show-tooltip="false"
          />
        </div>
        <el-button
          type="primary"
          class="full-width"
          :loading="submittingCabin"
          @click="submitCabinRequest"
        >
          发送 Cabin 请求
        </el-button>
      </div>
      <div v-else class="empty-cabin">
        点击电梯顶部标签以发起轿厢内请求
      </div>
    </section>

    <LogTerminal :logs="logs" />
  </aside>
</template>

<script setup>
import { computed, inject, ref, watch } from 'vue'
import {
  createRequest,
  setElevatorCount,
  setFloorCount,
  setScheduler,
} from '../api.js'
import LogTerminal from './LogTerminal.vue'

const stateRef = inject('state')

const props = defineProps({
  selectedElevatorId: { type: Number, default: null },
})

const emit = defineEmits(['clear-selection'])

const state = computed(() => stateRef.value)
const floorDraft = ref(2)
const elevatorDraft = ref(1)
const schedulerDraft = ref('first-available')
const cabinFloor = ref(1)
const submittingCabin = ref(false)
const logs = ref([])
let nextLogID = 1

const schedulerOptions = [
  { value: 'first-available', label: '优先空闲算法' },
  { value: 'nearest-idle', label: '最近优先算法' },
  { value: 'fcfs', label: '先来先服务算法' },
  { value: 'scan', label: 'SCAN 电梯算法' },
  { value: 'look', label: 'LOOK 电梯算法' },
]

const selectedElevator = computed(() => {
  return state.value.elevators.find((e) => e.id === props.selectedElevatorId) || null
})

watch(
  state,
  (next) => {
    if (!next) return
    floorDraft.value = next.floorCount
    elevatorDraft.value = next.elevators.length
    schedulerDraft.value = next.schedulerName
    cabinFloor.value = Math.min(Math.max(cabinFloor.value, 1), next.floorCount)
  },
  { immediate: true },
)

watch(
  () => props.selectedElevatorId,
  () => {
    if (selectedElevator.value) {
      cabinFloor.value = selectedElevator.value.currentFloor
    }
  },
)

function appendLog(text) {
  logs.value.unshift({
    id: nextLogID++,
    tick: state.value.currentTick,
    text,
  })
  logs.value = logs.value.slice(0, 24)
}

async function submitFloorCount(value) {
  try {
    await setFloorCount(value)
    appendLog(`楼层数调整为 ${value}`)
  } catch (err) {
    appendLog(`楼层数调整失败：${err.message}`)
  }
}

async function submitElevatorCount(value) {
  try {
    await setElevatorCount(value)
    appendLog(`电梯数调整为 ${value}`)
  } catch (err) {
    appendLog(`电梯数调整失败：${err.message}`)
  }
}

async function submitScheduler(value) {
  const label = schedulerOptions.find((item) => item.value === value)?.label || value
  try {
    await setScheduler(value)
    appendLog(`调度算法切换为 ${label}`)
  } catch (err) {
    appendLog(`调度算法切换失败：${err.message}`)
  }
}

async function submitCabinRequest() {
  if (!selectedElevator.value) return

  submittingCabin.value = true
  try {
    await createRequest(cabinFloor.value, 'idle', 'cabin')
    appendLog(`#${selectedElevator.value.id} cabin 请求：${cabinFloor.value} 楼`)
    emit('clear-selection')
  } catch (err) {
    appendLog(`cabin 请求失败：${err.message}`)
  } finally {
    submittingCabin.value = false
  }
}
</script>

<style scoped>
.control-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: auto;
  padding-right: 2px;
}

.panel-section {
  flex-shrink: 0;
  border: 1px solid #dfe3e8;
  border-radius: 6px;
  padding: 12px;
  background: #fff;
}

.overview {
  display: flex;
  align-items: stretch;
  justify-content: space-between;
  gap: 12px;
}

.section-label {
  font-size: 12px;
  color: #8a94a3;
  margin-bottom: 4px;
}

.tick-value {
  font-size: clamp(28px, 4vw, 34px);
  line-height: 1;
  font-weight: 700;
  color: #111827;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  min-width: 110px;
}

.meta-grid div {
  border: 1px solid #edf0f3;
  background: #f8fafc;
  border-radius: 5px;
  padding: 7px 8px;
}

.meta-grid span {
  display: block;
  color: #9aa3af;
  font-size: 11px;
}

.meta-grid strong {
  color: #111827;
  font-size: 18px;
}

.section-title {
  font-size: 13px;
  font-weight: 700;
  color: #111827;
  margin-bottom: 12px;
}

.field + .field {
  margin-top: 12px;
}

.field-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: #6b7280;
  margin-bottom: 2px;
}

.field-header strong {
  color: #111827;
}

.full-width {
  width: 100%;
}

:deep(.el-slider__bar) {
  background-color: #111827;
}

:deep(.el-slider__button) {
  border-color: #111827;
}

:deep(.el-button--primary) {
  --el-button-bg-color: #111827;
  --el-button-border-color: #111827;
  --el-button-hover-bg-color: #374151;
  --el-button-hover-border-color: #374151;
  --el-button-active-bg-color: #030712;
  --el-button-active-border-color: #030712;
}

.cabin-card {
  border: 1px solid #d8e9ff;
  background: #f7fbff;
  border-radius: 6px;
  padding: 10px;
}

.cabin-selected {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  margin-bottom: 8px;
  color: #5b6675;
}

.cabin-selected strong {
  font-size: 18px;
  color: #1f5f99;
}

.empty-cabin {
  min-height: 88px;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  color: #9aa3af;
  font-size: 13px;
  line-height: 1.6;
  border: 1px dashed #d6dbe2;
  border-radius: 6px;
  background: #fafafa;
}

@media (max-width: 900px) {
  .control-panel {
    height: auto;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    align-items: start;
    overflow: visible;
    padding-right: 0;
  }

  .overview {
    grid-column: 1 / -1;
  }
}

@media (max-width: 620px) {
  .control-panel {
    grid-template-columns: 1fr;
  }

  .overview {
    flex-direction: column;
  }
}
</style>
