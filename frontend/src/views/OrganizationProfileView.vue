<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { organizationsApi } from '@/api/organizations'
import type { OrganizationDetail, OrganizationRating, OrganizationReview, OrganizationStats } from '@/types/admin'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { logger } from '@/utils/logger'

// UI Components
import { Card, CardContent } from '@/components/ui/card'
import { AppLink } from '@/components/ui/app-link'

// Shared Components
import { DetailPageHeader, LoadingSpinner, ErrorBanner } from '@/components/shared'

const route = useRoute()

const organization = ref<OrganizationDetail | null>(null)
const rating = ref<OrganizationRating | null>(null)
const stats = ref<OrganizationStats | null>(null)
const reviews = ref<OrganizationReview[]>([])
const reviewsCursor = ref<string | undefined>(undefined)
const reviewsHasMore = ref(false)
const isLoading = ref(true)
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
  pending: 'bg-warning/10 text-warning',
  active: 'bg-success/10 text-success',
  suspended: 'bg-destructive/10 text-destructive',
  rejected: 'bg-muted text-muted-foreground',
}

// Infinite scroll setup
const canLoadMore = computed(() => reviewsHasMore.value && reviewsCursor.value !== undefined)
const { sentinelRef, isLoadingMore } = useInfiniteScroll(loadMoreReviews, {
  threshold: 300,
  enabled: canLoadMore,
})

// Template refs used in template via ref="..." (vue-tsc false positive workaround)
void sentinelRef

async function loadData() {
  isLoading.value = true
  error.value = ''
  // Reset reviews pagination
  reviews.value = []
  reviewsCursor.value = undefined
  reviewsHasMore.value = false

  try {
    const id = route.params.id as string
    const [orgData, ratingData, reviewsData] = await Promise.all([
      organizationsApi.get(id),
      organizationsApi.getRating(id),
      organizationsApi.getReviews(id, { limit: REVIEWS_PER_PAGE }),
    ])
    organization.value = orgData
    rating.value = ratingData

    organizationsApi.getStats(id).then(s => { stats.value = s }).catch(() => {})
    reviews.value = reviewsData.items ?? []
    reviewsCursor.value = reviewsData.next_cursor
    reviewsHasMore.value = reviewsData.has_more
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

async function loadMoreReviews() {
  if (!reviewsHasMore.value || !reviewsCursor.value) return

  try {
    const id = route.params.id as string
    const reviewsData = await organizationsApi.getReviews(id, {
      limit: REVIEWS_PER_PAGE,
      cursor: reviewsCursor.value,
    })
    reviews.value.push(...(reviewsData.items ?? []))
    reviewsCursor.value = reviewsData.next_cursor
    reviewsHasMore.value = reviewsData.has_more
  } catch (e) {
    logger.error('Failed to load more reviews', e)
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
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <DetailPageHeader back-to="/" back-label="Назад" use-history back-tutorial-id="org-back-button" />

    <!-- Content -->
    <main class="max-w-4xl mx-auto px-4 py-6">
      <!-- Loading -->
      <LoadingSpinner v-if="isLoading" text="Загрузка..." />

      <!-- Error -->
      <ErrorBanner v-else-if="error" :message="error" @retry="loadData" />

      <!-- Content -->
      <div v-else-if="organization" class="space-y-6">
        <!-- Header Card with Rating -->
        <Card>
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0 flex-1">
                <h1 class="text-2xl font-bold text-foreground break-words">{{ organization.name }}</h1>
                <p class="text-muted-foreground mt-1 break-words">{{ organization.legal_name }}</p>

                <!-- Rating -->
                <div v-if="rating" class="flex items-center gap-2 mt-3">
                  <div class="flex">
                    <span
                      v-for="star in 5"
                      :key="star"
                      :class="[
                        'text-xl',
                        star <= Math.round(rating.average_rating) ? 'text-warning' : 'text-muted-foreground/30'
                      ]"
                    >&#9733;</span>
                  </div>
                  <span class="text-foreground font-medium">
                    {{ rating.average_rating.toFixed(1) }}
                  </span>
                  <span class="text-muted-foreground text-sm">
                    ({{ rating.total_reviews }} {{ rating.total_reviews === 1 ? 'отзыв' : rating.total_reviews < 5 ? 'отзыва' : 'отзывов' }})
                  </span>
                </div>

                <!-- Stats -->
                <div v-if="stats" class="flex flex-wrap gap-3 mt-4 text-sm">
                  <div class="flex flex-col items-center px-3 py-2 bg-muted rounded-lg min-w-[80px]">
                    <span class="text-lg font-bold text-foreground">{{ stats.total_freight_requests }}</span>
                    <span class="text-muted-foreground text-xs">Заявок</span>
                  </div>
                  <div class="flex flex-col items-center px-3 py-2 bg-muted rounded-lg min-w-[80px]">
                    <span class="text-lg font-bold text-primary">{{ stats.active_freight_requests }}</span>
                    <span class="text-muted-foreground text-xs">Активных</span>
                  </div>
                  <div class="flex flex-col items-center px-3 py-2 bg-muted rounded-lg min-w-[80px]">
                    <span class="text-lg font-bold text-success">{{ stats.completed_deals }}</span>
                    <span class="text-muted-foreground text-xs">Завершено</span>
                  </div>
                  <div class="flex flex-col items-center px-3 py-2 bg-muted rounded-lg min-w-[80px]">
                    <span class="text-lg font-bold text-foreground">{{ stats.total_offers_made }}</span>
                    <span class="text-muted-foreground text-xs">Предложений</span>
                  </div>
                  <div class="flex flex-col items-center px-3 py-2 bg-muted rounded-lg min-w-[80px]">
                    <span class="text-lg font-bold text-success">{{ stats.successful_offers }}</span>
                    <span class="text-muted-foreground text-xs">Успешных</span>
                  </div>
                </div>
              </div>
              <span :class="[statusColors[organization.status], 'px-3 py-1 rounded-full text-sm font-medium shrink-0']">
                {{ statusLabels[organization.status] }}
              </span>
            </div>
          </CardContent>
        </Card>

        <!-- Details Card -->
        <Card>
          <CardContent class="p-6">
            <h2 class="text-lg font-semibold text-foreground mb-4">Информация</h2>
            <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <dt class="text-sm text-muted-foreground">ИНН</dt>
                <dd class="text-foreground font-medium">{{ organization.inn }}</dd>
              </div>
              <div>
                <dt class="text-sm text-muted-foreground">Страна</dt>
                <dd class="text-foreground">{{ countryLabels[organization.country] || organization.country }}</dd>
              </div>
              <div>
                <dt class="text-sm text-muted-foreground">Телефон</dt>
                <dd class="text-foreground">{{ organization.phone }}</dd>
              </div>
              <div>
                <dt class="text-sm text-muted-foreground">Email</dt>
                <dd class="text-foreground">{{ organization.email }}</dd>
              </div>
              <div class="sm:col-span-2">
                <dt class="text-sm text-muted-foreground">Адрес</dt>
                <dd class="text-foreground break-words">{{ organization.address }}</dd>
              </div>
              <div>
                <dt class="text-sm text-muted-foreground">Дата регистрации</dt>
                <dd class="text-foreground">{{ formatDate(organization.created_at) }}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        <!-- Reviews -->
        <div>
          <h2 class="text-lg font-semibold text-foreground mb-4">
            Отзывы
            <span v-if="rating && rating.total_reviews > 0" class="text-muted-foreground font-normal">({{ rating.total_reviews }})</span>
          </h2>

          <div v-if="reviews.length === 0" class="text-center py-8 text-muted-foreground">
            Пока нет отзывов
          </div>

          <div v-else class="space-y-4">
            <div
              v-for="review in reviews"
              :key="review.id"
              class="bg-card border border-border rounded-lg p-4 shadow-sm"
            >
              <div class="flex items-start justify-between mb-2">
                <div>
                  <AppLink :to="{ name: 'organization-profile', params: { id: review.reviewer_org_id } }">
                    {{ review.reviewer_org_name || 'Организация' }}
                  </AppLink>
                  <div class="flex mt-1">
                    <span
                      v-for="star in 5"
                      :key="star"
                      :class="[
                        'text-lg',
                        star <= review.rating ? 'text-warning' : 'text-muted-foreground/30'
                      ]"
                    >&#9733;</span>
                  </div>
                </div>
                <div class="text-xs text-muted-foreground">
                  {{ formatDateTime(review.created_at) }}
                </div>
              </div>
              <p v-if="review.comment" class="text-muted-foreground mt-2 break-words">{{ review.comment }}</p>
            </div>

            <!-- Infinite scroll sentinel -->
            <div ref="sentinelRef" class="h-12 flex items-center justify-center">
              <LoadingSpinner v-if="isLoadingMore" text="Загрузка..." />
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>
