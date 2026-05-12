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
import { computed, inject } from 'vue'
import ElevatorShaft from './ElevatorShaft.vue'
import { createRequest } from '../api.js'

const state = inject('state')

const props = defineProps({
  selectedElevatorId: { type: Number, default: null },
})

const emit = defineEmits(['update:selectedElevatorId'])

function onSelect(id) {
  const next = props.selectedElevatorId === id ? null : id
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
  min-width: max-content;
  background: #f8fafc;
}

.floor-labels {
  width: 34px;
  display: flex;
  flex-direction: column;
  background: #f3f4f6;
  border-right: 1px solid #d1d5db;
  flex-shrink: 0;
  position: sticky;
  left: 0;
  z-index: 4;
}

.floor-label-spacer {
  height: 28px;
  border-bottom: 1px solid #d1d5db;
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
  color: #8b95a1;
  border-bottom: 1px solid #e5e7eb;
  box-sizing: border-box;
}

@media (max-width: 900px) {
  .building {
    min-width: 760px;
  }
}
</style>
