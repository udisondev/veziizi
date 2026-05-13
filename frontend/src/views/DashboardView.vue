<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useFreightFiltersStore } from '@/stores/freightFilters'
import FreightRequestsDashboardTab from '@/components/freight-request/FreightRequestsDashboardTab.vue'

const router = useRouter()
const filtersStore = useFreightFiltersStore()

function handleGoToList(filtered = false) {
  if (!filtered) filtersStore.resetFilters()
  router.push('/requests')
}

function handleGoToNew() {
  router.push({ path: '/requests', query: { tab: 'new' } })
}
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <FreightRequestsDashboardTab
      @go-to-new="handleGoToNew"
      @go-to-list="handleGoToList"
    />
  </div>
</template>
