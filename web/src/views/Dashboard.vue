<template>
  <div>
    <h1 style="margin-bottom:20px;">Dashboard</h1>
    <div class="dashboard-grid">
      <div class="stat-card"><div class="stat-value">{{ isos.length }}</div><div class="stat-label">ISOs in Library</div></div>
      <div class="stat-card"><div class="stat-value">{{ readyCount }}</div><div class="stat-label">Ready to Boot</div></div>
      <div class="stat-card"><div class="stat-value">{{ formatBytes(totalSize) }}</div><div class="stat-label">Total Size</div></div>
      <div class="stat-card"><div class="stat-value">{{ uptime }}</div><div class="stat-label">Uptime</div></div>
    </div>

    <div class="card">
      <h2>Recent ISOs</h2>
      <div v-if="loading" style="text-align:center;padding:20px;color:#888;">
        <span class="spinner"></span> Loading...
      </div>
      <div v-else-if="error" style="color:#dc3545;text-align:center;padding:20px;">{{ error }}</div>
      <table v-else-if="isos.length">
        <thead><tr><th>Name</th><th>Distro</th><th>Size</th><th>Status</th></tr></thead>
        <tbody>
          <tr v-for="iso in recentIsos" :key="iso.id">
            <td>{{ iso.name }}</td><td>{{ iso.distro || '-' }}</td><td>{{ formatBytes(iso.size) }}</td>
            <td><span :class="statusClass(iso.status)">{{ iso.status }}</span></td>
          </tr>
        </tbody>
      </table>
      <p v-else style="color:#888;text-align:center;padding:20px;">No ISOs found. Upload one in the ISO Library.</p>
    </div>

    <div style="display:flex;gap:12px;">
      <button class="btn btn-primary" @click="rescan">Rescan ISOs</button>
      <button class="btn" @click="$router.push('/isos')" style="background:#e9ecef;">Go to ISO Library</button>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { apiGet, apiPost } from '../api.js'

export default {
  name: 'Dashboard',
  setup() {
    const isos = ref([])
    const uptime = ref('--')
    const loading = ref(true)
    const error = ref('')
    let uptimeTimer = null

    const readyCount = computed(() => isos.value.filter(i => i.status === 'ready').length)
    const totalSize = computed(() => isos.value.reduce((s, i) => s + (i.size || 0), 0))
    const recentIsos = computed(() => isos.value.slice(0, 5))

    function formatBytes(b) { if(!b) return '0 B'; const u=['B','KB','MB','GB','TB']; let i=0; while(b>=1024&&i<u.length-1){b/=1024;i++} return b.toFixed(i?1:0)+' '+u[i] }
    function statusClass(s) { return 'badge badge-'+(s||'unknown') }

    async function load() {
      loading.value = true
      error.value = ''
      try { isos.value = await apiGet('/isos') }
      catch(e) { error.value = e.message || 'Failed to load ISOs' }
      finally { loading.value = false }
    }
    async function rescan() { try { await apiPost('/scan', {}); await load() } catch {} }

    function tickUptime() {
      const start = window._pageLoad || Date.now()
      window._pageLoad = start
      const diff = Math.floor((Date.now()-start)/1000)
      const h=Math.floor(diff/3600), m=Math.floor((diff%3600)/60), s=diff%60
      uptime.value = h+'h '+m+'m '+s+'s'
    }

    onMounted(() => { load(); window._pageLoad=Date.now(); tickUptime(); uptimeTimer=setInterval(tickUptime,1000) })
    onUnmounted(() => clearInterval(uptimeTimer))

    return { isos, uptime, loading, error, readyCount, totalSize, recentIsos, formatBytes, statusClass, rescan }
  }
}
</script>
