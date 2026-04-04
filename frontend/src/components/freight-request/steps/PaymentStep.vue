<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Payment, Currency, VatType, PaymentMethod, PaymentTerms } from '@/types/freightRequest'
import {
  currencyOptions,
  vatTypeOptions,
  paymentMethodOptions,
  paymentTermsOptions,
  currencyLabels,
} from '@/types/freightRequest'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useBreakpoint } from '@/composables/useBreakpoint'
import BottomSheet from '@/components/shared/BottomSheet.vue'

interface Props {
  payment: Payment
  errors: Record<string, string | null>
}

interface Emits {
  (e: 'update:payment', value: Payment): void
  (e: 'validateField', field: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { isMobile } = useBreakpoint()

const currencySheetOpen = ref(false)
const vatSheetOpen = ref(false)
const methodSheetOpen = ref(false)
const termsSheetOpen = ref(false)

const selectedCurrencyLabel = computed(() =>
  currencyOptions.find(o => o.value === (props.payment.price?.currency || 'RUB'))?.label
)
const selectedVatLabel = computed(() =>
  vatTypeOptions.find(o => o.value === props.payment.vat_type)?.label ?? null
)
const selectedMethodLabel = computed(() =>
  paymentMethodOptions.find(o => o.value === props.payment.method)?.label ?? null
)
const selectedTermsLabel = computed(() =>
  paymentTermsOptions.find(o => o.value === props.payment.terms)?.label ?? null
)

function updateField<K extends keyof Payment>(field: K, value: Payment[K]) {
  emit('update:payment', { ...props.payment, [field]: value })
}

// Конвертация рублей в копейки для отображения/ввода
const displayAmount = computed(() => {
  if (!props.payment.price?.amount) return ''
  return (props.payment.price.amount / 100).toString()
})

function handleAmountInput(event: Event) {
  const inputValue = (event.target as HTMLInputElement).value

  if (!inputValue) {
    // Если поле очищено, убираем price
    updateField('price', undefined)
    return
  }

  const value = parseFloat(inputValue) || 0
  // Конвертируем в копейки
  const amountInCents = Math.round(value * 100)
  const currentCurrency = props.payment.price?.currency || 'RUB'
  updateField('price', { amount: amountInCents, currency: currentCurrency })
}

function handleCurrencyChange(currency: Currency) {
  const currentAmount = props.payment.price?.amount || 0
  updateField('price', { amount: currentAmount, currency })
}

function handleVatTypeChange(vatType: VatType) {
  updateField('vat_type', vatType)
}

function handleMethodChange(method: PaymentMethod) {
  updateField('method', method)
}

function handleCurrencySheetSelect(currency: Currency) {
  handleCurrencyChange(currency)
  currencySheetOpen.value = false
}

function handleVatSheetSelect(vatType: VatType) {
  handleVatTypeChange(vatType)
  vatSheetOpen.value = false
}

function handleMethodSheetSelect(method: PaymentMethod) {
  handleMethodChange(method)
  methodSheetOpen.value = false
}

function handleTermsSheetSelect(terms: PaymentTerms) {
  handleTermsChange(terms)
  termsSheetOpen.value = false
}

function handleTermsChange(terms: PaymentTerms) {
  updateField('terms', terms)
  // Очищаем дни отсрочки если не deferred
  if (terms !== 'deferred') {
    updateField('deferred_days', undefined)
  }
}

function handleDeferredDaysInput(event: Event) {
  const value = parseInt((event.target as HTMLInputElement).value) || undefined
  updateField('deferred_days', value)
}

function handleNoPriceChange(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  emit('update:payment', {
    ...props.payment,
    no_price: checked,
    price: checked ? undefined : props.payment.price,
  })
}

const showDeferredDays = computed(() => props.payment.terms === 'deferred')

const hasPrice = computed(() => !!props.payment.price?.amount)

const inputClass = (field: string) => [
  'appearance-none block w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500',
  props.errors[field] ? 'border-red-300' : 'border-gray-300',
]
</script>

<template>
  <div class="space-y-6">
    <!-- Галочка "Не указывать цену" -->
    <div data-tutorial="payment-no-price" class="flex items-center gap-3">
      <input
        id="no-price"
        type="checkbox"
        :checked="payment.no_price"
        class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
        @change="handleNoPriceChange"
      />
      <label for="no-price" class="text-sm text-gray-700">
        Не указывать цену
        <span class="text-gray-500">(перевозчики предложат свою)</span>
      </label>
    </div>

    <!-- Price (показываем только если галочка не отмечена) -->
    <div v-if="!payment.no_price" data-tutorial="payment-price">
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Стоимость перевозки <span class="text-red-500">*</span>
      </label>
      <div class="relative">
        <input
          type="number"
          :value="displayAmount"
          placeholder="Укажите сумму"
          min="0"
          step="100"
          :class="inputClass('price')"
          @input="handleAmountInput"
          @blur="emit('validateField', 'price')"
        />
        <span v-if="hasPrice" class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400">
          {{ currencyLabels[payment.price?.currency || 'RUB'] }}
        </span>
      </div>
      <p v-if="errors.price" class="mt-1 text-sm text-red-600">
        {{ errors.price }}
      </p>
    </div>

    <!-- Currency -->
    <div v-if="!payment.no_price" data-tutorial="payment-currency">
      <label class="block text-sm font-medium text-gray-700 mb-1">Валюта</label>
      <template v-if="!isMobile()">
        <Select :model-value="payment.price?.currency || 'RUB'" @update:model-value="handleCurrencyChange($event as Currency)">
          <SelectTrigger><SelectValue placeholder="Выберите валюту" /></SelectTrigger>
          <SelectContent>
            <SelectItem v-for="o in currencyOptions" :key="o.value" :value="o.value">{{ o.label }}</SelectItem>
          </SelectContent>
        </Select>
      </template>
      <template v-else>
        <button type="button" class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-left bg-white" @click="currencySheetOpen = true">
          <span class="text-gray-900">{{ selectedCurrencyLabel }}</span>
        </button>
        <BottomSheet v-model="currencySheetOpen" label="Валюта">
          <div class="overflow-y-auto flex-1">
            <button v-for="o in currencyOptions" :key="o.value" type="button"
              :class="['w-full px-4 py-3 text-left text-sm border-b border-gray-50 active:bg-gray-100', o.value === (payment.price?.currency || 'RUB') ? 'bg-blue-50 text-blue-700 font-medium' : 'text-gray-900']"
              @click="handleCurrencySheetSelect(o.value)">{{ o.label }}</button>
          </div>
        </BottomSheet>
      </template>
    </div>

    <!-- Все остальные поля оплаты показываем только если цена указывается -->
    <template v-if="!payment.no_price">
      <!-- VAT type -->
      <div data-tutorial="payment-vat">
        <label class="block text-sm font-medium text-gray-700 mb-1">НДС</label>
        <template v-if="!isMobile()">
          <Select :model-value="payment.vat_type" @update:model-value="handleVatTypeChange($event as VatType)">
            <SelectTrigger><SelectValue placeholder="Выберите тип НДС" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="o in vatTypeOptions" :key="o.value" :value="o.value">{{ o.label }}</SelectItem>
            </SelectContent>
          </Select>
        </template>
        <template v-else>
          <button type="button" class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-left bg-white" @click="vatSheetOpen = true">
            <span :class="selectedVatLabel ? 'text-gray-900' : 'text-gray-400'">{{ selectedVatLabel || 'Выберите тип НДС' }}</span>
          </button>
          <BottomSheet v-model="vatSheetOpen" label="НДС">
            <div class="overflow-y-auto flex-1">
              <button v-for="o in vatTypeOptions" :key="o.value" type="button"
                :class="['w-full px-4 py-3 text-left text-sm border-b border-gray-50 active:bg-gray-100', o.value === payment.vat_type ? 'bg-blue-50 text-blue-700 font-medium' : 'text-gray-900']"
                @click="handleVatSheetSelect(o.value)">{{ o.label }}</button>
            </div>
          </BottomSheet>
        </template>
      </div>

      <!-- Payment method -->
      <div data-tutorial="payment-method">
        <label class="block text-sm font-medium text-gray-700 mb-1">Способ оплаты</label>
        <template v-if="!isMobile()">
          <Select :model-value="payment.method" @update:model-value="handleMethodChange($event as PaymentMethod)">
            <SelectTrigger><SelectValue placeholder="Выберите способ оплаты" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="o in paymentMethodOptions" :key="o.value" :value="o.value">{{ o.label }}</SelectItem>
            </SelectContent>
          </Select>
        </template>
        <template v-else>
          <button type="button" class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-left bg-white" @click="methodSheetOpen = true">
            <span :class="selectedMethodLabel ? 'text-gray-900' : 'text-gray-400'">{{ selectedMethodLabel || 'Выберите способ оплаты' }}</span>
          </button>
          <BottomSheet v-model="methodSheetOpen" label="Способ оплаты">
            <div class="overflow-y-auto flex-1">
              <button v-for="o in paymentMethodOptions" :key="o.value" type="button"
                :class="['w-full px-4 py-3 text-left text-sm border-b border-gray-50 active:bg-gray-100', o.value === payment.method ? 'bg-blue-50 text-blue-700 font-medium' : 'text-gray-900']"
                @click="handleMethodSheetSelect(o.value)">{{ o.label }}</button>
            </div>
          </BottomSheet>
        </template>
      </div>

      <!-- Payment terms -->
      <div data-tutorial="payment-terms">
        <label class="block text-sm font-medium text-gray-700 mb-1">Условия оплаты</label>
        <template v-if="!isMobile()">
          <Select :model-value="payment.terms" @update:model-value="handleTermsChange($event as PaymentTerms)">
            <SelectTrigger><SelectValue placeholder="Выберите условия оплаты" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="o in paymentTermsOptions" :key="o.value" :value="o.value">{{ o.label }}</SelectItem>
            </SelectContent>
          </Select>
        </template>
        <template v-else>
          <button type="button" class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-left bg-white" @click="termsSheetOpen = true">
            <span :class="selectedTermsLabel ? 'text-gray-900' : 'text-gray-400'">{{ selectedTermsLabel || 'Выберите условия оплаты' }}</span>
          </button>
          <BottomSheet v-model="termsSheetOpen" label="Условия оплаты">
            <div class="overflow-y-auto flex-1">
              <button v-for="o in paymentTermsOptions" :key="o.value" type="button"
                :class="['w-full px-4 py-3 text-left text-sm border-b border-gray-50 active:bg-gray-100', o.value === payment.terms ? 'bg-blue-50 text-blue-700 font-medium' : 'text-gray-900']"
                @click="handleTermsSheetSelect(o.value)">{{ o.label }}</button>
            </div>
          </BottomSheet>
        </template>
      </div>

      <!-- Deferred days -->
      <div v-if="showDeferredDays">
        <label class="block text-sm font-medium text-gray-700 mb-1">
          Дней отсрочки <span class="text-red-500">*</span>
        </label>
        <input
          type="number"
          :value="payment.deferred_days || ''"
          placeholder="30"
          min="1"
          step="1"
          :class="inputClass('deferred_days')"
          @input="handleDeferredDaysInput"
          @blur="emit('validateField', 'deferred_days')"
        />
        <p v-if="errors.deferred_days" class="mt-1 text-sm text-red-600">
          {{ errors.deferred_days }}
        </p>
      </div>

      <!-- Summary (только если указана цена) -->
      <div v-if="hasPrice" class="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h4 class="text-sm font-medium text-blue-900 mb-2">Итого</h4>
        <div class="text-2xl font-bold text-blue-900">
          {{ Number(displayAmount).toLocaleString('ru-RU') }}
          {{ currencyLabels[payment.price?.currency || 'RUB'] }}
        </div>
        <div class="text-sm text-blue-700 mt-1">
          {{ vatTypeOptions.find(o => o.value === payment.vat_type)?.label }}
          •
          {{ paymentTermsOptions.find(o => o.value === payment.terms)?.label }}
          <template v-if="showDeferredDays && payment.deferred_days">
            ({{ payment.deferred_days }} дн.)
          </template>
        </div>
      </div>
    </template>
  </div>
</template>
