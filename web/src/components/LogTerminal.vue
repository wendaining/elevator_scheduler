<template>
  <section class="terminal">
    <div class="terminal-header">
      <span>事件日志</span>
      <span class="terminal-count">{{ logs.length }}</span>
    </div>
    <div class="terminal-body">
      <div v-if="logs.length === 0" class="empty-line">
        暂无事件
      </div>
      <div
        v-for="entry in logs"
        :key="entry.id"
        class="log-line"
      >
        <span class="log-time">T{{ entry.tick }}</span>
        <span class="log-text">{{ entry.text }}</span>
      </div>
    </div>
  </section>
</template>

<script setup>
defineProps({
  logs: { type: Array, default: () => [] },
})
</script>

<style scoped>
.terminal {
  flex-shrink: 0;
  border: 1px solid #20252b;
  background: #111418;
  color: #d8dde4;
  border-radius: 6px;
  overflow: hidden;
}

.terminal-header {
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 10px;
  background: #1b2027;
  border-bottom: 1px solid #2b323b;
  font-size: 12px;
  font-weight: 600;
}

.terminal-count {
  color: #8b96a5;
  font-weight: 500;
}

.terminal-body {
  height: clamp(120px, 20vh, 180px);
  overflow: auto;
  padding: 8px 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  line-height: 1.65;
}

.empty-line {
  color: #6f7a87;
}

.log-line {
  display: flex;
  gap: 8px;
  white-space: nowrap;
}

.log-time {
  color: #86efac;
  flex-shrink: 0;
}

.log-text {
  color: #d8dde4;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (max-width: 900px) {
  .terminal {
    min-height: 0;
  }
}
</style>
