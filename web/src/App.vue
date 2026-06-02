<template>
  <div id="lightboot-app">
    <nav v-if="!route.meta.noAuth" class="sidebar">
      <div class="sidebar-brand">
        <span class="brand-icon">&#x1F4A1;</span>
        <span class="brand-text">LightBoot</span>
      </div>
      <router-link to="/" class="nav-item" active-class="active">
        <span class="mdi mdi-view-dashboard"></span> Dashboard
      </router-link>
      <router-link to="/isos" class="nav-item" active-class="active">
        <span class="mdi mdi-disc"></span> ISO Library
      </router-link>
      <router-link to="/menu" class="nav-item" active-class="active">
        <span class="mdi mdi-monitor-screenshot"></span> Boot Menu
      </router-link>
      <router-link to="/logs" class="nav-item" active-class="active">
        <span class="mdi mdi-text-box-outline"></span> Logs
      </router-link>
      <router-link to="/status" class="nav-item" active-class="active">
        <span class="mdi mdi-cog"></span> System
      </router-link>
      <div class="sidebar-footer">
        <a href="/docs/" target="_blank" class="nav-item">
          <span class="mdi mdi-help-circle"></span> Help
        </a>
        <a href="#" @click.prevent="logout" class="nav-item">
          <span class="mdi mdi-logout"></span> Logout
        </a>
      </div>
    </nav>
    <main :class="{ 'full-width': route.meta.noAuth }">
      <router-view />
    </main>
  </div>
</template>

<script>
import { useRoute } from 'vue-router'
import { clearToken } from './api.js'

export default {
  name: 'App',
  setup() {
    const route = useRoute()
    function logout() { clearToken(); window.location.hash = '#/login' }
    return { route, logout }
  }
}
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f0f2f5; color: #1a1a2e; }
#lightboot-app { display: flex; min-height: 100vh; }
.sidebar { width: 220px; background: linear-gradient(180deg, #1a1a2e 0%, #16213e 100%); color: #ccc; display: flex; flex-direction: column; flex-shrink: 0; }
.sidebar-brand { padding: 20px 16px; border-bottom: 1px solid rgba(255,255,255,0.08); }
.brand-icon { font-size: 24px; margin-right: 8px; }
.brand-text { font-size: 18px; font-weight: 700; color: #fff; }
.nav-item { display: flex; align-items: center; gap: 10px; padding: 12px 20px; color: #aab; text-decoration: none; font-size: 14px; transition: all 0.2s; }
.nav-item:hover { background: rgba(255,255,255,0.06); color: #fff; }
.nav-item.active { background: rgba(0,150,255,0.15); color: #4da3ff; border-left: 3px solid #4da3ff; }
.sidebar-footer { margin-top: auto; border-top: 1px solid rgba(255,255,255,0.08); padding-top: 8px; }
main { flex: 1; padding: 24px; overflow-y: auto; }
main.full-width { padding: 0; }
.card { background: #fff; border-radius: 12px; padding: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-bottom: 20px; }
.card h2 { font-size: 18px; margin-bottom: 16px; color: #1a1a2e; }
.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 16px; border: none; border-radius: 8px; font-size: 14px; cursor: pointer; transition: all 0.2s; }
.btn-primary { background: #007aff; color: #fff; }
.btn-primary:hover { background: #0062cc; }
.btn-danger { background: #dc3545; color: #fff; }
.btn-danger:hover { background: #bb2d3b; }
.btn-sm { padding: 4px 10px; font-size: 12px; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 10px 12px; border-bottom: 1px solid #eee; font-size: 13px; }
th { font-weight: 600; color: #555; }
input, select { padding: 8px 12px; border: 1px solid #ccc; border-radius: 8px; font-size: 14px; width: 100%; }
.badge { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 11px; font-weight: 600; }
.badge-ready { background: #d4edda; color: #155724; }
.badge-error { background: #f8d7da; color: #721c24; }
.badge-processing { background: #fff3cd; color: #856404; }
.badge-unknown { background: #e2e3e5; color: #383d41; }
.login-page { display: flex; align-items: center; justify-content: center; min-height: 100vh; background: linear-gradient(135deg, #1a1a2e, #16213e); }
.login-box { background: #fff; border-radius: 16px; padding: 40px; width: 400px; max-width: 90vw; text-align: center; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.login-box h1 { font-size: 32px; margin-bottom: 8px; }
.login-box .subtitle { color: #666; margin-bottom: 24px; }
.login-box input { text-align: center; font-size: 16px; padding: 12px; margin-bottom: 16px; }
.login-error { color: #dc3545; font-size: 13px; margin-bottom: 12px; }
.drop-zone { border: 2px dashed #ccc; border-radius: 12px; padding: 40px; text-align: center; transition: all 0.3s; cursor: pointer; }
.drop-zone.dragging { border-color: #007aff; background: rgba(0,122,255,0.05); }
.drop-zone .mdi { font-size: 48px; color: #007aff; }
.progress-bar { background: #e9ecef; border-radius: 8px; height: 12px; overflow: hidden; margin-top: 12px; }
.progress-bar-fill { background: #007aff; height: 100%; transition: width 0.3s; border-radius: 8px; }
.log-viewer { background: #1e1e1e; color: #d4d4d4; border-radius: 8px; padding: 16px; max-height: 500px; overflow-y: auto; font-family: 'Courier New', monospace; font-size: 13px; }
.log-entry { padding: 2px 0; border-bottom: 1px solid rgba(255,255,255,0.04); }
.log-entry .ts { color: #6a9955; margin-right: 8px; }
.log-entry.level-info { color: #4fc3f7; }
.log-entry.level-warn { color: #ffd54f; }
.log-entry.level-error { color: #ef5350; }
.dashboard-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
.stat-card { background: #fff; border-radius: 12px; padding: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
.stat-card .stat-value { font-size: 32px; font-weight: 700; color: #1a1a2e; }
.stat-card .stat-label { font-size: 13px; color: #888; margin-top: 4px; }
.spinner { display: inline-block; width: 16px; height: 16px; border: 2px solid #ccc; border-top-color: #007aff; border-radius: 50%; animation: spin 0.6s linear infinite; vertical-align: middle; margin-right: 6px; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
