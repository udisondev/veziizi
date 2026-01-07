<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useFreightRequestForm } from '@/composables/useFreightRequestForm'
import { useTutorialEvent } from '@/composables/useTutorialEvent'
import { useOnboardingStore } from '@/stores/onboarding'
import { freightRequestsApi } from '@/api/freightRequests'
import { scrollToFirstError } from '@/utils/scrollToError'
import WizardStepIndicator from './WizardStepIndicator.vue'
import RouteStep from './steps/RouteStep.vue'
import CargoStep from './steps/CargoStep.vue'
import VehicleStep from './steps/VehicleStep.vue'
import PaymentStep from './steps/PaymentStep.vue'
import ConfirmationStep from './steps/ConfirmationStep.vue'
import type { FreightRequest } from '@/types/freightRequest'

interface Props {
  editMode?: boolean
  freightRequestId?: string
  initialData?: FreightRequest
}

const props = withDefaults(defineProps<Props>(), {
  editMode: false,
})

const router = useRouter()
const form = useFreightRequestForm()
const { emit: emitTutorial } = useTutorialEvent()
const onboarding = useOnboardingStore()

const isLoading = ref(false)
const apiError = ref('')

const steps = ['Маршрут', 'Груз', 'Транспорт', 'Оплата', 'Подтверждение']

onMounted(() => {
  if (props.editMode && props.initialData) {
    form.loadFromRequest(props.initialData)
  }
})

// Прокрутка наверх при смене шага
watch(
  () => form.currentStep.value,
  () => {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
)

async function handleSubmit() {
  // В sandbox режиме не делаем реальный API вызов
  if (onboarding.isSandboxMode) {
    const routePoints = form.routePoints.value
    const firstPoint = routePoints[0]
    const lastPoint = routePoints[routePoints.length - 1]

    onboarding.setSandboxCreatedRequest({
      id: 'sandbox-request-' + Date.now(),
      origin_address: firstPoint?.address || 'Не указан',
      destination_address: lastPoint?.address || 'Не указан',
      cargo_weight: form.cargo.weight || 0,
      price_amount: form.payment.price?.amount,
      price_currency: form.payment.price?.currency,
      vehicle_type: form.vehicle.vehicle_type || 'truck',
      vehicle_subtype: form.vehicle.vehicle_subtype || 'tilt',
      created_at: new Date().toISOString(),
    })
    emitTutorial('freightRequest:created', { id: 'sandbox-request' })
    router.push('/')
    return
  }

  isLoading.value = true
  apiError.value = ''

  try {
    if (props.editMode && props.freightRequestId) {
      await freightRequestsApi.update(props.freightRequestId, form.requestData.value)
      router.push(`/freight-requests/${props.freightRequestId}`)
    } else {
      const result = await freightRequestsApi.create(form.requestData.value)
      router.push(`/freight-requests/${result.id}`)
    }
  } catch (e) {
    apiError.value = e instanceof Error ? e.message : (props.editMode ? 'Ошибка сохранения' : 'Ошибка создания заявки')
    isLoading.value = false
  }
}

function handleNext() {
  if (form.currentStep.value === form.totalSteps) {
    handleSubmit()
  } else {
    // Отправляем событие только если валидация прошла и переход выполнен
    if (form.nextStep()) {
      // Выходим из режима ошибки валидации если были в нём
      if (onboarding.validationErrorMode) {
        onboarding.exitValidationErrorMode()
      }
      emitTutorial('wizard:next')
    } else {
      // Валидация не прошла - скроллим к первой ошибке
      console.log('[handleNext] validation failed, errors:', JSON.stringify(form.errors))
      scrollToFirstError(form.errors)
    }
  }
}
</script>

<template>
  <div class="max-w-3xl mx-auto">
    <!-- Step indicator -->
    <WizardStepIndicator
      :steps="steps"
      :current-step="form.currentStep.value"
      @go-to="form.goToStep"
    />

    <!-- API Error -->
    <div
      v-if="apiError"
      class="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded-md mb-6"
    >
      {{ apiError }}
    </div>

    <!-- Steps content -->
    <div class="bg-white shadow-sm rounded-lg p-6">
      <!-- Step 1: Route -->
      <RouteStep
        v-if="form.currentStep.value === 1"
        data-tutorial="route-step"
        :route-points="form.routePoints.value"
        :errors="form.errors"
        @add-point="form.addRoutePoint"
        @remove-point="form.removeRoutePoint"
        @update-point="form.updateRoutePoint"
        @reorder="form.reorderRoutePoints"
      />

      <!-- Step 2: Cargo -->
      <CargoStep
        v-else-if="form.currentStep.value === 2"
        data-tutorial="cargo-step"
        :cargo="form.cargo"
        :errors="form.errors"
        @update:cargo="Object.assign(form.cargo, $event)"
        @validate-field="form.validateField"
      />

      <!-- Step 3: Vehicle -->
      <VehicleStep
        v-else-if="form.currentStep.value === 3"
        data-tutorial="vehicle-step"
        :vehicle="form.vehicle"
        :errors="form.errors"
        @update:vehicle="Object.assign(form.vehicle, $event)"
        @validate-field="form.validateField"
      />

      <!-- Step 4: Payment -->
      <PaymentStep
        v-else-if="form.currentStep.value === 4"
        data-tutorial="payment-step"
        :payment="form.payment"
        :errors="form.errors"
        @update:payment="Object.assign(form.payment, $event)"
        @validate-field="form.validateField"
      />

      <!-- Step 5: Confirmation -->
      <ConfirmationStep
        v-else-if="form.currentStep.value === 5"
        :request-data="form.requestData.value"
        :comment="form.comment.value"
        @update:comment="form.comment.value = $event"
      />
    </div>

    <!-- Navigation buttons -->
    <div class="flex gap-4 mt-6" data-tutorial="wizard-buttons">
      <button
        v-if="form.currentStep.value > 1"
        type="button"
        data-tutorial="back-btn"
        class="flex-1 py-3 px-4 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-colors"
        @click="form.prevStep"
      >
        Назад
      </button>

      <button
        type="button"
        data-tutorial="submit-btn"
        :disabled="isLoading"
        :class="[
          'flex-1 py-3 px-4 border border-transparent rounded-md text-sm font-medium text-white focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-colors',
          form.currentStep.value === form.totalSteps
            ? 'bg-green-600 hover:bg-green-700'
            : 'bg-blue-600 hover:bg-blue-700',
          isLoading ? 'opacity-50 cursor-not-allowed' : '',
        ]"
        @click="handleNext"
      >
        <template v-if="isLoading">
          <svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-white inline" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          {{ editMode ? 'Сохранение...' : 'Публикация...' }}
        </template>
        <template v-else>
          {{ form.currentStep.value === form.totalSteps ? (editMode ? 'Сохранить изменения' : 'Опубликовать') : 'Далее' }}
        </template>
      </button>
    </div>

    <!-- Cancel link -->
    <div class="text-center mt-4">
      <router-link
        :to="editMode && freightRequestId ? `/freight-requests/${freightRequestId}` : '/'"
        class="text-gray-500 hover:text-gray-700 text-sm"
      >
        Отмена
      </router-link>
    </div>
  </div>
</template>
