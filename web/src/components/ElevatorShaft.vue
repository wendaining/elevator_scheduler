<template>
  <div class="shaft" :class="{ selected: isSelected }">
    <!-- 顶部标签 -->
    <div class="shaft-header" @click="$emit('select')">
      #{{ elevator.id }}
    </div>

    <!-- 电梯井道 -->
    <div class="shaft-track" :style="trackStyle">
      <!-- 楼层方块 -->
      <div
        v-for="floor in floors"
        :key="floor.num"
        class="floor-block"
        :class="floorStateClass(floor.num)"
      >
        <span v-if="isCurrentFloor(floor.num)" class="floor-status">
          {{ directionSymbol }}
        </span>
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
        <span v-if="isCurrentFloor(floor.num) && elevator.stops.length" class="stop-badge">
          {{ elevator.stops.length }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  elevator: { type: Object, required: true },
  floorCount: { type: Number, required: true },
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

// 用 CSS Grid 均分楼层高度，避免手动测量 DOM 和计算轿厢绝对位置。
const trackStyle = computed(() => {
  return {
    gridTemplateRows: `repeat(${props.floorCount}, minmax(0, 1fr))`,
  }
})

function isCurrentFloor(floor) {
  return floor === props.elevator.currentFloor
}

// 当前楼层格子的状态颜色。
// 不再渲染额外的轿厢方块，而是直接用楼层格子表示电梯所在位置。
function floorStateClass(floor) {
  const e = props.elevator
  if (floor !== e.currentFloor) return ''
  if (e.doorOpen) return 'is-current state-open'
  if (e.direction !== 'idle') return 'is-current state-moving'
  return 'is-current state-idle'
}

// 当前楼层格子中的方向标识。
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
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
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
  display: grid;
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
  min-height: 0;
  background: #fafafa;
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
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

.floor-status {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 13px;
  font-weight: 700;
  pointer-events: none;
}

.floor-block.is-current {
  border-bottom-color: transparent;
  z-index: 1;
}

/* 空闲 — 当前楼层绿色高亮 */
.floor-block.state-idle {
  background: rgba(103, 194, 58, 0.24);
  border: 1px solid rgba(103, 194, 58, 0.55);
  box-shadow: inset 0 0 0 1px rgba(103, 194, 58, 0.16);
}

/* 移动 — 当前楼层黄色高亮 */
.floor-block.state-moving {
  background: rgba(230, 162, 60, 0.28);
  border: 1px solid rgba(230, 162, 60, 0.58);
  box-shadow: inset 0 0 0 1px rgba(230, 162, 60, 0.18);
}

/* 开门 — 当前楼层红色高亮 */
.floor-block.state-open {
  background: rgba(245, 108, 108, 0.26);
  border: 1px solid rgba(245, 108, 108, 0.58);
  box-shadow: inset 0 0 0 1px rgba(245, 108, 108, 0.18);
}

.floor-block.state-idle .floor-status {
  color: rgba(103, 194, 58, 0.95);
}

.floor-block.state-moving .floor-status {
  color: rgba(181, 112, 18, 0.95);
}

.floor-block.state-open .floor-status {
  color: rgba(207, 75, 75, 0.95);
}

/* 停靠数量角标 */
.stop-badge {
  position: absolute;
  top: 4px;
  right: 4px;
  background: #909399;
  color: #fff;
  font-size: 10px;
  border-radius: 50%;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
}
</style>
