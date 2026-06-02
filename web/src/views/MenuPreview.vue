<template>
  <div>
    <h1 style="margin-bottom:12px;">Boot Menu Preview</h1>
    <p style="color:#666;margin-bottom:16px;">This is what clients see when they network-boot.</p>
    <div class="card">
      <div v-if="loading" style="text-align:center;padding:20px;color:#888;">
        <span class="spinner"></span> Loading...
      </div>
      <div v-else-if="error" style="color:#dc3545;text-align:center;padding:20px;">{{ error }}</div>
      <pre v-else class="log-viewer"><code>{{ menuText || 'No menu data.' }}</code></pre>
    </div>
    <button class="btn btn-primary" @click="refresh">Refresh</button>
    <span style="margin-left:12px;font-size:13px;color:#666;">Base URL: {{ baseUrl }}</span>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { apiGet } from '../api.js'

export default {
  name: 'MenuPreview',
  setup() {
    const menuText = ref(''); const baseUrl = ref('')
    const loading = ref(true)
    const error = ref('')
    async function refresh() {
      loading.value = true
      error.value = ''
      try {
        const data = await apiGet('/boot/menu')
        menuText.value = data.script || data
        baseUrl.value = data.base_url || ''
      } catch(e) {
        error.value = e.message || 'Failed to load boot menu'
      } finally {
        loading.value = false
      }
    }
    onMounted(refresh)
    return { menuText, baseUrl, loading, error, refresh }
  }
}
</script>
