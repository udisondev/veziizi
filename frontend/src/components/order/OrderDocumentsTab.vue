<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import type { Order, OrderDocument } from '@/types/order'
import { isOrderFinished } from '@/types/order'
import { formatDateTime, formatFileSize } from '@/utils/formatters'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

// Icons
import { Upload, Download, Trash2 } from 'lucide-vue-next'

interface Props {
  order: Order
  actionLoading?: boolean
}

interface Emits {
  (e: 'upload', file: File): void
  (e: 'download', doc: OrderDocument): void
  (e: 'remove', doc: OrderDocument): void
}

const props = withDefaults(defineProps<Props>(), {
  actionLoading: false,
})

const emit = defineEmits<Emits>()

const auth = useAuthStore()

const fileInput = ref<HTMLInputElement | null>(null)
const uploadingFile = ref(false)

const isResponsible = computed(() => {
  if (!auth.memberId) return false
  return props.order.customer_member_id === auth.memberId || props.order.carrier_member_id === auth.memberId
})

const canUploadDocument = computed(() => {
  if (!isResponsible.value) return false
  return !isOrderFinished(props.order.status)
})

function triggerFileUpload() {
  fileInput.value?.click()
}

function handleFileUpload(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  uploadingFile.value = true
  emit('upload', file)
  target.value = ''
}

// Called by parent after upload completes
function onUploadComplete() {
  uploadingFile.value = false
}

defineExpose({
  onUploadComplete,
})
</script>

<template>
  <div>
    <!-- Upload button -->
    <div v-if="canUploadDocument" class="mb-4">
      <input
        ref="fileInput"
        type="file"
        @change="handleFileUpload"
        class="hidden"
      />
      <Button
        :disabled="uploadingFile || actionLoading"
        @click="triggerFileUpload"
      >
        <Upload class="mr-2 h-4 w-4" />
        {{ uploadingFile ? 'Загрузка...' : 'Загрузить документ' }}
      </Button>
    </div>

    <!-- Documents List -->
    <div v-if="order.documents.length === 0" class="text-center text-muted-foreground py-8">
      Документов пока нет
    </div>

    <div v-else class="space-y-3">
      <Card
        v-for="doc in order.documents"
        :key="doc.id"
      >
        <CardContent class="flex items-center justify-between p-4">
          <div class="flex-1 min-w-0">
            <p class="font-medium text-foreground truncate">{{ doc.name }}</p>
            <p class="text-sm text-muted-foreground">
              {{ formatFileSize(doc.size) }} &middot; {{ formatDateTime(doc.created_at) }}
            </p>
          </div>
          <div class="flex gap-2 ml-4">
            <Button
              variant="ghost"
              size="sm"
              @click="emit('download', doc)"
            >
              <Download class="h-4 w-4" />
            </Button>
            <Button
              v-if="isResponsible && !isOrderFinished(order.status)"
              variant="ghost"
              size="sm"
              class="text-destructive hover:text-destructive"
              :disabled="actionLoading"
              @click="emit('remove', doc)"
            >
              <Trash2 class="h-4 w-4" />
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
