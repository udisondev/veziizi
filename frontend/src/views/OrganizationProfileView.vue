<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { organizationsApi } from '@/api/organizations'
import type { OrganizationDetail, OrganizationRating, OrganizationReview } from '@/types/admin'

const route = useRoute()
const router = useRouter()

const organization = ref<OrganizationDetail | null>(null)
const rating = ref<OrganizationRating | null>(null)
const reviews = ref<OrganizationReview[]>([])
const reviewsTotal = ref(0)
const isLoading = ref(true)
const isLoadingMore = ref(false)
const error = ref('')

const REVIEWS_PER_PAGE = 5

const countryLabels: Record<string, string> = {
  RU: 'Россия',
  KZ: 'Казахстан',
  BY: 'Беларусь',
}

const statusLabels: Record<string, string> = {
  pending: 'На модерации',
  active: 'Активна',
  suspended: 'Приостановлена',
  rejected: 'Отклонена',
}

const statusColors: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  active: 'bg-green-100 text-green-800',
  suspended: 'bg-red-100 text-red-800',
  rejected: 'bg-gray-100 text-gray-800',
}

const hasMoreReviews = computed(() => reviews.value.length < reviewsTotal.value)

async function loadData() {
  isLoading.value = true
  error.value = ''
  try {
    const id = route.params.id as string
    const [orgData, ratingData, reviewsData] = await Promise.all([
      organizationsApi.get(id),
      organizationsApi.getRating(id),
      organizationsApi.getReviews(id, { limit: REVIEWS_PER_PAGE }),
    ])
    organization.value = orgData
    rating.value = ratingData
    reviews.value = reviewsData.items ?? []
    reviewsTotal.value = reviewsData.total ?? 0
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

async function loadMoreReviews() {
  if (isLoadingMore.value || !hasMoreReviews.value) return
  isLoadingMore.value = true
  try {
    const id = route.params.id as string
    const reviewsData = await organizationsApi.getReviews(id, {
      limit: REVIEWS_PER_PAGE,
      offset: reviews.value.length,
    })
    reviews.value.push(...(reviewsData.items ?? []))
  } catch (e) {
    console.error('Failed to load more reviews:', e)
  } finally {
    isLoadingMore.value = false
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

onMounted(() => {
  loadData()
})

watch(() => route.params.id, () => {
  loadData()
})
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Header -->
    <header class="bg-white shadow">
      <div class="max-w-4xl mx-auto px-4 py-4">
        <button
          @click="router.back()"
          class="text-blue-600 hover:text-blue-800 text-sm"
        >
          &larr; Назад
        </button>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-4xl mx-auto px-4 py-6">
      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="text-gray-500">Загрузка...</div>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
        {{ error }}
        <button @click="loadData" class="ml-4 text-red-600 underline">Повторить</button>
      </div>

      <!-- Content -->
      <div v-else-if="organization" class="space-y-6">
        <!-- Header Card with Rating -->
        <div class="bg-white rounded-lg shadow p-6">
          <div class="flex items-start justify-between">
            <div>
              <h1 class="text-2xl font-bold text-gray-900 break-words">{{ organization.name }}</h1>
              <p class="text-gray-500 mt-1 break-words">{{ organization.legal_name }}</p>

              <!-- Rating -->
              <div v-if="rating" class="flex items-center gap-2 mt-3">
                <div class="flex">
                  <span
                    v-for="star in 5"
                    :key="star"
                    :class="[
                      'text-xl',
                      star <= Math.round(rating.average_rating) ? 'text-yellow-400' : 'text-gray-300'
                    ]"
                  >&#9733;</span>
                </div>
                <span class="text-gray-700 font-medium">
                  {{ rating.average_rating.toFixed(1) }}
                </span>
                <span class="text-gray-500 text-sm">
                  ({{ rating.total_reviews }} {{ rating.total_reviews === 1 ? 'отзыв' : rating.total_reviews < 5 ? 'отзыва' : 'отзывов' }})
                </span>
              </div>
            </div>
            <span :class="[statusColors[organization.status], 'px-3 py-1 rounded-full text-sm font-medium']">
              {{ statusLabels[organization.status] }}
            </span>
          </div>
        </div>

        <!-- Details Card -->
        <div class="bg-white rounded-lg shadow p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Информация</h2>
          <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <dt class="text-sm text-gray-500">ИНН</dt>
              <dd class="text-gray-900 font-medium">{{ organization.inn }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-500">Страна</dt>
              <dd class="text-gray-900">{{ countryLabels[organization.country] || organization.country }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-500">Телефон</dt>
              <dd class="text-gray-900">{{ organization.phone }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-500">Email</dt>
              <dd class="text-gray-900">{{ organization.email }}</dd>
            </div>
            <div class="sm:col-span-2">
              <dt class="text-sm text-gray-500">Адрес</dt>
              <dd class="text-gray-900 break-words">{{ organization.address }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-500">Дата регистрации</dt>
              <dd class="text-gray-900">{{ formatDate(organization.created_at) }}</dd>
            </div>
          </dl>
        </div>

        <!-- Reviews Card -->
        <div class="bg-white rounded-lg shadow p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">
            Отзывы
            <span v-if="reviewsTotal > 0" class="text-gray-500 font-normal">({{ reviewsTotal }})</span>
          </h2>

          <div v-if="reviews.length === 0" class="text-center py-8 text-gray-500">
            Пока нет отзывов
          </div>

          <div v-else class="space-y-4">
            <div
              v-for="review in reviews"
              :key="review.id"
              class="border border-gray-200 rounded-lg p-4"
            >
              <div class="flex items-start justify-between mb-2">
                <div>
                  <router-link
                    :to="{ name: 'organization-profile', params: { id: review.reviewer_org_id } }"
                    class="font-medium text-blue-600 hover:text-blue-800 hover:underline"
                  >
                    {{ review.reviewer_org_name || 'Организация' }}
                  </router-link>
                  <div class="flex mt-1">
                    <span
                      v-for="star in 5"
                      :key="star"
                      :class="[
                        'text-lg',
                        star <= review.rating ? 'text-yellow-400' : 'text-gray-300'
                      ]"
                    >&#9733;</span>
                  </div>
                </div>
                <div class="text-xs text-gray-400">
                  {{ formatDateTime(review.created_at) }}
                </div>
              </div>
              <p v-if="review.comment" class="text-gray-700 mt-2 break-words">{{ review.comment }}</p>
            </div>

            <!-- Load More Button -->
            <div v-if="hasMoreReviews" class="text-center pt-4">
              <button
                @click="loadMoreReviews"
                :disabled="isLoadingMore"
                class="px-4 py-2 text-blue-600 hover:bg-blue-50 rounded-lg text-sm font-medium disabled:opacity-50"
              >
                {{ isLoadingMore ? 'Загрузка...' : 'Показать ещё' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>
