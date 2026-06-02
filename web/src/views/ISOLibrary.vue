<template>
  <div>
    <h1 style="margin-bottom:20px;">ISO Library</h1>

    <div class="card">
      <div class="drop-zone" :class="{ dragging }" @dragover.prevent="dragging=true" @dragleave="dragging=false" @drop.prevent="handleDrop" @click="$refs.fileInput.click()">
        <span class="mdi" :class="uploading ? 'mdi-loading mdi-spin' : 'mdi-cloud-upload'"></span>
        <p v-if="!uploading">Drag and drop an ISO file here, or click to browse</p>
        <p v-else>Uploading... {{ uploadProgress }}%</p>
      </div>
      <div v-if="uploading" class="progress-bar"><div class="progress-bar-fill" :style="{ width: uploadProgress+'%' }"></div></div>
      <div v-if="uploadError" style="color:#dc3545;margin-top:8px;">{{ uploadError }}</div>
      <input ref="fileInput" type="file" accept=".iso" style="display:none" @change="handleFileSelect" />
    </div>

    <div style="display:flex;gap:12px;margin-bottom:16px;">
      <input v-model="search" placeholder="Filter by name or distro..." style="max-width:300px;" />
      <select v-model="statusFilter" style="max-width:160px;">
        <option value="">All Statuses</option>
        <option value="ready">Ready</option><option value="processing">Processing</option>
        <option value="error">Error</option><option value="unknown">Unknown</option>
      </select>
    </div>

    <div class="card">
      <div v-if="loading" style="text-align:center;padding:20px;color:#888;"><span class="spinner"></span> Loading...</div>
      <div v-else-if="fetchError" style="color:#dc3545;text-align:center;padding:20px;">{{ fetchError }}</div>
      <table v-else-if="filteredIsos.length">
        <thead><tr><th>Name</th><th>Distro</th><th>Version</th><th>Size</th><th>Arch</th><th>Status</th><th>Actions</th></tr></thead>
        <tbody>
          <tr v-for="iso in filteredIsos" :key="iso.id">
            <td>{{ iso.name }}</td><td>{{ iso.distro || '-' }}</td><td>{{ iso.version || '-' }}</td>
            <td>{{ formatBytes(iso.size) }}</td><td>{{ iso.architecture || '-' }}</td>
            <td><span :class="statusClass(iso.status)">{{ iso.status }}</span></td>
            <td><button class="btn btn-sm btn-danger" @click="deleteIso(iso)" title="Delete">🗑</button></td>
          </tr>
        </tbody>
      </table>
      <p v-else style="color:#888;text-align:center;padding:20px;">{{ isos.length ? 'No ISOs match the filter.' : 'No ISOs yet. Upload one above!' }}</p>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { apiGet, apiUpload, apiDelete } from '../api.js'

export default {
  name: 'ISOLibrary',
  setup() {
    const isos = ref([])
    const search = ref(''); const statusFilter = ref('')
    const dragging = ref(false); const uploading = ref(false)
    const uploadProgress = ref(0); const uploadError = ref('')
    const loading = ref(true)
    const fetchError = ref('')

    const filteredIsos = computed(() => {
      let list = isos.value
      if (search.value) { const q=search.value.toLowerCase(); list=list.filter(i=>(i.name||'').toLowerCase().includes(q)||(i.distro||'').toLowerCase().includes(q)) }
      if (statusFilter.value) list=list.filter(i=>i.status===statusFilter.value)
      return list
    })

    function formatBytes(b) { if(!b) return '0 B'; const u=['B','KB','MB','GB','TB']; let i=0; while(b>=1024&&i<u.length-1){b/=1024;i++} return b.toFixed(i?1:0)+' '+u[i] }
    function statusClass(s) { return 'badge badge-'+(s||'unknown') }

    async function load() { loading.value=true; fetchError.value=''; try { isos.value = await apiGet('/isos') } catch(e) { fetchError.value = e.message || 'Failed to load' } finally { loading.value=false } }

    async function doUpload(file) {
      uploadError.value=''; uploading.value=true; uploadProgress.value=0
      try { await apiUpload('/isos/upload', file, (p)=>uploadProgress.value=p); await load() }
      catch(e) { uploadError.value = e.message||'Upload failed' }
      finally { uploading.value=false }
    }

    function handleDrop(e) { dragging.value=false; const f=e.dataTransfer?.files?.[0]; if(f) doUpload(f) }
    function handleFileSelect(e) { const f=e.target?.files?.[0]; if(f) doUpload(f) }
    async function deleteIso(iso) { if(!confirm('Delete '+iso.name+'?')) return; try { await apiDelete('/isos/'+iso.id); await load() } catch {} }

    onMounted(load)
    return { isos, search, statusFilter, dragging, uploading, uploadProgress, uploadError, loading, fetchError, filteredIsos, formatBytes, statusClass, handleDrop, handleFileSelect, deleteIso }
  }
}
</script>
