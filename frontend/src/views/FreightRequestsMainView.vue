<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePermissions } from '@/composables/usePermissions'
import { useFreightFiltersStore } from '@/stores/freightFilters'
import { TabsSlider, type TabSliderItem } from '@/components/ui/tabs'
import FreightRequestsListTab from '@/components/freight-request/FreightRequestsListTab.vue'
import FreightRequestWizard from '@/components/freight-request/FreightRequestWizard.vue'
import { List, Plus, Check } from 'lucide-vue-next'

type TabValue = 'list' | 'new'

// Подсказки для каждого шага визарда — индекс совпадает с порядком шагов в FreightRequestWizard.
// Метки шагов берём из самого визарда через defineExpose, чтобы не дублировать.
const WIZARD_HINTS = [
  'Укажите пункт отправления и назначения. При необходимости добавьте промежуточные остановки.',
  'Опишите груз: вес, объём и тип. Чем точнее данные — тем точнее подберётся транспорт.',
  'Выберите тип кузова и особые требования: ADR, термо-режим, тип погрузки.',
  'Укажите ставку и условия оплаты. Конкретная цена привлечёт больше предложений от перевозчиков.',
  'Проверьте данные перед публикацией. После публикации заявку увидят перевозчики.',
]

const route = useRoute()
const router = useRouter()
const { canCreateFreightRequest } = usePermissions()
const filtersStore = useFreightFiltersStore()

function parseTab(raw: unknown): TabValue {
  return raw === 'new' || raw === 'list' ? raw : 'list'
}

const activeTab = ref<TabValue>(parseTab(route.query.tab))

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
  router.replace({ query: { ...route.query, tab } })
})

watch(
  () => route.query.tab,
  (tab) => {
    const parsed = parseTab(tab)
    if (parsed !== activeTab.value) activeTab.value = parsed
  }
)

// Wizard ref — для чтения текущего шага и меток шагов в сайдбар-степпере.
// Метки берём из самого визарда через defineExpose, чтобы не дублировать.
const FALLBACK_STEP_LABELS = ['Маршрут', 'Груз', 'Транспорт', 'Оплата', 'Готово']
const wizardRef = ref<InstanceType<typeof FreightRequestWizard>>()
const wizardCurrentStep = computed(() => (wizardRef.value?.currentStep as number) ?? 1)
const wizardSteps = computed(() => {
  const labels = (wizardRef.value?.steps as string[] | undefined) ?? FALLBACK_STEP_LABELS
  return labels.map((label, i) => ({ label, hint: WIZARD_HINTS[i] ?? '' }))
})
</script>

<template>
  <div class="max-w-7xl mx-auto pt-4 pb-6 px-4 sm:px-6 lg:px-8">

    <!-- Список заявок: табы в левом сайдбаре над фильтрами -->
    <FreightRequestsListTab
      v-if="activeTab === 'list'"
      @go-to-new="activeTab = 'new'"
    >
      <template #sidebar-header>
        <div class="mb-3">
          <TabsSlider
            v-model="activeTab"
            :items="tabs"
            no-overflow
            @update:model-value="(v) => handleTabClick(v as TabValue)"
          />
        </div>
      </template>
    </FreightRequestsListTab>

    <!-- Новая заявка: табы + вертикальный степпер слева, визард справа -->
    <div v-else class="grid grid-cols-1 lg:grid-cols-[300px_1fr] gap-6 items-start">

      <!-- Левая колонка: табы + степпер (только десктоп) -->
      <div class="lg:sticky lg:top-16">
        <div class="mb-3">
          <TabsSlider
            v-model="activeTab"
            :items="tabs"
            no-overflow
            @update:model-value="(v) => handleTabClick(v as TabValue)"
          />
        </div>

        <!-- Вертикальный степпер с подсказками — только на десктопе -->
        <div class="hidden lg:block mt-4">
          <nav class="flex flex-col">
            <template v-for="(step, i) in wizardSteps" :key="step.label">
              <!-- Соединительная линия между шагами -->
              <div
                v-if="i > 0"
                class="ml-[13px] w-0.5 h-4 transition-colors"
                :class="i < wizardCurrentStep ? 'bg-primary' : 'bg-border'"
              />
              <!-- Шаг -->
              <button
                type="button"
                class="flex items-center gap-3 rounded-md py-0.5 transition-colors"
                :disabled="i + 1 > wizardCurrentStep"
                :class="i + 1 < wizardCurrentStep ? 'cursor-pointer' : 'cursor-default'"
                @click="i + 1 < wizardCurrentStep && wizardRef?.goToStep(i + 1)"
              >
                <!-- Круг -->
                <div
                  class="flex items-center justify-center w-7 h-7 rounded-full text-sm font-semibold shrink-0 transition-all"
                  :class="[
                    i + 1 === wizardCurrentStep
                      ? 'bg-primary text-primary-foreground ring-4 ring-primary/20'
                      : i + 1 < wizardCurrentStep
                        ? 'bg-primary text-primary-foreground'
                        : 'border-2 border-border text-muted-foreground bg-background',
                  ]"
                >
                  <Check v-if="i + 1 < wizardCurrentStep" class="h-3.5 w-3.5" />
                  <span v-else>{{ i + 1 }}</span>
                </div>
                <!-- Лейбл -->
                <span
                  class="text-sm transition-colors"
                  :class="[
                    i + 1 === wizardCurrentStep
                      ? 'font-semibold text-foreground'
                      : i + 1 < wizardCurrentStep
                        ? 'font-medium text-primary/70'
                        : 'text-muted-foreground',
                  ]"
                >
                  {{ step.label }}
                </span>
              </button>
            </template>
          </nav>

          <!-- Подсказка для текущего шага -->
          <div class="mt-4 rounded-lg bg-muted/50 p-3 text-sm text-muted-foreground leading-relaxed">
            {{ wizardSteps[wizardCurrentStep - 1]?.hint }}
          </div>
        </div>
      </div>

      <!-- Правая колонка: визард без горизонтального степпера на десктопе -->
      <div class="bg-white -mx-4 sm:-mx-6 lg:mx-0 lg:bg-background">
        <div class="px-4 sm:px-6 lg:px-0 pt-4 lg:pt-0">
          <FreightRequestWizard
            ref="wizardRef"
            hide-stepper
            title="Новая заявка на перевозку"
          />
        </div>
      </div>

    </div>
  </div>
</template>
