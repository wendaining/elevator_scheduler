<template>
  <div class="shaft" :class="{ selected: isSelected }">
    <!-- 顶部标签 -->
    <div class="shaft-header" @click="$emit('select')">
      #{{ elevator.id }}
    </div>

    <!-- 电梯井道 -->
    <div class="shaft-track" ref="trackRef">
      <!-- 楼层方块 -->
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
      <div class="car-wrapper" :style="carStyle">
        <div class="elevator-car" :class="carStateClass">
          <span class="car-direction">{{ directionSymbol }}</span>
        </div>
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

// 电梯位置（含 tick 间平滑偏移）
const carStyle = computed(() => {
  const e = props.elevator
  const n = props.floorCount

  let effective = e.currentFloor
  if (e.moveRemainingTicks > 0 && e.direction !== 'idle') {
    const progress = 1 - e.moveRemainingTicks / props.ticksPerFloor
    effective += e.direction === 'up' ? progress : -progress
  }
  effective = Math.max(1, Math.min(n, effective))

  return {
    top: ((n - effective) / n) * 100 + '%',
    height: (100 / n) + '%',
  }
})

// 状态颜色类名
const carStateClass = computed(() => {
  const e = props.elevator
  if (e.doorOpen) return 'state-open'
  if (e.direction !== 'idle') return 'state-moving'
  return 'state-idle'
})

// 方向箭头（电梯运行时显示）
const directionSymbol = computed(() => {
  const e = props.elevator
  if (e.direction === 'up') return '▲'
  if (e.direction === 'down') return '▼'
  return ''
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
  background:
    repeating-linear-gradient(
      to bottom,
      #fafafa 0px,
      #fafafa calc(100% / attr(data-rows, 20) - 1px),
      #eee calc(100% / attr(data-rows, 20) - 1px),
      #eee calc(100% / attr(data-rows, 20))
    );
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
  opacity: 0.2;
  cursor: default;
}

/* ── 电梯轿厢 wrapper ── */
.car-wrapper {
  position: absolute;
  left: 28%;
  right: 8%;
  display: flex;
  align-items: center;
  z-index: 2;
  transition: top 0.45s ease;
}

.elevator-car {
  width: 100%;
  height: 92%;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.5s, box-shadow 0.5s, border-color 0.5s;
}

.car-direction {
  font-size: 14px;
  font-weight: 600;
  pointer-events: none;
}

/* 空闲 — 绿色半透明 */
.state-idle {
  background: rgba(103, 194, 58, 0.25);
  border: 1.5px solid rgba(103, 194, 58, 0.45);
  box-shadow: 0 0 12px rgba(103, 194, 58, 0.15);
  color: rgba(103, 194, 58, 0.9);
}

/* 移动 — 黄色半透明 */
.state-moving {
  background: rgba(230, 162, 60, 0.3);
  border: 1.5px solid rgba(230, 162, 60, 0.5);
  box-shadow: 0 0 16px rgba(230, 162, 60, 0.2);
  color: rgba(230, 162, 60, 0.95);
}

/* 开门 — 红色半透明 */
.state-open {
  background: rgba(245, 108, 108, 0.3);
  border: 1.5px solid rgba(245, 108, 108, 0.5);
  box-shadow: 0 0 18px rgba(245, 108, 108, 0.25);
  color: rgba(245, 108, 108, 0.95);
}

/* 停靠数量角标 */
.car-badge {
  position: absolute;
  top: -4px;
  right: -6px;
  background: #909399;
  color: #fff;
  font-size: 10px;
  border-radius: 50%;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 3;
}
</style>
