<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import type { Order, OrderMessage } from '@/types/order'
import { isOrderCancelled } from '@/types/order'
import { formatDateTime } from '@/utils/formatters'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

// Icons
import { Send } from 'lucide-vue-next'

interface Props {
  order: Order
  actionLoading?: boolean
}

interface Emits {
  (e: 'send', message: string): void
}

const props = withDefaults(defineProps<Props>(), {
  actionLoading: false,
})

const emit = defineEmits<Emits>()

const auth = useAuthStore()

const messageInput = ref('')
const messagesContainer = ref<HTMLDivElement | null>(null)

const isResponsible = computed(() => {
  if (!auth.memberId) return false
  return props.order.customer_member_id === auth.memberId || props.order.carrier_member_id === auth.memberId
})

const canSendMessage = computed(() => {
  if (!isResponsible.value) return false
  return !isOrderCancelled(props.order.status)
})

const sortedMessages = computed(() => {
  return [...props.order.messages].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  )
})

function isMyMessage(msg: OrderMessage): boolean {
  return msg.sender_org_id === auth.organizationId
}

function getMessageSenderLabel(msg: OrderMessage): string {
  if (msg.sender_org_id === props.order.customer_org_id) return 'Заказчик'
  if (msg.sender_org_id === props.order.carrier_org_id) return 'Перевозчик'
  return ''
}

async function scrollToBottom() {
  await nextTick()
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

function handleSendMessage() {
  if (!messageInput.value.trim()) return
  emit('send', messageInput.value.trim())
  messageInput.value = ''
}

// Expose for parent to call after successful send
defineExpose({
  scrollToBottom,
})

onMounted(() => {
  scrollToBottom()
})
</script>

<template>
  <div>
    <!-- Messages List -->
    <div
      ref="messagesContainer"
      class="h-64 sm:h-80 overflow-y-auto border rounded-lg p-3 sm:p-4 mb-4 space-y-3"
    >
      <div v-if="sortedMessages.length === 0" class="text-center text-muted-foreground py-8">
        Сообщений пока нет
      </div>

      <div
        v-for="msg in sortedMessages"
        :key="msg.id"
        :class="[
          'max-w-[80%] p-3 rounded-lg',
          isMyMessage(msg)
            ? 'ml-auto bg-primary/10 text-foreground'
            : 'bg-muted text-foreground'
        ]"
      >
        <div class="text-xs text-muted-foreground mb-1">
          {{ getMessageSenderLabel(msg) }} &middot; {{ formatDateTime(msg.created_at) }}
        </div>
        <div class="whitespace-pre-wrap break-words">{{ msg.content }}</div>
      </div>
    </div>

    <!-- Message Input -->
    <div v-if="canSendMessage" class="flex flex-col gap-2 sm:flex-row">
      <Input
        v-model="messageInput"
        @keyup.enter="handleSendMessage"
        placeholder="Введите сообщение..."
        :disabled="actionLoading"
        class="flex-1"
      />
      <Button
        :disabled="!messageInput.trim() || actionLoading"
        @click="handleSendMessage"
      >
        <Send class="mr-2 h-4 w-4" />
        Отправить
      </Button>
    </div>
    <div v-else-if="isOrderCancelled(order.status)" class="text-sm text-muted-foreground">
      Отправка сообщений недоступна для отменённого заказа
    </div>
  </div>
</template>
