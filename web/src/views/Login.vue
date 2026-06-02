<template>
  <div class="login-page">
    <div class="login-box">
      <h1>&#x1F4A1; LightBoot</h1>
      <p class="subtitle">Enter your API token to continue</p>
      <form @submit.prevent="doLogin">
        <input v-model="token" type="password" placeholder="API Token" autofocus />
        <div v-if="error" class="login-error">{{ error }}</div>
        <button type="submit" class="btn btn-primary" style="width:100%;justify-content:center;">
          {{ loading ? 'Verifying...' : 'Login' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { setToken, apiGet } from '../api.js'

export default {
  name: 'Login',
  setup() {
    const router = useRouter()
    const token = ref('')
    const error = ref('')
    const loading = ref(false)

    async function doLogin() {
      error.value = ''
      if (!token.value.trim()) { error.value = 'Token is required'; return }
      loading.value = true
      setToken(token.value.trim())
      try { await apiGet('/health'); router.push('/') }
      catch { error.value = 'Invalid token. Try again.'; setToken('') }
      finally { loading.value = false }
    }
    return { token, error, loading, doLogin }
  }
}
</script>
