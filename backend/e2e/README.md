# E2E Tests

End-to-end тесты для API veziizi.

## Структура

```
backend/e2e/
├── setup/          # Инфраструктура тестов
│   ├── suite.go    # Тестовый сервер, lifecycle
│   ├── config.go   # Тестовая конфигурация
│   └── database.go # Миграции, очистка БД
│
├── client/         # Типизированный HTTP клиент
│   ├── client.go   # API методы
│   └── types.go    # Request/Response типы
│
├── fixtures/       # Builders для тестовых данных
│   ├── organization.go  # Создание организаций
│   ├── freight.go       # Заявки и офферы
│   └── scenarios.go     # Комплексные сценарии
│
├── helpers/        # Утилиты
│   ├── assert.go   # Кастомные ассерты
│   ├── wait.go     # Ожидание условий
│   └── random.go   # Генерация данных
│
└── tests/          # Тесты
    ├── main_test.go              # TestMain
    ├── auth_test.go              # AUTH-* тесты
    ├── organizations_test.go     # ORG-* тесты
    └── freight_requests_test.go  # FR-* тесты
```

## Запуск

```bash
# Полный запуск (создаёт тестовую БД, запускает тесты)
make test-e2e

# Параллельный запуск (быстрее, но требует изоляции)
make test-e2e-parallel

# Только подготовка БД
make test-e2e-setup

# Запуск конкретного теста
TEST_DATABASE_URL=postgres://veziizi:veziizi@localhost:5432/veziizi_test?sslmode=disable \
  go test -v -run TestLogin ./backend/e2e/tests/...
```

## Написание тестов

### Базовый тест

```go
func TestMyEndpoint(t *testing.T) {
    t.Parallel()
    c := getClient(t)

    // Создаём тестовые данные
    org := fixtures.NewOrganization(t, c).Create()

    // Выполняем тест
    resp, err := org.Client.SomeMethod()
    helpers.AssertNil(t, err)
    helpers.AssertStatusOK(t, resp.StatusCode, resp.RawBody)
}
```

### Table-Driven Tests

```go
func TestEndpoint(t *testing.T) {
    t.Parallel()
    c := getClient(t)

    tests := []struct {
        id         string  // ID из E2E_TESTS.md
        name       string
        wantStatus int
        wantErr    string
    }{
        {id: "XXX-001", name: "happy path", wantStatus: 200},
        {id: "XXX-002", name: "error case", wantStatus: 400, wantErr: "error"},
    }

    for _, tt := range tests {
        t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
            t.Parallel() // Для независимых тестов

            resp, err := c.SomeMethod()
            helpers.AssertNil(t, err)
            helpers.AssertStatus(t, resp.StatusCode, tt.wantStatus, resp.RawBody)

            if tt.wantErr != "" {
                helpers.AssertErrorContains(t, resp.RawBody, tt.wantErr)
            }
        })
    }
}
```

### Fixtures

```go
// Простая организация (pending)
org := fixtures.NewOrganization(t, c).Create()

// Кастомизация
org := fixtures.NewOrganization(t, c).
    WithCountry("KZ").
    WithOwnerEmail("custom@test.local").
    Create()

// Активная организация (approved)
ctx := fixtures.NewTestContext(t, suite.BaseURL)
// ctx.Customer - approved заказчик
// ctx.Carrier - approved перевозчик
// ctx.AdminClient - авторизованный админ

// Создание заявки
fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).
    WithWeight(5000).
    WithPrice(100000).
    Create()

// Создание оффера
offer := fixtures.NewOffer(t, ctx.Carrier.Client, fr.ID).
    WithPrice(90000).
    Create()

// Полный сценарий: заявка -> оффер -> заказ
fr, offer, orderID := ctx.CreateConfirmedOrder()
```

### Helpers

```go
// Ассерты
helpers.AssertStatus(t, resp.StatusCode, 200, resp.RawBody)
helpers.AssertStatusOK(t, resp.StatusCode, resp.RawBody)
helpers.AssertStatusCreated(t, resp.StatusCode, resp.RawBody)
helpers.AssertStatusNoContent(t, resp.StatusCode, resp.RawBody)
helpers.AssertStatusBadRequest(t, resp.StatusCode, resp.RawBody)
helpers.AssertStatusUnauthorized(t, resp.StatusCode, resp.RawBody)
helpers.AssertStatusForbidden(t, resp.StatusCode, resp.RawBody)
helpers.AssertStatusNotFound(t, resp.StatusCode, resp.RawBody)
helpers.AssertStatusConflict(t, resp.StatusCode, resp.RawBody)
helpers.AssertErrorContains(t, resp.RawBody, "some error")

// Ожидание
helpers.Wait(t, func() bool { return condition }, "message")
helpers.Eventually(t, func() bool { return condition }, 5*time.Second, "message")

// Генерация
email := helpers.RandomEmail()
phone := helpers.RandomPhone()
inn := helpers.RandomINN()
```

## Изоляция тестов

### Для параллельных тестов

Используйте уникальные данные через fixtures:
```go
// Каждый вызов создаёт уникальный email/ИНН
org1 := fixtures.NewOrganization(t, c).Create()
org2 := fixtures.NewOrganization(t, c).Create()
```

### Для теста, требующего чистой БД

```go
func TestNeedsCleanDB(t *testing.T) {
    suite := setup.NewSuite(t) // Изолированный suite
    // ...
}
```

## Добавление новых тестов

1. Найдите тест-кейс в `E2E_TESTS.md`
2. Добавьте в соответствующий файл `*_test.go`
3. Используйте ID из документа (AUTH-001, ORG-001, FR-001...)
4. Следуйте паттерну table-driven tests

## Соответствие E2E_TESTS.md

| Группа | Файл | Статус |
|--------|------|--------|
| Auth (AUTH-*) | auth_test.go | Частично |
| Organizations (ORG-*) | organizations_test.go | Частично |
| Freight Requests (FR-*) | freight_requests_test.go | Частично |
| Orders (ORD-*) | orders_test.go | TODO |
| Geo (GEO-*) | geo_test.go | TODO |
| Admin (ADM-*) | admin_test.go | TODO |
| Notifications (NOT-*) | notifications_test.go | TODO |
| Support (SUP-*) | support_test.go | TODO |
