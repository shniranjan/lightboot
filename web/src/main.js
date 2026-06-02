import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import '@mdi/font/css/materialdesignicons.css'
import App from './App.vue'
import Login from './views/Login.vue'
import Dashboard from './views/Dashboard.vue'
import ISOLibrary from './views/ISOLibrary.vue'
import MenuPreview from './views/MenuPreview.vue'
import LogsView from './views/LogsView.vue'
import SystemStatus from './views/SystemStatus.vue'

const routes = [
  { path: '/login', name: 'Login', component: Login, meta: { noAuth: true } },
  { path: '/', name: 'Dashboard', component: Dashboard },
  { path: '/isos', name: 'ISOLibrary', component: ISOLibrary },
  { path: '/menu', name: 'MenuPreview', component: MenuPreview },
  { path: '/logs', name: 'LogsView', component: LogsView },
  { path: '/status', name: 'SystemStatus', component: SystemStatus },
]

const router = createRouter({ history: createWebHashHistory(), routes })
const app = createApp(App)
app.use(router)
app.mount('#app')
