/**
 * Composable для управления модальными окнами
 * Устраняет дублирование паттерна showModal/formData/loading во всех views
 */

import { ref, reactive, type Ref, type UnwrapNestedRefs } from 'vue'

export interface UseModalOptions<T> {
  /**
   * Начальные данные формы
   */
  initialData?: T

  /**
   * Callback при открытии модала
   */
  onOpen?: () => void

  /**
   * Callback при закрытии модала
   */
  onClose?: () => void
}

export interface UseModalReturn<T> {
  /**
   * Открыт ли модал
   */
  isOpen: Ref<boolean>

  /**
   * Данные формы
   */
  formData: UnwrapNestedRefs<T>

  /**
   * Состояние загрузки
   */
  isLoading: Ref<boolean>

  /**
   * Ошибка
   */
  error: Ref<string | null>

  /**
   * Открыть модал
   */
  open: (data?: Partial<T>) => void

  /**
   * Закрыть модал
   */
  close: () => void

  /**
   * Сбросить данные формы
   */
  resetForm: () => void

  /**
   * Установить ошибку
   */
  setError: (message: string | null) => void

  /**
   * Установить состояние загрузки
   */
  setLoading: (loading: boolean) => void
}

/**
 * Composable для управления модальным окном с формой
 *
 * @example
 * ```ts
 * const cancelModal = useModal({
 *   initialData: { reason: '' }
 * })
 *
 * // Открытие
 * cancelModal.open({ reason: 'По запросу клиента' })
 *
 * // В шаблоне
 * <Dialog :open="cancelModal.isOpen.value" @update:open="val => val || cancelModal.close()">
 *   <Input v-model="cancelModal.formData.reason" />
 *   <Button :disabled="cancelModal.isLoading.value" @click="handleSubmit">
 *     Отменить
 *   </Button>
 * </Dialog>
 * ```
 */
export function useModal<T extends Record<string, unknown>>(
  options: UseModalOptions<T> = {}
): UseModalReturn<T> {
  const { initialData = {} as T, onOpen, onClose } = options

  const isOpen = ref(false)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const formData = reactive<T>({ ...initialData })

  function open(data?: Partial<T>): void {
    // Сбросить состояние
    error.value = null
    isLoading.value = false

    // Применить начальные данные
    Object.assign(formData, { ...initialData })

    // Применить переданные данные
    if (data) {
      Object.assign(formData, data)
    }

    isOpen.value = true
    onOpen?.()
  }

  function close(): void {
    isOpen.value = false
    onClose?.()
  }

  function resetForm(): void {
    Object.assign(formData, { ...initialData })
    error.value = null
  }

  function setError(message: string | null): void {
    error.value = message
  }

  function setLoading(loading: boolean): void {
    isLoading.value = loading
  }

  return {
    isOpen,
    formData,
    isLoading,
    error,
    open,
    close,
    resetForm,
    setError,
    setLoading,
  }
}

/**
 * Composable для простого модала подтверждения (без формы)
 *
 * @example
 * ```ts
 * const deleteConfirm = useConfirmModal()
 *
 * // Открытие с данными
 * deleteConfirm.open({ id: 'item-123', name: 'Заявка #1' })
 *
 * // В шаблоне
 * <ConfirmDialog
 *   :open="deleteConfirm.isOpen.value"
 *   title="Удалить?"
 *   :description="`Вы уверены что хотите удалить ${deleteConfirm.data.value?.name}?`"
 *   @confirm="handleDelete(deleteConfirm.data.value?.id)"
 *   @cancel="deleteConfirm.close()"
 * />
 * ```
 */
export function useConfirmModal<T = unknown>(): {
  isOpen: Ref<boolean>
  data: Ref<T | null>
  isLoading: Ref<boolean>
  open: (data: T) => void
  close: () => void
  setLoading: (loading: boolean) => void
} {
  const isOpen = ref(false)
  const data = ref<T | null>(null) as Ref<T | null>
  const isLoading = ref(false)

  function open(payload: T): void {
    data.value = payload
    isLoading.value = false
    isOpen.value = true
  }

  function close(): void {
    isOpen.value = false
    data.value = null
  }

  function setLoading(loading: boolean): void {
    isLoading.value = loading
  }

  return {
    isOpen,
    data,
    isLoading,
    open,
    close,
    setLoading,
  }
}

/**
 * Composable для группы связанных модалов
 * Гарантирует что открыт только один модал
 *
 * @example
 * ```ts
 * const modals = useModalGroup(['cancel', 'reassign', 'makeOffer'] as const)
 *
 * modals.open('cancel')
 * modals.isOpen('cancel') // true
 * modals.isOpen('reassign') // false
 *
 * modals.open('reassign') // автоматически закроет 'cancel'
 * ```
 */
export function useModalGroup<T extends readonly string[]>(
  names: T
): {
  activeModal: Ref<T[number] | null>
  isOpen: (name: T[number]) => boolean
  open: (name: T[number]) => void
  close: () => void
} {
  const activeModal = ref<T[number] | null>(null) as Ref<T[number] | null>

  function isOpen(name: T[number]): boolean {
    return activeModal.value === name
  }

  function open(name: T[number]): void {
    if (!names.includes(name)) {
      console.warn(`Modal "${name}" is not registered`)
      return
    }
    activeModal.value = name
  }

  function close(): void {
    activeModal.value = null
  }

  return {
    activeModal,
    isOpen,
    open,
    close,
  }
}
