<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { freightRequestsApi } from '@/api/freightRequests'
import { useSandboxReady } from '@/composables/useSandboxReady'
import type { FreightRequest, Currency, VatType, PaymentMethod } from '@/types/freightRequest'
import { currencyOptions, vatTypeOptions, paymentMethodOptions } from '@/types/freightRequest'
import { centsToAmount, amountToCents } from '@/utils/formatters'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import SelectField from '@/components/ui/select-field/SelectField.vue'
import { DetailPageHeader, LoadingSpinner, ErrorBanner } from '@/components/shared'

const route = useRoute()
const router = useRouter()
const { waitForReady } = useSandboxReady()

const freightRequestId = route.params.id as string

const isLoading = ref(false)
const isSubmitting = ref(false)
const error = ref('')
const freightRequest = ref<FreightRequest | null>(null)

const form = ref({
  price: { amount: 0, currency: 'RUB' as Currency },
  comment: '',
  vat_type: 'included' as VatType,
  payment_method: 'bank_transfer' as PaymentMethod,
})
const acceptRequestTerms = ref(false)

const hasRequestRate = computed(() =>
  !!(freightRequest.value?.payment?.price?.amount && freightRequest.value.payment.price.amount > 0)
)

const isValid = computed(() => form.value.price.amount > 0)

watch(acceptRequestTerms, (accepted) => {
  if (!freightRequest.value) return
  if (accepted && hasRequestRate.value) {
    const payment = freightRequest.value.payment
    form.value.price.amount = centsToAmount(payment.price!.amount)
    form.value.price.currency = payment.price!.currency
    form.value.vat_type = payment.vat_type
    form.value.payment_method = payment.method
  } else {
    form.value.price.amount = 0
    form.value.price.currency = 'RUB'
    form.value.vat_type = 'included'
    form.value.payment_method = 'bank_transfer'
  }
})

async function loadData() {
  isLoading.value = true
  error.value = ''
  await waitForReady()
  try {
    freightRequest.value = await freightRequestsApi.get(freightRequestId)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

async function handleSubmit() {
  if (!isValid.value) return
  isSubmitting.value = true
  error.value = ''
  try {
    await freightRequestsApi.makeOffer(freightRequestId, {
      price: {
        amount: amountToCents(form.value.price.amount),
        currency: form.value.price.currency,
      },
      comment: form.value.comment || undefined,
      vat_type: form.value.vat_type,
      payment_method: form.value.payment_method,
    })
    router.push({ name: 'freight-request-detail', params: { id: freightRequestId } })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка отправки предложения'
  } finally {
    isSubmitting.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="min-h-screen bg-background">
    <DetailPageHeader
      :back-to="{ path: '/', query: { tab: 'list' } }"
      back-label="К списку заявок"
    />

    <div class="max-w-lg mx-auto px-4 py-6">
      <LoadingSpinner v-if="isLoading" text="Загрузка..." class="py-12" />

      <template v-else>
        <ErrorBanner v-if="error" :message="error" class="mb-4" />

        <Card>
          <CardHeader>
            <CardTitle>Сделать предложение</CardTitle>
          </CardHeader>
          <CardContent>
            <form class="space-y-5" @submit.prevent="handleSubmit">
              <!-- Согласен с условиями заказчика -->
              <div v-if="hasRequestRate" class="flex items-center gap-3 rounded-lg border border-border bg-muted/40 px-4 py-3">
                <Checkbox
                  id="accept-terms"
                  :checked="acceptRequestTerms"
                  @update:checked="acceptRequestTerms = $event"
                />
                <Label for="accept-terms" class="text-sm cursor-pointer mb-0 inline leading-none">
                  Согласен с условиями заказчика
                </Label>
              </div>

              <!-- Цена -->
              <div>
                <Label for="price">Цена *</Label>
                <div class="flex gap-2">
                  <Input
                    id="price"
                    v-model.number="form.price.amount"
                    type="number"
                    min="0"
                    step="100"
                    placeholder="0"
                    class="flex-[7] min-w-0"
                    :disabled="acceptRequestTerms"
                  />
                  <div class="flex-[3] min-w-0">
                    <SelectField
                      v-model="form.price.currency"
                      :options="currencyOptions"
                      sheet-label="Валюта"
                      :disabled="acceptRequestTerms"
                    />
                  </div>
                </div>
              </div>

              <!-- НДС -->
              <div>
                <Label>НДС</Label>
                <SelectField
                  v-model="form.vat_type"
                  :options="vatTypeOptions"
                  sheet-label="НДС"
                  :disabled="acceptRequestTerms"
                />
              </div>

              <!-- Способ оплаты -->
              <div>
                <Label>Способ оплаты</Label>
                <SelectField
                  v-model="form.payment_method"
                  :options="paymentMethodOptions"
                  sheet-label="Способ оплаты"
                  :disabled="acceptRequestTerms"
                />
              </div>

              <!-- Комментарий -->
              <div>
                <Label for="comment">Комментарий</Label>
                <Textarea
                  id="comment"
                  v-model="form.comment"
                  rows="3"
                  placeholder="Дополнительная информация..."
                />
              </div>

              <!-- Кнопки -->
              <div class="flex gap-3 pt-2">
                <Button
                  type="button"
                  variant="outline"
                  class="flex-1 h-12 text-base md:h-10 md:text-sm"
                  @click="router.push({ name: 'freight-request-detail', params: { id: freightRequestId } })"
                >
                  Отмена
                </Button>
                <Button
                  type="submit"
                  class="flex-1 h-12 text-base md:h-10 md:text-sm"
                  :disabled="!isValid || isSubmitting"
                >
                  {{ isSubmitting ? 'Отправка...' : 'Отправить' }}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </template>
    </div>
  </div>
</template>
