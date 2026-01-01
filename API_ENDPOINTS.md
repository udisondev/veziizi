# API Endpoints Summary

Полный список всех HTTP endpoints проекта veziizi.

**Всего endpoints: 73**

---

## 1. Authentication & Authorization (AuthHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| POST | `/api/v1/auth/login` | Вход по email/password |
| POST | `/api/v1/auth/logout` | Выход из системы |
| GET | `/api/v1/auth/me` | Получить текущего пользователя и организацию |
| GET | `/api/v1/members/{id}` | Получить публичный профиль члена организации |

---

## 2. Organizations (OrganizationHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| POST | `/api/v1/organizations` | Регистрация новой организации с владельцем |
| GET | `/api/v1/organizations/{id}` | Получить публичный профиль организации |
| GET | `/api/v1/organizations/{id}/full` | Получить полные данные организации с членами |
| GET | `/api/v1/organizations/{id}/rating` | Получить рейтинг организации |
| GET | `/api/v1/organizations/{id}/reviews` | Получить список отзывов на организацию |
| POST | `/api/v1/organizations/{id}/invitations` | Создать приглашение для сотрудника |
| GET | `/api/v1/organizations/{id}/invitations` | Получить список приглашений |
| DELETE | `/api/v1/organizations/{id}/invitations/{invitationId}` | Отменить приглашение |
| GET | `/api/v1/invitations/{token}` | Получить данные приглашения по токену |
| POST | `/api/v1/invitations/{token}/accept` | Принять приглашение |
| PATCH | `/api/v1/organizations/{id}/members/{memberId}/role` | Изменить роль члена |
| POST | `/api/v1/organizations/{id}/members/{memberId}/block` | Заблокировать сотрудника |
| POST | `/api/v1/organizations/{id}/members/{memberId}/unblock` | Разблокировать сотрудника |

---

## 3. Freight Requests & Offers (FreightRequestHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| POST | `/api/v1/freight-requests` | Создать заявку на перевозку |
| GET | `/api/v1/freight-requests` | Получить список заявок с фильтрами |
| GET | `/api/v1/freight-requests/{id}` | Получить детали заявки |
| PATCH | `/api/v1/freight-requests/{id}` | Обновить заявку |
| DELETE | `/api/v1/freight-requests/{id}` | Отменить заявку |
| POST | `/api/v1/freight-requests/{id}/reassign` | Переназначить ответственного |
| POST | `/api/v1/freight-requests/{id}/offers` | Создать оффер на перевозку |
| GET | `/api/v1/freight-requests/{id}/offers` | Получить список офферов |
| DELETE | `/api/v1/freight-requests/{id}/offers/{offerId}` | Отозвать оффер |
| POST | `/api/v1/freight-requests/{id}/offers/{offerId}/select` | Выбрать оффер |
| POST | `/api/v1/freight-requests/{id}/offers/{offerId}/reject` | Отклонить оффер (заказчик) |
| POST | `/api/v1/freight-requests/{id}/offers/{offerId}/confirm` | Подтвердить оффер (создает заказ) |
| POST | `/api/v1/freight-requests/{id}/offers/{offerId}/decline` | Отклонить оффер после выбора |
| GET | `/api/v1/offers` | Получить мои офферы |

---

## 4. Orders (OrderHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| GET | `/api/v1/orders` | Получить список заказов |
| GET | `/api/v1/orders/{id}` | Получить детали заказа |
| POST | `/api/v1/orders/{id}/messages` | Отправить сообщение в чате |
| POST | `/api/v1/orders/{id}/documents` | Загрузить документ (max 10MB) |
| GET | `/api/v1/orders/{id}/documents/{docId}` | Скачать документ |
| DELETE | `/api/v1/orders/{id}/documents/{docId}` | Удалить документ |
| POST | `/api/v1/orders/{id}/complete` | Отметить заказ завершённым |
| POST | `/api/v1/orders/{id}/cancel` | Отменить заказ |
| POST | `/api/v1/orders/{id}/review` | Оставить отзыв |
| POST | `/api/v1/orders/{id}/reassign` | Переназначить ответственного |

---

## 5. Geo Data (GeoHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| GET | `/api/v1/geo/countries` | Получить список стран |
| GET | `/api/v1/geo/countries/{id}` | Получить страну по ID |
| GET | `/api/v1/geo/countries/{id}/cities` | Получить города страны с поиском |
| GET | `/api/v1/geo/cities/{id}` | Получить город по ID |

---

## 6. History (HistoryHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| GET | `/api/v1/organizations/{id}/history` | История событий организации |
| GET | `/api/v1/freight-requests/{id}/history` | История заявки |
| GET | `/api/v1/orders/{id}/history` | История заказа |

---

## 7. Admin Panel (AdminHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| POST | `/api/v1/admin/auth/login` | Вход администратора |
| POST | `/api/v1/admin/auth/logout` | Выход администратора |
| GET | `/api/v1/admin/organizations` | Список организаций на модерации |
| GET | `/api/v1/admin/organizations/{id}` | Детали организации для модерации |
| POST | `/api/v1/admin/organizations/{id}/approve` | Одобрить организацию |
| POST | `/api/v1/admin/organizations/{id}/reject` | Отклонить организацию |
| POST | `/api/v1/admin/organizations/{id}/mark-fraudster` | Отметить мошенником |
| POST | `/api/v1/admin/organizations/{id}/unmark-fraudster` | Убрать флаг мошенника |
| GET | `/api/v1/admin/fraudsters` | Список мошенников |
| GET | `/api/v1/admin/reviews` | Отзывы на модерацию |
| GET | `/api/v1/admin/reviews/{id}` | Детали отзыва |
| POST | `/api/v1/admin/reviews/{id}/approve` | Одобрить отзыв |
| POST | `/api/v1/admin/reviews/{id}/reject` | Отклонить отзыв |

---

## 8. Notifications (NotificationHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| GET | `/api/v1/notifications` | Получить in-app уведомления |
| GET | `/api/v1/notifications/unread-count` | Количество непрочитанных |
| POST | `/api/v1/notifications/read` | Отметить прочитанными |
| POST | `/api/v1/notifications/read-all` | Отметить все прочитанными |
| GET | `/api/v1/notifications/preferences` | Настройки уведомлений |
| PATCH | `/api/v1/notifications/preferences` | Обновить настройки |
| POST | `/api/v1/notifications/telegram/link-code` | Код для привязки Telegram |
| DELETE | `/api/v1/notifications/telegram` | Отключить Telegram |

---

## 9. Subscriptions (SubscriptionsHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| GET | `/api/v1/subscriptions` | Получить подписки на заявки |
| POST | `/api/v1/subscriptions` | Создать подписку |
| GET | `/api/v1/subscriptions/{id}` | Получить подписку по ID |
| PUT | `/api/v1/subscriptions/{id}` | Обновить подписку |
| DELETE | `/api/v1/subscriptions/{id}` | Удалить подписку |
| PATCH | `/api/v1/subscriptions/{id}/active` | Включить/выключить подписку |

---

## 10. Support Tickets (SupportHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| GET | `/api/v1/support/faq` | Получить FAQ |
| POST | `/api/v1/support/tickets` | Создать тикет поддержки |
| GET | `/api/v1/support/tickets` | Получить свои тикеты |
| GET | `/api/v1/support/tickets/{id}` | Детали тикета |
| POST | `/api/v1/support/tickets/{id}/messages` | Добавить сообщение |
| POST | `/api/v1/support/tickets/{id}/reopen` | Переоткрыть тикет |

---

## 11. Admin Support (AdminSupportHandler)

| Метод | URL | Описание |
|-------|-----|---------|
| GET | `/api/v1/admin/support/tickets` | Все тикеты |
| GET | `/api/v1/admin/support/tickets/{id}` | Тикет с сообщениями |
| POST | `/api/v1/admin/support/tickets/{id}/messages` | Ответ администратора |
| POST | `/api/v1/admin/support/tickets/{id}/close` | Закрыть тикет |

---

## 12. Dev User Switcher (DevHandler) *[Development only]*

| Метод | URL | Описание |
|-------|-----|---------|
| POST | `/api/v1/dev/login` | Переключиться на другого пользователя |

---

## Безопасность

### Middleware (применяется в порядке):
1. **SecurityHeaders** — заголовки безопасности (CSP, X-Frame-Options)
2. **CORS** — настройка кросс-доменных запросов
3. **BodyLimit** — лимит размера тела (10MB для документов)
4. **RequireAuth** — проверка сессии
5. **RateLimiter** — ограничение запросов + анализ фрода
6. **CSRFProtection** — проверка X-Requested-With header

### Публичные endpoints (без авторизации):
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `POST /api/v1/organizations` (регистрация)
- `GET /api/v1/organizations/{id}` (публичный профиль)
- `GET /api/v1/organizations/{id}/rating`
- `GET /api/v1/organizations/{id}/reviews`
- `GET /api/v1/invitations/{token}`
- `POST /api/v1/invitations/{token}/accept`
- `GET /api/v1/geo/*` (все geo endpoints)
- `GET /api/v1/support/faq`
- `POST /api/v1/admin/auth/login`
