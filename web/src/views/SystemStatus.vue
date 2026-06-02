<template>
  <div>
    <h1 style="margin-bottom:20px;">System Status</h1>

    <div class="card">
      <h2>Services</h2>
      <table>
        <thead><tr><th>Service</th><th>Status</th></tr></thead>
        <tbody>
          <tr><td>HTTP Server</td><td><span class="badge badge-ready">Running</span></td></tr>
          <tr><td>DHCP Proxy</td><td><span class="badge badge-ready">{{ dhcpStatus }}</span></td></tr>
          <tr><td>TFTP Server</td><td><span class="badge badge-ready">{{ tftpStatus }}</span></td></tr>
          <tr><td>ISO Scanner</td><td><span class="badge badge-ready">Active</span></td></tr>
        </tbody>
      </table>
    </div>

    <div class="card">
      <h2>Configuration</h2>
      <div v-if="loading" style="text-align:center;padding:20px;color:#888;">
        <span class="spinner"></span> Loading...
      </div>
      <div v-else-if="fetchError" style="color:#dc3545;text-align:center;padding:20px;">{{ fetchError }}</div>
      <table v-else-if="config">
        <tbody>
          <tr><th>Version</th><td>{{ config.version || '-' }}</td></tr>
          <tr><th>HTTP Listen</th><td>{{ config.http_listen }}</td></tr>
          <tr><th>DHCP Listen</th><td>{{ config.dhcp_listen }}</td></tr>
          <tr><th>TFTP Listen</th><td>{{ config.tftp_listen }}</td></tr>
          <tr><th>ISO Dir</th><td>{{ config.iso_dir }}</td></tr>
          <tr><th>Cache Dir</th><td>{{ config.cache_dir }}</td></tr>
        </tbody>
      </table>
      <p v-else style="color:#888;text-align:center;padding:20px;">No configuration data.</p>
    </div>

    <div class="card">
      <h2>API Token</h2>
      <div style="display:flex;gap:8px;align-items:center;">
        <input :value="tokenDisplay" readonly type="text" style="max-width:400px;font-family:monospace;" />
        <button class="btn btn-sm" @click="copyToken">{{ copied ? 'Copied' : 'Copy' }}</button>
      </div>
      <button class="btn btn-danger btn-sm" style="margin-top:12px;" @click="regenerateToken">Regenerate Token</button>
      <p v-if="tokenError" style="color:#dc3545;margin-top:8px;">{{ tokenError }}</p>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { apiGet, apiPost, getToken } from '../api.js'

export default {
  name: 'SystemStatus',
  setup() {
    const config = ref(null); const tokenDisplay = ref(getToken())
    const copied = ref(false); const tokenError = ref('')
    const dhcpStatus = ref('Running'); const tftpStatus = ref('Running')
    const loading = ref(true)
    const fetchError = ref('')

    onMounted(async () => {
      loading.value = true
      fetchError.value = ''
      try { config.value = await apiGet('/config') }
      catch(e) { fetchError.value = e.message || 'Failed to load config' }
      finally { loading.value = false }
    })

    function copyToken() { navigator.clipboard?.writeText(tokenDisplay.value); copied.value=true; setTimeout(()=>copied.value=false,2000) }
    async function regenerateToken() {
      if (!confirm('Regenerate the API token? All current sessions will need the new token.')) return
      try { const resp = await apiPost('/config/regenerate-token', {}); if (resp.token) tokenDisplay.value = resp.token; tokenError.value = '' }
      catch(e) { tokenError.value = e.message || 'Failed' }
    }
    return { config, loading, fetchError, tokenDisplay, copied, tokenError, dhcpStatus, tftpStatus, copyToken, regenerateToken }
  }
}
</script>
