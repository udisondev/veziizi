/**
 * Mock Data for Email Templates
 */

import type { EmailTemplate, VariableSpec } from '@/types/admin'

interface EmailTemplateStore {
  templates: EmailTemplate[]
}

const store: EmailTemplateStore = {
  templates: [
    {
      id: 'tpl-001',
      slug: 'password-reset',
      name: 'Сброс пароля',
      subject: 'Сброс пароля для {{.Name}}',
      body_html: `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .button { display: inline-block; padding: 12px 24px; background: #4F46E5; color: white; text-decoration: none; border-radius: 6px; }
    .footer { margin-top: 30px; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Сброс пароля</h1>
    <p>Здравствуйте, {{.Name}}!</p>
    <p>Вы запросили сброс пароля для вашего аккаунта Veziizi.</p>
    <p>Нажмите на кнопку ниже, чтобы установить новый пароль:</p>
    <p><a href="{{.ResetLink}}" class="button">Сбросить пароль</a></p>
    <p>Ссылка действительна в течение 1 часа.</p>
    <p>Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.</p>
    <div class="footer">
      <p>С уважением,<br>Команда Veziizi</p>
    </div>
  </div>
</body>
</html>`,
      body_text: `Сброс пароля

Здравствуйте, {{.Name}}!

Вы запросили сброс пароля для вашего аккаунта Veziizi.

Перейдите по ссылке, чтобы установить новый пароль:
{{.ResetLink}}

Ссылка действительна в течение 1 часа.

Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.

С уважением,
Команда Veziizi`,
      category: 'transactional',
      variables_schema: {
        Name: { type: 'string', required: true, description: 'Имя пользователя' },
        ResetLink: { type: 'string', required: true, description: 'Ссылка для сброса пароля' },
      },
      is_system: true,
      is_active: true,
      created_at: '2026-01-15T10:00:00Z',
      updated_at: '2026-01-15T10:00:00Z',
    },
    {
      id: 'tpl-002',
      slug: 'email-verification',
      name: 'Подтверждение email',
      subject: 'Подтвердите ваш email',
      body_html: `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .code { font-size: 32px; font-weight: bold; letter-spacing: 4px; color: #4F46E5; }
    .footer { margin-top: 30px; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Подтверждение email</h1>
    <p>Здравствуйте, {{.Name}}!</p>
    <p>Для подтверждения вашего email введите код:</p>
    <p class="code">{{.Code}}</p>
    <p>Код действителен в течение 15 минут.</p>
    <div class="footer">
      <p>С уважением,<br>Команда Veziizi</p>
    </div>
  </div>
</body>
</html>`,
      body_text: `Подтверждение email

Здравствуйте, {{.Name}}!

Для подтверждения вашего email введите код: {{.Code}}

Код действителен в течение 15 минут.

С уважением,
Команда Veziizi`,
      category: 'transactional',
      variables_schema: {
        Name: { type: 'string', required: true, description: 'Имя пользователя' },
        Code: { type: 'string', required: true, description: 'Код подтверждения' },
      },
      is_system: true,
      is_active: true,
      created_at: '2026-01-15T10:00:00Z',
      updated_at: '2026-01-15T10:00:00Z',
    },
    {
      id: 'tpl-003',
      slug: 'offer-received',
      name: 'Новое предложение',
      subject: 'Новое предложение по заявке #{{.RequestNumber}}',
      body_html: `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .button { display: inline-block; padding: 12px 24px; background: #4F46E5; color: white; text-decoration: none; border-radius: 6px; }
    .details { background: #f5f5f5; padding: 15px; border-radius: 6px; margin: 20px 0; }
    .footer { margin-top: 30px; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Новое предложение</h1>
    <p>Здравствуйте!</p>
    <p>По вашей заявке #{{.RequestNumber}} поступило новое предложение от <strong>{{.CarrierName}}</strong>.</p>
    <div class="details">
      <p><strong>Стоимость:</strong> {{.Price}} {{.Currency}}</p>
      <p><strong>Комментарий:</strong> {{.Comment}}</p>
    </div>
    <p><a href="{{.RequestLink}}" class="button">Посмотреть предложение</a></p>
    <div class="footer">
      <p>С уважением,<br>Команда Veziizi</p>
    </div>
  </div>
</body>
</html>`,
      body_text: `Новое предложение

Здравствуйте!

По вашей заявке #{{.RequestNumber}} поступило новое предложение от {{.CarrierName}}.

Стоимость: {{.Price}} {{.Currency}}
Комментарий: {{.Comment}}

Посмотреть предложение: {{.RequestLink}}

С уважением,
Команда Veziizi`,
      category: 'transactional',
      variables_schema: {
        RequestNumber: { type: 'string', required: true, description: 'Номер заявки' },
        CarrierName: { type: 'string', required: true, description: 'Название перевозчика' },
        Price: { type: 'number', required: true, description: 'Цена предложения' },
        Currency: { type: 'string', required: true, description: 'Валюта' },
        Comment: { type: 'string', required: false, description: 'Комментарий перевозчика' },
        RequestLink: { type: 'string', required: true, description: 'Ссылка на заявку' },
      },
      is_system: true,
      is_active: true,
      created_at: '2026-01-15T10:00:00Z',
      updated_at: '2026-01-15T10:00:00Z',
    },
    {
      id: 'tpl-004',
      slug: 'offer-selected',
      name: 'Предложение выбрано',
      subject: 'Ваше предложение выбрано по заявке #{{.RequestNumber}}',
      body_html: `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .button { display: inline-block; padding: 12px 24px; background: #059669; color: white; text-decoration: none; border-radius: 6px; }
    .details { background: #f0fdf4; padding: 15px; border-radius: 6px; margin: 20px 0; border: 1px solid #bbf7d0; }
    .footer { margin-top: 30px; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Поздравляем!</h1>
    <p>Ваше предложение выбрано по заявке #{{.RequestNumber}}.</p>
    <div class="details">
      <p><strong>Заказчик:</strong> {{.CustomerName}}</p>
      <p><strong>Маршрут:</strong> {{.Route}}</p>
      <p><strong>Стоимость:</strong> {{.Price}} {{.Currency}}</p>
    </div>
    <p>Свяжитесь с заказчиком для уточнения деталей перевозки.</p>
    <p><a href="{{.RequestLink}}" class="button">Открыть заявку</a></p>
    <div class="footer">
      <p>С уважением,<br>Команда Veziizi</p>
    </div>
  </div>
</body>
</html>`,
      body_text: `Поздравляем!

Ваше предложение выбрано по заявке #{{.RequestNumber}}.

Заказчик: {{.CustomerName}}
Маршрут: {{.Route}}
Стоимость: {{.Price}} {{.Currency}}

Свяжитесь с заказчиком для уточнения деталей перевозки.

Открыть заявку: {{.RequestLink}}

С уважением,
Команда Veziizi`,
      category: 'transactional',
      variables_schema: {
        RequestNumber: { type: 'string', required: true, description: 'Номер заявки' },
        CustomerName: { type: 'string', required: true, description: 'Название заказчика' },
        Route: { type: 'string', required: true, description: 'Маршрут перевозки' },
        Price: { type: 'number', required: true, description: 'Цена' },
        Currency: { type: 'string', required: true, description: 'Валюта' },
        RequestLink: { type: 'string', required: true, description: 'Ссылка на заявку' },
      },
      is_system: true,
      is_active: true,
      created_at: '2026-01-15T10:00:00Z',
      updated_at: '2026-01-15T10:00:00Z',
    },
    {
      id: 'tpl-005',
      slug: 'weekly-digest',
      name: 'Еженедельный дайджест',
      subject: 'Veziizi: итоги недели',
      body_html: `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .stat { display: inline-block; padding: 15px 20px; background: #f5f5f5; border-radius: 6px; margin: 5px; text-align: center; }
    .stat-value { font-size: 24px; font-weight: bold; color: #4F46E5; }
    .stat-label { font-size: 12px; color: #666; }
    .footer { margin-top: 30px; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Итоги недели</h1>
    <p>Здравствуйте, {{.Name}}!</p>
    <p>Ваша статистика за прошедшую неделю:</p>
    <div style="margin: 20px 0;">
      <div class="stat">
        <div class="stat-value">{{.NewRequests}}</div>
        <div class="stat-label">Новых заявок</div>
      </div>
      <div class="stat">
        <div class="stat-value">{{.CompletedDeals}}</div>
        <div class="stat-label">Завершено сделок</div>
      </div>
    </div>
    <div class="footer">
      <p>С уважением,<br>Команда Veziizi</p>
      <p><a href="{{.UnsubscribeLink}}">Отписаться от рассылки</a></p>
    </div>
  </div>
</body>
</html>`,
      body_text: `Итоги недели

Здравствуйте, {{.Name}}!

Ваша статистика за прошедшую неделю:
- Новых заявок: {{.NewRequests}}
- Завершено сделок: {{.CompletedDeals}}

С уважением,
Команда Veziizi

Отписаться: {{.UnsubscribeLink}}`,
      category: 'marketing',
      variables_schema: {
        Name: { type: 'string', required: true, description: 'Имя пользователя' },
        NewRequests: { type: 'number', required: true, description: 'Количество новых заявок' },
        CompletedDeals: { type: 'number', required: true, description: 'Количество завершённых сделок' },
        UnsubscribeLink: { type: 'string', required: true, description: 'Ссылка отписки' },
      },
      is_system: false,
      is_active: true,
      created_at: '2026-01-16T14:00:00Z',
      updated_at: '2026-01-16T14:00:00Z',
    },
  ],
}

export const mockEmailTemplates = {
  list(filter?: {
    category?: string
    is_active?: boolean
    is_system?: boolean
    search?: string
    limit?: number
    offset?: number
  }): { templates: EmailTemplate[]; total: number } {
    let templates = [...store.templates]

    if (filter?.category) {
      templates = templates.filter(t => t.category === filter.category)
    }
    if (filter?.is_active !== undefined) {
      templates = templates.filter(t => t.is_active === filter.is_active)
    }
    if (filter?.is_system !== undefined) {
      templates = templates.filter(t => t.is_system === filter.is_system)
    }
    if (filter?.search) {
      const search = filter.search.toLowerCase()
      templates = templates.filter(
        t => t.name.toLowerCase().includes(search) || t.slug.toLowerCase().includes(search)
      )
    }

    const total = templates.length
    const offset = filter?.offset || 0
    const limit = filter?.limit || 50
    templates = templates.slice(offset, offset + limit)

    return { templates, total }
  },

  get(id: string): EmailTemplate | null {
    return store.templates.find(t => t.id === id) || null
  },

  create(data: {
    slug: string
    name: string
    subject: string
    body_html: string
    body_text: string
    category: 'transactional' | 'marketing'
    variables_schema?: Record<string, VariableSpec>
  }): EmailTemplate {
    const now = new Date().toISOString()
    const template: EmailTemplate = {
      id: `tpl-${Date.now()}`,
      slug: data.slug,
      name: data.name,
      subject: data.subject,
      body_html: data.body_html,
      body_text: data.body_text,
      category: data.category,
      variables_schema: data.variables_schema || {},
      is_system: false,
      is_active: true,
      created_at: now,
      updated_at: now,
    }
    store.templates.push(template)
    return template
  },

  update(
    id: string,
    data: {
      name?: string
      subject?: string
      body_html?: string
      body_text?: string
      category?: 'transactional' | 'marketing'
      variables_schema?: Record<string, VariableSpec>
      is_active?: boolean
    }
  ): EmailTemplate | null {
    const index = store.templates.findIndex(t => t.id === id)
    if (index === -1) return null

    const template = store.templates[index]
    if (!template) return null

    const updated: EmailTemplate = {
      ...template,
      ...data,
      updated_at: new Date().toISOString(),
    }
    store.templates[index] = updated
    return updated
  },

  delete(id: string): boolean {
    const index = store.templates.findIndex(t => t.id === id)
    if (index === -1) return false

    const template = store.templates[index]
    if (template?.is_system) return false

    store.templates.splice(index, 1)
    return true
  },

  preview(data: {
    subject: string
    body_html: string
    body_text: string
    variables: Record<string, string>
  }): { subject: string; body_html: string; body_text: string } {
    let subject = data.subject
    let bodyHtml = data.body_html
    let bodyText = data.body_text

    for (const [key, value] of Object.entries(data.variables)) {
      const pattern = new RegExp(`{{\\s*\\.${key}\\s*}}`, 'g')
      subject = subject.replace(pattern, value)
      bodyHtml = bodyHtml.replace(pattern, value)
      bodyText = bodyText.replace(pattern, value)
    }

    return { subject, body_html: bodyHtml, body_text: bodyText }
  },
}
