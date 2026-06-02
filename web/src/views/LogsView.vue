<template>
  <div>
    <h1 style="margin-bottom:12px;">Live Logs</h1>
    <div style="display:flex;gap:12px;margin-bottom:12px;align-items:center;">
      <select v-model="levelFilter" style="max-width:140px;">
        <option value="">All Levels</option><option value="info">Info</option>
        <option value="warn">Warning</option><option value="error">Error</option>
      </select>
      <label style="font-size:13px;display:flex;align-items:center;gap:6px;">
        <input type="checkbox" v-model="autoScroll" /> Auto-scroll
      </label>
      <button class="btn btn-sm" @click="clear">Clear</button>
      <span style="font-size:12px;color:#888;margin-left:auto;">{{ logs.length }} entries</span>
    </div>
    <div class="log-viewer" ref="logContainer">
      <div v-for="(entry, idx) in filteredLogs" :key="idx" class="log-entry" :class="'level-' + (entry.level || 'info')">
        <span class="ts">[{{ entry.timestamp || entry.time || '' }}]</span>
        <span>{{ entry.message || entry.msg || JSON.stringify(entry) }}</span>
      </div>
      <div v-if="!filteredLogs.length" style="color:#888;">Waiting for log events...</div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { createLogStream } from '../api.js'

export default {
  name: 'LogsView',
  setup() {
    const logs = ref([]); const levelFilter = ref(''); const autoScroll = ref(true)
    const logContainer = ref(null); let es = null

    const filteredLogs = computed(() => {
      if (!levelFilter.value) return logs.value
      return logs.value.filter(l => l.level === levelFilter.value)
    })

    function scrollBottom() { nextTick(() => { if (logContainer.value && autoScroll.value) logContainer.value.scrollTop = logContainer.value.scrollHeight }) }
    watch(filteredLogs, scrollBottom)

    onMounted(() => { es = createLogStream((entry)=>{logs.value.push(entry);if(logs.value.length>500)logs.value.shift()}, ()=>{}) })
    onUnmounted(() => { if(es) es.close() })

    function clear() { logs.value = [] }
    return { logs, levelFilter, autoScroll, logContainer, filteredLogs, clear }
  }
}
</script>
