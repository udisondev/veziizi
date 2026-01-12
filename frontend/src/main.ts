import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { logger } from '@/utils/logger'
import { initSandboxInterceptor } from '@/sandbox/api/interceptor'
import { useOnboardingStore } from '@/stores/onboarding'
import './style.css'

// ВАЖНО: Инициализируем sandbox interceptor ДО создания Vue app
// чтобы все API запросы перехватывались с самого начала
initSandboxInterceptor()

const app = createApp(App)
const pinia = createPinia()

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

app.use(pinia)
app.use(router)

// ВАЖНО: Загружаем sandbox state ДО монтирования приложения
// чтобы isSandboxMode был установлен до того как компоненты начнут делать запросы
async function init() {
  const onboarding = useOnboardingStore()
  await onboarding.loadProgress()
  app.mount('#app')
}

init()
