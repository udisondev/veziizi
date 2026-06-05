<script setup lang="ts">
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { FormField } from '@/components/ui/form-field'
import { computed } from 'vue'
import { parseInputInt, parseInputString } from '@/utils/inputParsers'
import { centsToAmount, amountToCents } from '@/utils/formatters'
import type { Payment, Currency, VatType, PaymentMethod, PaymentTerms } from '@/types/freightRequest'
import {
  currencyOptions,
  vatTypeOptions,
  paymentMethodOptions,
  paymentTermsOptions,
  currencyLabels,
} from '@/types/freightRequest'
import { SelectField } from '@/components/ui/select-field'

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

function updateField<K extends keyof Payment>(field: K, value: Payment[K]) {
  emit('update:payment', { ...props.payment, [field]: value })
}

// Конвертация рублей в копейки для отображения/ввода
const displayAmount = computed(() => {
  if (!props.payment.price?.amount) return ''
  return centsToAmount(props.payment.price.amount).toString()
})

function handleAmountInput(event: Event) {
  const inputValue = parseInputString(event)

  if (!inputValue) {
    updateField('price', undefined)
    return
  }

  const currentCurrency = props.payment.price?.currency || 'RUB'
  updateField('price', { amount: amountToCents(parseFloat(inputValue) || 0), currency: currentCurrency })
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

function handleTermsChange(terms: PaymentTerms) {
  updateField('terms', terms)
  // Очищаем дни отсрочки если не deferred
  if (terms !== 'deferred') {
    updateField('deferred_days', undefined)
  }
}

function handleDeferredDaysInput(event: Event) {
  updateField('deferred_days', parseInputInt(event))
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

</script>

<template>
  <div class="space-y-6">
    <!-- Галочка "Не указывать цену" -->
    <div data-tutorial="payment-no-price" class="flex items-center gap-3">
      <input
        id="no-price"
        type="checkbox"
        :checked="payment.no_price"
        class="h-4 w-4 accent-primary rounded"
        @change="handleNoPriceChange"
      />
      <Label for="no-price" class="mb-0 inline cursor-pointer">
        Не указывать цену
        <span class="text-muted-foreground">(перевозчики предложат свою)</span>
      </Label>
    </div>

    <!-- Price (показываем только если галочка не отмечена) -->
    <FormField
      v-if="!payment.no_price"
      label="Стоимость перевозки"
      required
      :error="errors.price"
      data-tutorial="payment-price"
    >
      <div class="relative">
        <Input
          type="number"
          :value="displayAmount"
          placeholder="Укажите сумму"
          min="0"
          step="100"
          :has-error="!!errors.price"
          @input="handleAmountInput"
          @blur="emit('validateField', 'price')"
        />
        <span v-if="hasPrice" class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground">
          {{ currencyLabels[payment.price?.currency || 'RUB'] }}
        </span>
      </div>
    </FormField>

    <!-- Currency -->
    <div v-if="!payment.no_price" data-tutorial="payment-currency">
      <Label>Валюта</Label>
      <SelectField
        :model-value="payment.price?.currency || 'RUB'"
        :options="currencyOptions"
        placeholder="Выберите валюту"
        sheet-label="Валюта"
        @update:model-value="handleCurrencyChange($event as Currency)"
      />
    </div>

    <!-- Все остальные поля оплаты показываем только если цена указывается -->
    <template v-if="!payment.no_price">
      <!-- VAT type -->
      <div data-tutorial="payment-vat">
        <Label>НДС</Label>
        <SelectField
          :model-value="payment.vat_type"
          :options="vatTypeOptions"
          placeholder="Выберите тип НДС"
          sheet-label="НДС"
          @update:model-value="handleVatTypeChange($event as VatType)"
        />
      </div>

      <!-- Payment method -->
      <div data-tutorial="payment-method">
        <Label>Способ оплаты</Label>
        <SelectField
          :model-value="payment.method"
          :options="paymentMethodOptions"
          placeholder="Выберите способ оплаты"
          sheet-label="Способ оплаты"
          @update:model-value="handleMethodChange($event as PaymentMethod)"
        />
      </div>

      <!-- Payment terms -->
      <div data-tutorial="payment-terms">
        <Label>Условия оплаты</Label>
        <SelectField
          :model-value="payment.terms"
          :options="paymentTermsOptions"
          placeholder="Выберите условия оплаты"
          sheet-label="Условия оплаты"
          @update:model-value="handleTermsChange($event as PaymentTerms)"
        />
      </div>

      <!-- Deferred days -->
      <FormField
        v-if="showDeferredDays"
        label="Дней отсрочки"
        required
        :error="errors.deferred_days"
      >
        <Input
          type="number"
          :value="payment.deferred_days || ''"
          placeholder="30"
          min="1"
          step="1"
          :has-error="!!errors.deferred_days"
          @input="handleDeferredDaysInput"
          @blur="emit('validateField', 'deferred_days')"
        />
      </FormField>

      <!-- Summary (только если указана цена) -->
      <div v-if="hasPrice" class="bg-accent border border-primary/20 rounded-lg p-4">
        <h4 class="text-sm font-medium text-accent-foreground mb-2">Итого</h4>
        <div class="text-2xl font-bold text-accent-foreground">
          {{ Number(displayAmount).toLocaleString('ru-RU') }}
          {{ currencyLabels[payment.price?.currency || 'RUB'] }}
        </div>
        <div class="text-sm text-accent-foreground/80 mt-1">
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
