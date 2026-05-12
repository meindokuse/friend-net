# friend-net API Tester

React-приложение для ручного тестирования всех HTTP-эндпоинтов `auth-service` и `user-service`.

## Требования

- Node.js 18+
- Запущенный стек через Traefik V2 на `http://localhost:80`

## Запуск

```bash
# Убедись, что стек уже поднят (из корня монорепо)
make up

# Перейди в директорию тестового UI
cd test-ui

# Установи зависимости (один раз)
npm install

# Запусти dev-сервер
npm run dev
```

Открой **http://localhost:5173** в браузере.

## Как работает прокси

Vite проксирует запросы к Traefik — ты работаешь с реальными сервисами:

```
Browser :5173  →  Vite proxy  →  Traefik :80  →  auth-service :8080
                                              →  user-service  :8081
```

Никаких CORS-проблем — все запросы идут через один origin.

---

## Сценарий работы

### 1. Регистрация + вход

1. Открой вкладку **Auth**
2. Раскрой `POST /auth/register` → заполни email, пароль (мин. 8 символов), display name (мин. 4 символа) → нажми **Зарегистрироваться**
3. Раскрой `POST /auth/login` → введи те же email и пароль → нажми **Войти**
4. Токены сохранятся автоматически в `localStorage` и отобразятся в сайдбаре

После входа в шапке появится зелёный индикатор **Авторизован**.

### 2. Работа с приватными роутами

Traefik сам проверяет Bearer-токен через `forwardAuth → /auth/validate` и добавляет `X-Account-Id` в запрос к user-service. Тебе ничего дополнительно передавать не нужно — просто нажимай кнопки.

Роуты с меткой **🔑 auth** требуют активного токена.

### 3. User-сервис

1. Переключись на вкладку **Users**
2. `GET /users/me` — получить свой профиль
3. `PATCH /users/me/profile` — обновить имя / bio / аватар
4. `PATCH /users/me/settings` — настройки приватности, язык, таймзона
5. `GET /users/search?q=...` — публичный поиск по username

> **Version** — optimistic locking. Перед обновлением сначала получи профиль через `GET /users/me`, скопируй поле `version` из ответа и вставь его в форму.

### 4. Управление сессиями

В секции **Auth**:
- `GET /auth/sessions` — список всех активных сессий (устройств)
- `DELETE /auth/sessions/:session_id` — отозвать конкретную сессию
- `POST /auth/logout-all` — выйти со всех устройств сразу

### 5. Обновление токенов

`POST /auth/refresh` использует текущий refresh-токен из стора. После успешного обновления access и refresh токены автоматически обновятся в сайдбаре.

---

## Роуты сервисов

### auth-service

| Метод | Путь | Доступ |
|-------|------|--------|
| POST | `/auth/register` | Публичный |
| POST | `/auth/login` | Публичный |
| POST | `/auth/refresh` | Публичный |
| POST | `/auth/introspect` | Публичный |
| GET | `/auth/google` | Публичный (OAuth redirect) |
| POST | `/auth/logout` | jwt-auth |
| POST | `/auth/logout-all` | jwt-auth |
| GET | `/auth/sessions` | jwt-auth |
| DELETE | `/auth/sessions/:id` | jwt-auth |
| GET | `/auth/linked` | jwt-auth |
| DELETE | `/auth/linked/:provider` | jwt-auth |
| GET | `/auth/link/google` | jwt-auth |

### user-service

| Метод | Путь | Доступ |
|-------|------|--------|
| POST | `/users/` | Публичный |
| GET | `/users/search` | Публичный |
| GET | `/users/username/{username}` | Публичный |
| GET | `/users/{id}` | jwt-auth |
| POST | `/users/batch` | jwt-auth |
| GET | `/users/me` | jwt-auth |
| PATCH | `/users/me/profile` | jwt-auth |
| PATCH | `/users/me/settings` | jwt-auth |
| PATCH | `/users/me/email` | jwt-auth |
| PATCH | `/users/me/phone` | jwt-auth |
| POST | `/users/me/last-seen` | jwt-auth |
| GET | `/users/me/list` | jwt-auth |
| DELETE | `/users/me` | jwt-auth |

**jwt-auth** — Traefik вызывает `/auth/validate`, проверяет Bearer-токен и прокидывает `X-Account-Id` в upstream-сервис.

---

## Сброс состояния

Нажми **Очистить токены** в сайдбаре — удалит токены из `localStorage` и UI вернётся в неавторизованное состояние.

## Сборка

```bash
npm run build
# Статика окажется в dist/
```
