<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePermissions } from '@/composables/usePermissions'
import { useFreightFiltersStore } from '@/stores/freightFilters'
import { TabsSlider, type TabSliderItem } from '@/components/ui/tabs'
import FreightRequestsListTab from '@/components/freight-request/FreightRequestsListTab.vue'
import FreightRequestWizard from '@/components/freight-request/FreightRequestWizard.vue'
import { List, Plus } from 'lucide-vue-next'

type TabValue = 'list' | 'new'

const route = useRoute()
const router = useRouter()
const { canCreateFreightRequest } = usePermissions()
const filtersStore = useFreightFiltersStore()

function parseTab(raw: unknown): TabValue {
  return raw === 'new' || raw === 'list' ? raw : 'list'
}

const activeTab = ref<TabValue>(parseTab(route.query.tab))
const wizardMounted = ref(activeTab.value === 'new')

const tabs = computed<TabSliderItem[]>(() => {
  const t: TabSliderItem[] = [
    { value: 'list', label: 'Список заявок', icon: List },
  ]
  if (canCreateFreightRequest.value) {
    t.push({ value: 'new', label: 'Новая заявка', icon: Plus })
  }
  return t
})

function handleTabClick(tab: TabValue) {
  if (tab === 'list') filtersStore.resetFilters()
}

watch(activeTab, (tab) => {
  if (tab === 'new') wizardMounted.value = true
  router.replace({ query: { ...route.query, tab } })
})

watch(
  () => route.query.tab,
  (tab) => {
    const parsed = parseTab(tab)
    if (parsed !== activeTab.value) activeTab.value = parsed
  }
)
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Tab navigation -->
    <div class="mb-6">
      <TabsSlider
        v-model="activeTab"
        :items="tabs"
        @update:model-value="(v) => handleTabClick(v as TabValue)"
      />
    </div>

    <!-- Tab content -->
    <FreightRequestsListTab
      v-if="activeTab === 'list'"
      @go-to-new="activeTab = 'new'"
    />

    <div
      v-if="wizardMounted"
      v-show="activeTab === 'new'"
      class="bg-white -mx-4 sm:-mx-6 md:mx-0 md:bg-background"
    >
      <div class="max-w-5xl mx-auto px-4 sm:px-6 md:px-0 pt-4 md:pt-0">
        <FreightRequestWizard title="Новая заявка на перевозку" />
      </div>
    </div>
  </div>
</template>
