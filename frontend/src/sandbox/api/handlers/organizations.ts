/**
 * Mock Handlers for Organizations
 * Фейковые организации для sandbox режима (перевозчики из tutorial)
 */

import { registerHandler } from './index'
import type { OrganizationDetail, OrganizationRating } from '@/types/admin'

// Mock организации-перевозчики (соответствуют CARRIERS из offers.ts)
const MOCK_ORGANIZATIONS: Record<string, OrganizationDetail> = {
  'carrier-1': {
    id: 'carrier-1',
    name: 'ТрансЛогистик',
    inn: '7712345678',
    legal_name: 'ООО "ТрансЛогистик"',
    country: 'RU',
    phone: '+7 (495) 123-45-67',
    email: 'info@translogistic.ru',
    address: 'г. Москва, ул. Логистическая, д. 1',
    status: 'active',
    members: [
      {
        id: 'carrier-1-member',
        email: 'petrov@translogistic.ru',
        name: 'Иван Петров',
        phone: '+7 (495) 123-45-68',
        role: 'owner',
        status: 'active',
        created_at: '2024-01-15T10:00:00Z',
      },
    ],
    created_at: '2024-01-15T10:00:00Z',
  },
  'carrier-2': {
    id: 'carrier-2',
    name: 'СпецГруз',
    inn: '7723456789',
    legal_name: 'ООО "СпецГруз"',
    country: 'RU',
    phone: '+7 (495) 234-56-78',
    email: 'info@specgruz.ru',
    address: 'г. Москва, ул. Грузовая, д. 5',
    status: 'active',
    members: [
      {
        id: 'carrier-2-member',
        email: 'smirnov@specgruz.ru',
        name: 'Алексей Смирнов',
        phone: '+7 (495) 234-56-79',
        role: 'owner',
        status: 'active',
        created_at: '2024-02-10T12:00:00Z',
      },
    ],
    created_at: '2024-02-10T12:00:00Z',
  },
  'carrier-3': {
    id: 'carrier-3',
    name: 'МегаФура',
    inn: '7734567890',
    legal_name: 'ООО "МегаФура"',
    country: 'RU',
    phone: '+7 (495) 345-67-89',
    email: 'info@megafura.ru',
    address: 'г. Москва, ул. Транспортная, д. 10',
    status: 'active',
    members: [
      {
        id: 'carrier-3-member',
        email: 'kozlov@megafura.ru',
        name: 'Дмитрий Козлов',
        phone: '+7 (495) 345-67-90',
        role: 'owner',
        status: 'active',
        created_at: '2024-03-05T14:00:00Z',
      },
    ],
    created_at: '2024-03-05T14:00:00Z',
  },
}

// Mock рейтинги организаций
const MOCK_RATINGS: Record<string, OrganizationRating> = {
  'carrier-1': { total_reviews: 47, average_rating: 4.8 },
  'carrier-2': { total_reviews: 23, average_rating: 4.5 },
  'carrier-3': { total_reviews: 31, average_rating: 4.6 },
}

export function organizationsHandlers(): void {
  // Get organization by ID
  registerHandler('GET', '/organizations/:id', (params) => {
    const org = MOCK_ORGANIZATIONS[params.id!]

    if (!org) {
      // Если организация не найдена в mock store — пропускаем к реальному API
      return null
    }

    return { data: org }
  })

  // Get organization rating
  registerHandler('GET', '/organizations/:id/rating', (params) => {
    const rating = MOCK_RATINGS[params.id!]

    if (!rating) {
      // Если рейтинг не найден — возвращаем дефолтный
      return {
        data: { total_reviews: 0, average_rating: 0 },
      }
    }

    return { data: rating }
  })

  // Get organization reviews (пустой список для mock)
  registerHandler('GET', '/organizations/:id/reviews', (params) => {
    // Для mock возвращаем несколько фейковых отзывов
    const orgName = MOCK_ORGANIZATIONS[params.id!]?.name

    if (!orgName) {
      return null
    }

    return {
      data: {
        items: [
          {
            id: 'review-1',
            order_id: 'order-1',
            reviewer_org_id: 'customer-1',
            reviewer_org_name: 'ООО Ромашка',
            rating: 5,
            comment: 'Отличный перевозчик! Доставили вовремя и аккуратно.',
            created_at: '2024-12-01T10:00:00Z',
          },
          {
            id: 'review-2',
            order_id: 'order-2',
            reviewer_org_id: 'customer-2',
            reviewer_org_name: 'ИП Сидоров',
            rating: 4,
            comment: 'Хорошая работа, небольшая задержка на погрузке.',
            created_at: '2024-11-15T14:30:00Z',
          },
        ],
        total: 2,
      },
    }
  })
}
