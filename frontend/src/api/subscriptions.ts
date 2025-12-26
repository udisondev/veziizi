import { api } from './client'
import type {
  FreightSubscription,
  FreightSubscriptionCreate,
  FreightSubscriptionUpdate,
} from '@/types/subscription'

export const subscriptionsApi = {
  // Получить список подписок пользователя
  async list(): Promise<FreightSubscription[]> {
    const result = await api.get<FreightSubscription[] | null>('/subscriptions')
    return result ?? []
  },

  // Получить подписку по ID
  async get(id: string): Promise<FreightSubscription> {
    return api.get(`/subscriptions/${id}`)
  },

  // Создать подписку
  async create(data: FreightSubscriptionCreate): Promise<FreightSubscription> {
    return api.post('/subscriptions', data)
  },

  // Обновить подписку
  async update(id: string, data: FreightSubscriptionUpdate): Promise<FreightSubscription> {
    return api.put(`/subscriptions/${id}`, data)
  },

  // Удалить подписку
  async delete(id: string): Promise<void> {
    await api.delete(`/subscriptions/${id}`)
  },

  // Включить/выключить подписку
  async setActive(id: string, isActive: boolean): Promise<FreightSubscription> {
    return api.patch(`/subscriptions/${id}/active`, { is_active: isActive })
  },
}
