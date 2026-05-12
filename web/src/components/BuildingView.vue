<template>
  <div class="building">
    <!-- 楼层标签列 -->
    <div class="floor-labels">
      <div class="floor-label-spacer"></div>
      <div class="floor-label-track" :style="labelTrackStyle">
        <div
          v-for="f in floorLabels"
          :key="f"
          class="floor-label-item"
        >
          {{ f }}
        </div>
      </div>
    </div>

    <!-- 电梯井道 -->
    <ElevatorShaft
      v-for="elevator in state.elevators"
      :key="elevator.id"
      :elevator="elevator"
      :floor-count="state.floorCount"
      :is-selected="selectedElevatorId === elevator.id"
      @select="onSelect(elevator.id)"
      @hall-request="onHallRequest"
    />
  </div>
</template>

<script setup>
import { computed, inject, ref } from 'vue'
import ElevatorShaft from './ElevatorShaft.vue'
import { createRequest } from '../api.js'

const state = inject('state')

// 选中的电梯 ID — 同时在本地和向上 emit
const selectedElevatorId = ref(null)
const emit = defineEmits(['update:selectedElevatorId'])

function onSelect(id) {
  const next = selectedElevatorId.value === id ? null : id
  selectedElevatorId.value = next
  emit('update:selectedElevatorId', next)
}

// 楼层标签：顶楼到底楼
const floorLabels = computed(() => {
  if (!state.value) return []
  const arr = []
  for (let f = state.value.floorCount; f >= 1; f--) arr.push(f)
  return arr
})

const labelTrackStyle = computed(() => {
  if (!state.value) return {}
  return {
    gridTemplateRows: `repeat(${state.value.floorCount}, minmax(0, 1fr))`,
  }
})

// hall 请求
async function onHallRequest(floor, direction) {
  try {
    await createRequest(floor, direction, 'hall')
  } catch (err) {
    console.error(err)
  }
}
</script>

<style scoped>
.building {
  display: flex;
  height: 100%;
}

.floor-labels {
  width: 32px;
  display: flex;
  flex-direction: column;
  background: #f0f0f0;
  border-right: 1px solid #ddd;
  flex-shrink: 0;
}

.floor-label-spacer {
  height: 28px;
  border-bottom: 1px solid #ddd;
  box-sizing: border-box;
}

.floor-label-track {
  flex: 1;
  display: grid;
  min-height: 0;
}

.floor-label-item {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  color: #999;
  border-bottom: 1px solid #eee;
  box-sizing: border-box;
}
</style>
