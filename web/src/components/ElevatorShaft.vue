<template>
  <div class="shaft" :class="{ selected: isSelected }">
    <!-- 顶部标签：点击选中/取消 -->
    <div class="shaft-header" @click="$emit('select')">
      #{{ elevator.id }}
    </div>

    <!-- 电梯井道 -->
    <div class="shaft-track" ref="trackRef">
      <!-- 楼层方块（从顶层到底层排列） -->
      <div
        v-for="floor in floors"
        :key="floor.num"
        class="floor-block"
        :style="{ height: blockHeight + 'px' }"
      >
        <span class="floor-label">{{ floor.num }}</span>
        <button
          class="floor-btn up"
          :disabled="floor.atTop"
          title="上行"
          @click.stop="$emit('hall-request', floor.num, 'up')"
        >▲</button>
        <button
          class="floor-btn down"
          :disabled="floor.atBottom"
          title="下行"
          @click.stop="$emit('hall-request', floor.num, 'down')"
        >▼</button>
      </div>

      <!-- 电梯轿厢 -->
      <div
        class="elevator-car"
        :style="carStyle"
      >
        <span v-if="elevator.stops.length" class="car-badge">{{ elevator.stops.length }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  elevator: { type: Object, required: true },
  floorCount: { type: Number, required: true },
  ticksPerFloor: { type: Number, default: 5 },
  isSelected: { type: Boolean, default: false },
})

defineEmits(['select', 'hall-request'])

// 楼层列表：顶楼 → 底楼（楼层号从大到小）
const floors = computed(() => {
  const list = []
  for (let f = props.floorCount; f >= 1; f--) {
    list.push({
      num: f,
      atTop: f === props.floorCount,
      atBottom: f === 1,
    })
  }
  return list
})

// 动态方块高度 — 填满轨道高度
const trackRef = ref(null)
const blockHeight = ref(80)

function recalcBlockHeight() {
  if (trackRef.value) {
    blockHeight.value = trackRef.value.clientHeight / props.floorCount
  }
}

onMounted(() => {
  recalcBlockHeight()
  window.addEventListener('resize', recalcBlockHeight)
})

onUnmounted(() => {
  window.removeEventListener('resize', recalcBlockHeight)
})

// 电梯轿厢的定位
const carStyle = computed(() => {
  const e = props.elevator
  const n = props.floorCount

  // 计算有效楼层（含 tick 间的平滑偏移）
  let effective = e.currentFloor
  if (e.moveRemainingTicks > 0 && e.direction !== 'idle') {
    const progress = 1 - (e.moveRemainingTicks / props.ticksPerFloor)
    effective += e.direction === 'up' ? progress : -progress
  }
  effective = Math.max(1, Math.min(n, effective))

  const topPct = ((n - effective) / n) * 100

  return {
    top: topPct + '%',
    height: (100 / n) + '%',
  }
})
</script>

<style scoped>
.shaft {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 80px;
  border-right: 1px solid #ddd;
}

.shaft.selected .shaft-header {
  background: #409eff;
  color: #fff;
}

.shaft-header {
  text-align: center;
  padding: 4px 0;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  user-select: none;
  background: #eee;
  border-bottom: 1px solid #ddd;
  transition: background 0.2s, color 0.2s;
}

.shaft-header:hover {
  background: #d0d0d0;
}

.shaft-track {
  flex: 1;
  position: relative;
  overflow: hidden;
  background: #fafafa;
}

/* 楼层方块 */
.floor-block {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
  position: relative;
  border-bottom: 1px solid #eee;
  box-sizing: border-box;
}

.floor-label {
  position: absolute;
  left: 4px;
  font-size: 10px;
  color: #bbb;
  pointer-events: none;
}

.floor-btn {
  width: 24px;
  height: 24px;
  border: 1px solid #ccc;
  border-radius: 3px;
  background: #fff;
  cursor: pointer;
  font-size: 11px;
  line-height: 1;
  padding: 0;
  color: #555;
  transition: background 0.15s;
}

.floor-btn:hover:not(:disabled) {
  background: #e8e8e8;
}

.floor-btn:disabled {
  opacity: 0.25;
  cursor: default;
}

.floor-btn.up  { color: #67c23a; }
.floor-btn.down { color: #f56c6c; }

/* 电梯轿厢 */
.elevator-car {
  position: absolute;
  left: 30%;
  right: 10%;
  background: #909399;
  border-radius: 3px;
  border: 1px solid #606266;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
  transition: top 0.4s ease;
  min-height: 30px;
}

.car-badge {
  background: #e6a23c;
  color: #fff;
  font-size: 10px;
  border-radius: 50%;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
