/**
 * Customer Flow Scenario (Оптимизированная версия)
 * Сценарий обучения: создание заявки на перевозку
 *
 * Сокращено с 47 до ~23 шагов для лучшего UX
 */

import type { TutorialStep } from './types'

export const steps: TutorialStep[] = [
  // ========================================
  // НАВИГАЦИЯ (2 шага)
  // ========================================

  // Мобильная навигация
  {
    id: 'open_menu',
    title: 'Откройте меню и выберите "Заявки"',
    description: 'Нажмите на кнопку меню в левом верхнем углу, затем выберите "Заявки".',
    target: 'mobile-menu-btn',
    tooltipPosition: 'right',
    completionType: 'action',
    completionAction: 'menu:opened',
    platform: 'mobile',
  },
  {
    id: 'select_requests',
    title: 'Выберите "Заявки"',
    description: 'Нажмите "Заявки" для перехода к списку заявок.',
    target: 'mobile-nav-requests',
    hideTooltip: true, // Только подсветка, без tooltip
    completionType: 'navigate',
    completionAction: '/',
    platform: 'mobile',
  },

  // Desktop навигация
  {
    id: 'desktop_nav_intro',
    title: 'Раздел "Заявки"',
    description: 'Нажмите на "Заявки" для перехода к списку заявок.',
    target: 'nav-requests',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'nav:requestsClicked',
    platform: 'desktop',
  },

  // Создание заявки
  {
    id: 'customer_start',
    title: 'Создание заявки',
    description: 'Нажмите "Новая заявка" чтобы создать заявку на перевозку груза.',
    target: 'create-request-btn',
    tooltipPosition: 'bottom',
    completionType: 'navigate',
    completionAction: '/freight-requests/new',
  },

  // ========================================
  // МАРШРУТ (5 шагов)
  // ========================================

  {
    id: 'route_loading_point',
    title: 'Точка погрузки',
    description: 'Выберите страну и город откуда нужно забрать груз.',
    hint: 'Синяя полоска слева означает погрузку',
    target: 'route-point-0',
    tooltipPosition: 'top',
    completionType: 'action',
    completionAction: 'route:citySelected',
  },
  {
    id: 'route_loading_date',
    title: 'Дата погрузки',
    description: 'Укажите дату погрузки. Можно указать диапазон дат если точная дата неизвестна.',
    target: 'route-date-fields',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'route:dateSet',
  },
  {
    id: 'route_optional_fields',
    title: 'Дополнительные поля',
    description: 'Можно добавить время, контакт на погрузке и примечание. Это опционально — пропустите если не нужно.',
    hint: 'Нажмите "+ Время", "+ Контакт" или "+ Примечание" чтобы добавить поле',
    target: 'route-optional-buttons',
    tooltipPosition: 'top',
    completionType: 'manual',
    skippable: true,
  },
  {
    id: 'route_unloading_point',
    title: 'Точка разгрузки',
    description: 'Выберите страну и город куда нужно доставить груз.',
    hint: 'Зелёная полоска слева означает разгрузку',
    target: 'route-point-1',
    tooltipPosition: 'top',
    completionType: 'action',
    completionAction: 'route:citySelected',
  },
  {
    id: 'route_unloading_date',
    title: 'Дата разгрузки',
    description: 'Укажите дату разгрузки. Можно указать диапазон дат если точная дата неизвестна.',
    target: 'route-date-fields-1',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'route:dateSet',
  },
  {
    id: 'route_add_point_demo',
    title: 'Кнопка "Добавить точку"',
    description: 'Эта кнопка позволяет добавить промежуточные точки маршрута — для транзитной погрузки или разгрузки.',
    hint: 'Нажмите на кнопку чтобы добавить промежуточную точку, или пропустите этот шаг',
    target: 'route-add-point-btn',
    tooltipPosition: 'top',
    completionType: 'manual',
    skippable: true,
  },
  {
    id: 'route_continue',
    title: 'Маршрут готов!',
    description: 'Нажмите "Далее" для перехода к описанию груза.',
    target: 'submit-btn',
    tooltipPosition: 'top',
    completionType: 'action',
    completionAction: 'wizard:next',
  },

  // ========================================
  // ГРУЗ (5 шагов)
  // ========================================

  {
    id: 'cargo_description',
    title: 'Описание груза',
    description: 'Опишите что перевозите: мебель, стройматериалы, продукты и т.д.',
    hint: 'Чем подробнее описание, тем точнее будут предложения перевозчиков',
    target: 'cargo-description',
    tooltipPosition: 'bottom',
    completionType: 'manual',
  },
  {
    id: 'cargo_weight_volume',
    title: 'Вес и объём',
    description: 'Укажите вес груза (обязательно). Объём и габариты — опционально.',
    hint: 'Если точный вес неизвестен — укажите примерный',
    target: 'cargo-weight',
    tooltipPosition: 'bottom',
    completionType: 'manual',
  },
  {
    id: 'cargo_quantity_adr',
    title: 'Количество и класс опасности',
    description: 'Укажите количество мест. Для опасных грузов выберите класс ADR.',
    target: 'cargo-quantity',
    tooltipPosition: 'bottom',
    completionType: 'manual',
  },
  {
    id: 'cargo_optional_fields',
    title: 'Дополнительные поля',
    description: 'На этой странице также можно указать объём груза, габариты (длина, ширина, высота) и класс опасности ADR. Эти поля опциональны.',
    completionType: 'manual',
    skippable: true,
  },
  {
    id: 'cargo_continue',
    title: 'Груз описан!',
    description: 'Нажмите "Далее" для перехода к требованиям к транспорту.',
    target: 'submit-btn',
    tooltipPosition: 'top',
    completionType: 'action',
    completionAction: 'wizard:next',
  },

  // ========================================
  // ТРАНСПОРТ (5 шагов)
  // ========================================

  {
    id: 'vehicle_type',
    title: 'Тип транспорта',
    description: 'Выберите тип транспорта: фургон, платформа, цистерна и т.д.',
    hint: 'Этот шаг можно пропустить — выберите сразу тип кузова и тип транспорта определится автоматически',
    target: 'vehicle-type',
    tooltipPosition: 'bottom',
    completionType: 'manual',
    skippable: true,
  },
  {
    id: 'vehicle_subtype',
    title: 'Тип кузова',
    description: 'Выберите тип кузова: тент, рефрижератор, изотерма, цельнометалл и т.д.',
    hint: 'Это обязательное поле. При выборе тип транспорта определится автоматически',
    target: 'vehicle-subtype',
    tooltipPosition: 'bottom',
    completionType: 'manual',
  },
  {
    id: 'vehicle_loading_capacity',
    title: 'Погрузка и грузоподъёмность',
    description: 'Укажите способ погрузки (задняя, боковая, верхняя) и минимальную грузоподъёмность.',
    target: 'vehicle-loading',
    tooltipPosition: 'bottom',
    completionType: 'manual',
  },
  {
    id: 'vehicle_optional',
    title: 'Дополнительные требования',
    description: 'Можно указать объём и размеры кузова. Для изотермы и рефрижератора — температурный режим и термописец.',
    hint: 'Все поля опциональны — заполните если важно',
    completionType: 'manual',
    skippable: true,
  },
  {
    id: 'vehicle_continue',
    title: 'Транспорт выбран!',
    description: 'Нажмите "Далее" для перехода к условиям оплаты.',
    target: 'submit-btn',
    tooltipPosition: 'top',
    completionType: 'action',
    completionAction: 'wizard:next',
  },

  // ========================================
  // ОПЛАТА (7 шагов)
  // ========================================

  {
    id: 'payment_price',
    title: 'Стоимость перевозки',
    description: 'Укажите желаемую сумму за перевозку.',
    target: 'payment-price',
    tooltipPosition: 'top',
    completionType: 'manual',
  },
  {
    id: 'payment_currency',
    title: 'Валюта',
    description: 'Выберите валюту оплаты: рубли, доллары, евро и т.д.',
    target: 'payment-currency',
    tooltipPosition: 'top',
    completionType: 'manual',
  },
  {
    id: 'payment_vat',
    title: 'НДС',
    description: 'Укажите включён ли НДС в стоимость: с НДС, без НДС, или НДС сверху.',
    target: 'payment-vat',
    tooltipPosition: 'top',
    completionType: 'manual',
  },
  {
    id: 'payment_method',
    title: 'Способ оплаты',
    description: 'Выберите как будет производиться оплата: банковский перевод, наличные или карта.',
    target: 'payment-method',
    tooltipPosition: 'top',
    completionType: 'manual',
  },
  {
    id: 'payment_terms',
    title: 'Условия оплаты',
    description: 'Выберите когда производится оплата: при погрузке, на разгрузке или с отсрочкой.',
    target: 'payment-terms',
    tooltipPosition: 'top',
    completionType: 'manual',
  },
  {
    id: 'payment_no_price',
    title: 'Без указания цены',
    description: 'Если не знаете цену — отметьте галочку. Перевозчики сами предложат свою стоимость.',
    hint: 'Это опционально — пропустите если цена уже указана',
    target: 'payment-no-price',
    tooltipPosition: 'bottom',
    completionType: 'manual',
    skippable: true,
  },
  {
    id: 'payment_continue',
    title: 'Оплата готова!',
    description: 'Нажмите "Далее" для проверки данных перед публикацией.',
    target: 'submit-btn',
    tooltipPosition: 'top',
    completionType: 'action',
    completionAction: 'wizard:next',
  },

  // ========================================
  // ПОДТВЕРЖДЕНИЕ (2 шага)
  // ========================================

  {
    id: 'summary_overview',
    title: 'Проверка данных',
    description: 'Здесь подведён итог вашей заявки. Проверьте маршрут, груз, транспорт и условия оплаты.',
    hint: 'Все данные можно отредактировать, вернувшись на предыдущие шаги',
    target: 'confirmation-summary',
    tooltipPosition: 'top',
    completionType: 'manual',
  },
  {
    id: 'customer_submit_request',
    title: 'Опубликовать заявку',
    description: 'Проверьте данные и нажмите "Опубликовать". Кнопка "Назад" позволяет вернуться к редактированию.',
    target: 'wizard-buttons',
    tooltipPosition: 'top',
    completionType: 'action',
    completionAction: 'freightRequest:created',
  },

  // ========================================
  // ЗАВЕРШЕНИЕ (2 шага)
  // ========================================

  {
    id: 'request_published',
    title: 'Заявка опубликована!',
    description: 'Ваша заявка появилась в списке. Теперь перевозчики могут её увидеть и сделать предложения.',
    hint: 'Когда перевозчик заинтересуется, вы получите уведомление',
    target: 'freight-request-card',
    tooltipPosition: 'bottom',
    route: '/',
    completionType: 'manual',
  },
  {
    id: 'customer_complete',
    title: 'Что дальше?',
    description: 'Перевозчики видят вашу заявку и могут сделать предложения с ценой и условиями. Вы сможете сравнить и выбрать лучшее предложение.',
    completionType: 'manual',
    showOffersTrainingButton: true,
  },
]

export default { steps }
