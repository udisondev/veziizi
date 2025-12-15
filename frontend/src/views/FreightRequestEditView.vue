<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { freightRequestsApi } from '@/api/freightRequests'
import { usePermissions } from '@/composables/usePermissions'
import FreightRequestWizard from '@/components/freight-request/FreightRequestWizard.vue'
import type { FreightRequest } from '@/types/freightRequest'

const route = useRoute()
const router = useRouter()
const permissions = usePermissions()

const freightRequest = ref<FreightRequest | null>(null)
const isLoading = ref(true)
const error = ref('')

onMounted(async () => {
  await loadFreightRequest()
})

async function loadFreightRequest() {
  isLoading.value = true
  error.value = ''

  try {
    const id = route.params.id as string
    const fr = await freightRequestsApi.get(id)

    // Проверяем права на редактирование
    if (!permissions.canEditFreightRequest(fr.customer_org_id, fr.customer_member_id)) {
      router.push(`/freight-requests/${id}`)
      return
    }

    // Проверяем статус — редактировать можно только published
    if (fr.status !== 'published') {
      error.value = 'Заявку можно редактировать только в статусе "Опубликована"'
      return
    }

    freightRequest.value = fr
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки заявки'
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Header -->
    <header class="bg-white shadow">
      <div class="max-w-3xl mx-auto px-4 py-4">
        <router-link
          :to="freightRequest ? `/freight-requests/${freightRequest.id}` : '/'"
          class="text-blue-600 hover:text-blue-800 text-sm"
        >
          &larr; Назад к заявке
        </router-link>
        <h1 class="text-xl font-bold text-gray-900 mt-2">Редактирование заявки</h1>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-3xl mx-auto px-4 py-6">
      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="text-gray-500">Загрузка...</div>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
        {{ error }}
        <router-link
          :to="route.params.id ? `/freight-requests/${route.params.id}` : '/'"
          class="ml-4 text-red-600 underline"
        >
          Вернуться
        </router-link>
      </div>

      <!-- Wizard -->
      <FreightRequestWizard
        v-else-if="freightRequest"
        :edit-mode="true"
        :freight-request-id="freightRequest.id"
        :initial-data="freightRequest"
      />
    </main>
  </div>
</template>
