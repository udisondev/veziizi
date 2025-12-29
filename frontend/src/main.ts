import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { logger } from '@/utils/logger'
import './style.css'

const app = createApp(App)

// Global error handler
app.config.errorHandler = (err, instance, info) => {
  logger.error('Vue error', {
    error: err,
    component: instance?.$options?.name,
    info,
  })
}

// Warning handler (development only)
if (import.meta.env.DEV) {
  app.config.warnHandler = (msg, instance, trace) => {
    logger.warn('Vue warning', { msg, component: instance?.$options?.name, trace })
  }
}

app.use(createPinia())
app.use(router)

app.mount('#app')
