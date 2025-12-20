<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api/admin'
import type { Fraudster } from '@/types/admin'

const router = useRouter()
const admin = useAdminStore()

const fraudsters = ref<Fraudster[]>([])
const total = ref(0)
const isLoading = ref(true)
const error = ref('')

// Modal state
const showUnmarkModal = ref(false)
const selectedFraudster = ref<Fraudster | null>(null)
const unmarkReason = ref('')
const isSubmitting = ref(false)

onMounted(async () => {
  await loadFraudsters()
})

async function loadFraudsters() {
  isLoading.value = true
  error.value = ''
  try {
    const response = await adminApi.getFraudsters()
    fraudsters.value = response.fraudsters ?? []
    total.value = response.total
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function openUnmarkModal(fraudster: Fraudster) {
  selectedFraudster.value = fraudster
  unmarkReason.value = ''
  showUnmarkModal.value = true
}

function closeModal() {
  showUnmarkModal.value = false
  selectedFraudster.value = null
}

async function submitUnmark() {
  if (!selectedFraudster.value || !unmarkReason.value.trim()) return
  isSubmitting.value = true
  try {
    await adminApi.unmarkFraudster(selectedFraudster.value.org_id, {
      reason: unmarkReason.value.trim(),
    })
    closeModal()
    await loadFraudsters()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка снятия метки'
  } finally {
    isSubmitting.value = false
  }
}

async function handleLogout() {
  await admin.logout()
  router.push('/admin/login')
}
</script>

<template>
  <div class="min-h-screen bg-gray-900">
    <!-- Header -->
    <header class="bg-gray-800 shadow">
      <div class="max-w-7xl mx-auto px-4 py-4 flex justify-between items-center">
        <div class="flex items-center gap-6">
          <h1 class="text-xl font-bold text-white">Панель администратора</h1>
          <nav class="flex gap-4">
            <router-link to="/admin/organizations" class="text-gray-400 hover:text-white text-sm">
              Организации
            </router-link>
            <router-link to="/admin/reviews" class="text-gray-400 hover:text-white text-sm">
              Отзывы
            </router-link>
            <router-link to="/admin/fraudsters" class="text-white text-sm font-medium">
              Накрутчики
            </router-link>
          </nav>
        </div>
        <div class="flex items-center gap-4">
          <span class="text-gray-400 text-sm">{{ admin.email }}</span>
          <button
            @click="handleLogout"
            class="text-gray-400 hover:text-white text-sm"
          >
            Выйти
          </button>
        </div>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-7xl mx-auto px-4 py-8">
      <div class="flex justify-between items-center mb-6">
        <div>
          <h2 class="text-2xl font-bold text-white">Накрутчики</h2>
          <p class="text-gray-400 text-sm mt-1">Всего: {{ total }}</p>
        </div>
        <button
          @click="loadFraudsters"
          :disabled="isLoading"
          class="px-4 py-2 text-sm bg-gray-700 text-white rounded hover:bg-gray-600 disabled:opacity-50"
        >
          Обновить
        </button>
      </div>

      <!-- Error -->
      <div v-if="error" class="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded mb-6">
        {{ error }}
      </div>

      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="text-gray-400">Загрузка...</div>
      </div>

      <!-- Empty -->
      <div v-else-if="fraudsters.length === 0" class="text-center py-12">
        <div class="text-gray-400 text-lg">Нет отмеченных накрутчиков</div>
        <p class="text-gray-500 mt-2">Организации с подозрительной активностью пока не обнаружены</p>
      </div>

      <!-- List -->
      <div v-else class="space-y-4">
        <div
          v-for="fraudster in fraudsters"
          :key="fraudster.org_id"
          class="bg-gray-800 rounded-lg p-6"
        >
          <div class="flex justify-between items-start mb-4">
            <div>
              <div class="flex items-center gap-3 mb-2">
                <span class="text-lg font-medium text-white">{{ fraudster.org_name }}</span>
                <span
                  :class="[
                    'px-2 py-0.5 rounded text-xs font-medium',
                    fraudster.is_confirmed
                      ? 'bg-red-900/50 text-red-200 border border-red-500'
                      : 'bg-yellow-900/50 text-yellow-200 border border-yellow-500'
                  ]"
                >
                  {{ fraudster.is_confirmed ? 'Подтверждённый' : 'Подозреваемый' }}
                </span>
              </div>
              <p v-if="fraudster.reason" class="text-gray-300 mb-2">
                <span class="text-gray-500">Причина:</span> {{ fraudster.reason }}
              </p>
              <div class="text-sm text-gray-500 space-y-1">
                <div>
                  <span>Оставлено отзывов: {{ fraudster.total_reviews_left }}</span>
                  <span class="mx-2">|</span>
                  <span>Деактивировано: {{ fraudster.deactivated_reviews }}</span>
                  <span class="mx-2">|</span>
                  <span>Репутация: {{ (fraudster.reputation_score * 100).toFixed(0) }}%</span>
                </div>
                <div>
                  <span>Отмечен: {{ formatDate(fraudster.marked_at) }}</span>
                </div>
              </div>
            </div>
            <div class="flex gap-2">
              <button
                @click="openUnmarkModal(fraudster)"
                class="px-3 py-1.5 text-sm bg-green-600 text-white rounded hover:bg-green-500"
              >
                Снять метку
              </button>
            </div>
          </div>

          <!-- ID -->
          <div class="mt-4 pt-4 border-t border-gray-700 text-xs text-gray-600">
            <span>ID: {{ fraudster.org_id }}</span>
          </div>
        </div>
      </div>
    </main>

    <!-- Unmark Modal -->
    <div v-if="showUnmarkModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
        <h3 class="text-lg font-bold text-white mb-4">Снять метку накрутчика</h3>

        <div class="mb-2">
          <p class="text-gray-300">
            Организация: <span class="font-medium text-white">{{ selectedFraudster?.org_name }}</span>
          </p>
        </div>

        <div class="mb-6">
          <label class="block text-sm text-gray-400 mb-1">Причина снятия метки</label>
          <textarea
            v-model="unmarkReason"
            rows="3"
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white resize-none"
            placeholder="Укажите причину..."
          ></textarea>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="closeModal"
            :disabled="isSubmitting"
            class="px-4 py-2 text-gray-400 hover:text-white"
          >
            Отмена
          </button>
          <button
            @click="submitUnmark"
            :disabled="isSubmitting || !unmarkReason.trim()"
            class="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-500 disabled:opacity-50"
          >
            {{ isSubmitting ? 'Сохранение...' : 'Снять метку' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
