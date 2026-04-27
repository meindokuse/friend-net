# user-service — документация

## Содержание

1. [Обзор](#обзор)
2. [Структура проекта](#структура-проекта)
3. [Domain](#domain)
   - [Value Objects](#value-objects)
   - [Агрегат User](#агрегат-user)
   - [Доменные ошибки](#доменные-ошибки)
4. [Usecase](#usecase)
   - [Контракт репозитория](#контракт-репозитория)
   - [DTO](#dto)
   - [Методы сервиса](#методы-сервиса)
   - [Ошибки usecase](#ошибки-usecase)

---

## Обзор

`user-service` отвечает за управление профилями пользователей. Он не занимается аутентификацией — это зона `auth-service`. Сюда приходят уже аутентифицированные запросы с известным `user_id`.

Архитектура — **Clean Architecture**:
- `domain` — бизнес-логика, не зависит ни от чего
- `usecase` — оркестрация, зависит только от `domain`
- `adapters` — реализации портов (MongoDB, HTTP, gRPC)

---

## Структура проекта

```
user-service/
├── cmd/                          — точки входа
├── internal/
│   ├── domain/
│   │   ├── shared/vo/            — value objects (Email, Phone, Username)
│   │   └── user/                 — агрегат User, доменные ошибки
│   ├── usecase/user/             — бизнес-сценарии, DTO, контракт репозитория
│   └── adapters/
│       └── mongo/                — реализация репозитория на MongoDB
```

---

## Domain

### Value Objects

Value Objects — неизменяемые объекты-значения. Создаются только через конструктор, который валидирует данные. Если данные невалидны — возвращается ошибка, невалидный объект создать нельзя.

---

#### `vo.Username`

Путь: `internal/domain/shared/vo/username.go`

Логин пользователя.

| Правило | Значение |
|---|---|
| Минимальная длина | 3 символа |
| Максимальная длина | 32 символа |
| Допустимые символы | латинские буквы, цифры, `_` |
| Нормализация | приводится к нижнему регистру |

```go
username, err := vo.NewUsername("Alice_99")
// username.String() == "alice_99"
```

Ошибки: `ErrUsernameEmpty`, `ErrUsernameTooShort`, `ErrUsernameTooLong`, `ErrUsernameInvalid`

---

#### `vo.Email`

Путь: `internal/domain/shared/vo/email.go`

Email пользователя.

| Правило | Значение |
|---|---|
| Максимальная длина | 254 символа |
| Формат | RFC 5322 |
| Нормализация | trim + нижний регистр |

```go
email, err := vo.NewEmail("  User@Example.COM  ")
// email.String() == "user@example.com"
```

Ошибки: `ErrEmailEmpty`, `ErrEmailInvalid`, `ErrEmailTooLong`

---

#### `vo.Phone`

Путь: `internal/domain/shared/vo/phone.go`

Номер телефона в формате E.164.

| Правило | Значение |
|---|---|
| Формат | E.164: `+` и до 15 цифр |
| Нормализация | убираются пробелы, `-`, `(`, `)` |

```go
phone, err := vo.NewPhone("+7 (999) 123-45-67")
// phone.String() == "+79991234567"
```

Ошибки: `ErrPhoneEmpty`, `ErrPhoneInvalid`

---

### Агрегат User

Путь: `internal/domain/user/entity.go`

`User` — корень агрегата. Все поля приватные, изменения только через бизнес-методы. Нельзя создать невалидного пользователя.

#### Поля

| Поле | Тип | Описание |
|---|---|---|
| `id` | `uuid.UUID` | Уникальный идентификатор |
| `username` | `vo.Username` | Логин, уникален в системе |
| `email` | `*vo.Email` | Email, опционален |
| `phone` | `*vo.Phone` | Телефон, опционален |
| `profile` | `Profile` | Отображаемые данные профиля |
| `settings` | `Settings` | Настройки приватности и локали |
| `verification` | `Verification` | Статус подтверждения email/phone |
| `isActive` | `bool` | Активен ли аккаунт |
| `createdAt` | `time.Time` | Дата создания |
| `updatedAt` | `time.Time` | Дата последнего изменения |
| `lastSeenAt` | `*time.Time` | Последний визит, опционален |
| `deletedAt` | `*time.Time` | Дата soft-delete, nil если активен |
| `version` | `int` | Версия для optimistic locking |

#### Вложенные типы

**`Profile`**
```
DisplayName  string   — отображаемое имя, 1–64 символа, обязательно
Bio          *string  — описание профиля, до 500 символов, опционально
AvatarURL    *string  — ссылка на аватар, опционально
```

**`Settings`**
```
Privacy   PrivacySettings — настройки приватности
Language  string          — язык интерфейса (default: "en")
Timezone  string          — часовой пояс (default: "UTC")
```

**`PrivacySettings`**
```
WhoCanMessage      PrivacyLevel — кто может писать сообщения
WhoCanSeeLastSeen  PrivacyLevel — кто видит время последнего визита
WhoCanSeeProfile   PrivacyLevel — кто видит профиль
```

**`PrivacyLevel`** — строковый enum:
```
"everyone"  — все пользователи
"friends"   — только друзья
"nobody"    — никто
```

**`Verification`**
```
EmailVerified  bool — email подтверждён
PhoneVerified  bool — телефон подтверждён
```

#### Инварианты при создании

- Хотя бы одно из `email` или `phone` обязательно
- `DisplayName` обязателен и не длиннее 64 символов
- Новый пользователь создаётся с `isActive = true`, `version = 1`
- Настройки по умолчанию: все privacy = `"everyone"`, язык `"en"`, таймзона `"UTC"`

#### Фабрики

```go
// Создание нового пользователя — применяет все инварианты
user, err := domainuser.NewUser(id, username, emailVO, phoneVO, displayName)

// Восстановление из БД — не валидирует, данные считаются валидными
user := domainuser.Reconstruct(id, username, email, phone, profile, settings,
    verification, isActive, createdAt, updatedAt, lastSeenAt, deletedAt, version)
```

#### Бизнес-методы

**`UpdateProfile(displayName string, bio, avatarURL *string) error`**
Обновляет профиль. Валидирует длину displayName (макс 64) и bio (макс 500). Вызывает `touch()` → `updatedAt` обновляется, `version++`.

**`UpdateSettings(s Settings) error`**
Заменяет настройки целиком. Валидирует все три `PrivacyLevel`. Вызывает `touch()`.

**`ChangeEmail(email vo.Email)`**
Меняет email. После смены `EmailVerified` сбрасывается в `false`. Вызывает `touch()`.

**`ChangePhone(phone vo.Phone)`**
Меняет телефон. После смены `PhoneVerified` сбрасывается в `false`. Вызывает `touch()`.

**`VerifyEmail()`**
Помечает email подтверждённым. Вызывает `touch()`.

**`VerifyPhone()`**
Помечает телефон подтверждённым. Вызывает `touch()`.

**`UpdateLastSeen()`**
Обновляет `lastSeenAt`. **Не вызывает `touch()`** — не меняет `version`. Это намеренно: last_seen обновляется очень часто и не является значимым изменением для optimistic lock.

**`SoftDelete() error`**
Помечает пользователя удалённым: устанавливает `deletedAt = now`, `isActive = false`. Вызывает `touch()`. Повторный вызов вернёт `ErrAlreadyDeleted`. Hard delete запрещён.

#### Optimistic Locking

Каждое значимое изменение увеличивает `version` через внутренний метод `touch()`. При сохранении в БД репозиторий проверяет что версия в БД равна `version - 1`. Если нет — кто-то успел изменить запись раньше, возвращается `ErrVersionConflict`.

Клиент обязан передавать текущую версию в мутирующих запросах.

---

### Доменные ошибки

Путь: `internal/domain/user/errrors.go`

#### Ошибки валидации

| Ошибка | Когда |
|---|---|
| `ErrEmailOrPhoneRequired` | При создании не передан ни email, ни phone |
| `ErrDisplayNameRequired` | DisplayName пустой |
| `ErrDisplayNameTooLong` | DisplayName длиннее 64 символов |
| `ErrBioTooLong` | Bio длиннее 500 символов |
| `ErrInvalidPrivacyLevel` | Передан неизвестный PrivacyLevel |

#### Ошибки жизненного цикла

| Ошибка | Когда |
|---|---|
| `ErrAlreadyDeleted` | Попытка удалить уже удалённого пользователя |

#### Ошибки репозитория

Определены в домене, бросаются адаптером БД.

| Ошибка | Когда |
|---|---|
| `ErrUserNotFound` | Пользователь не найден |
| `ErrUsernameAlreadyTaken` | Username уже занят |
| `ErrEmailAlreadyTaken` | Email уже занят |
| `ErrPhoneAlreadyTaken` | Телефон уже занят |
| `ErrVersionConflict` | Optimistic lock: версия устарела |

---

## Usecase

### Контракт репозитория

Путь: `internal/usecase/user/repository.go`

Интерфейс определён на стороне usecase (consumer), реализуется адаптером. Usecase не знает про MongoDB.

```go
type UserRepository interface {
    Create(ctx context.Context, u *domainuser.User) error
    Update(ctx context.Context, u *domainuser.User) error

    GetByID(ctx context.Context, id uuid.UUID) (*domainuser.User, error)
    GetByUsername(ctx context.Context, username vo.Username) (*domainuser.User, error)
    GetByEmail(ctx context.Context, email vo.Email) (*domainuser.User, error)
    GetByPhone(ctx context.Context, phone vo.Phone) (*domainuser.User, error)
    GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domainuser.User, error)

    Search(ctx context.Context, query string, limit, offset int) ([]*domainuser.User, error)
    UpdateLastSeen(ctx context.Context, id uuid.UUID) error
}
```

Все методы чтения не возвращают soft-deleted пользователей.

---

### DTO

Путь: `internal/usecase/user/dto.go`

#### Input DTO

**`CreateUserInput`**
```
ID           *uuid.UUID  — если nil, usecase генерирует сам
Username     string
Email        *string     — опционально
Phone        *string     — опционально
DisplayName  string
```

**`UpdateProfileInput`**
```
UserID       uuid.UUID
DisplayName  string
Bio          *string
AvatarURL    *string
Version      int         — текущая версия для optimistic lock
```

**`UpdateSettingsInput`**
```
UserID              uuid.UUID
WhoCanMessage       string
WhoCanSeeLastSeen   string
WhoCanSeeProfile    string
Language            string
Timezone            string
Version             int
```

**`ChangeEmailInput`**
```
UserID   uuid.UUID
Email    string
Version  int
```

**`ChangePhoneInput`**
```
UserID   uuid.UUID
Phone    string
Version  int
```

**`DeleteUserInput`**
```
UserID   uuid.UUID
Version  int
```

**`SearchUsersInput`**
```
Query   string
Limit   int     — default 20, max 100
Offset  int
```

#### Output DTO

**`UserOutput`** — полный профиль, для владельца аккаунта (`/users/me`)
```
ID             uuid.UUID
Username       string
Email          *string
Phone          *string
DisplayName    string
Bio            *string
AvatarURL      *string
EmailVerified  bool
PhoneVerified  bool
Privacy        PrivacyOutput
Language       string
Timezone       string
IsActive       bool
CreatedAt      time.Time
UpdatedAt      time.Time
LastSeenAt     *time.Time
Version        int
```

**`PublicUserOutput`** — публичный профиль, для просмотра чужого аккаунта. Не содержит email, phone, настройки.
```
ID           uuid.UUID
Username     string
DisplayName  string
Bio          *string
AvatarURL    *string
LastSeenAt   *time.Time
```

**`PrivacyOutput`**
```
WhoCanMessage      string
WhoCanSeeLastSeen  string
WhoCanSeeProfile   string
```

---

### Методы сервиса

Сервис: `internal/usecase/user/usecase.go` → `type Service struct`

---

#### `CreateUser(ctx, CreateUserInput) (*UserOutput, error)`

Создаёт нового пользователя.

1. Валидирует и создаёт VO: `Username`, `Email` (если передан), `Phone` (если передан)
2. Если `ID` не передан — генерирует `uuid.New()`
3. Создаёт доменную сущность через `domainuser.NewUser()` — проверяются инварианты
4. Сохраняет через `repo.Create()`

Возвращаемые ошибки: `ErrInvalidInput`, `ErrUsernameAlreadyTaken`, `ErrEmailAlreadyTaken`, `ErrPhoneAlreadyTaken`

---

#### `UpdateProfile(ctx, UpdateProfileInput) (*UserOutput, error)`

Обновляет профиль пользователя.

1. Загружает пользователя по `UserID`
2. Fast-fail проверка версии до обращения к БД
3. Вызывает `u.UpdateProfile()` — валидация на уровне домена
4. Сохраняет через `repo.Update()`

Возвращаемые ошибки: `ErrUserNotFound`, `ErrVersionConflict`, `ErrInvalidInput`

---

#### `UpdateSettings(ctx, UpdateSettingsInput) (*UserOutput, error)`

Обновляет настройки приватности и локали.

1. Загружает пользователя по `UserID`
2. Fast-fail проверка версии
3. Собирает `domainuser.Settings` из input
4. Вызывает `u.UpdateSettings()` — валидирует PrivacyLevel
5. Сохраняет через `repo.Update()`

Возвращаемые ошибки: `ErrUserNotFound`, `ErrVersionConflict`, `ErrInvalidInput`

---

#### `ChangeEmail(ctx, ChangeEmailInput) (*UserOutput, error)`

Меняет email пользователя.

1. Валидирует новый email через `vo.NewEmail()`
2. Загружает пользователя по `UserID`
3. Fast-fail проверка версии
4. Проверяет уникальность: ищет пользователя с таким email, если найден и это не тот же пользователь — ошибка
5. Вызывает `u.ChangeEmail()` — сбрасывает `EmailVerified = false`
6. Сохраняет через `repo.Update()`

Возвращаемые ошибки: `ErrInvalidInput`, `ErrUserNotFound`, `ErrVersionConflict`, `ErrEmailAlreadyTaken`

---

#### `ChangePhone(ctx, ChangePhoneInput) (*UserOutput, error)`

Меняет телефон пользователя. Логика аналогична `ChangeEmail`.

1. Валидирует новый телефон через `vo.NewPhone()`
2. Загружает пользователя по `UserID`
3. Fast-fail проверка версии
4. Проверяет уникальность телефона
5. Вызывает `u.ChangePhone()` — сбрасывает `PhoneVerified = false`
6. Сохраняет через `repo.Update()`

Возвращаемые ошибки: `ErrInvalidInput`, `ErrUserNotFound`, `ErrVersionConflict`, `ErrPhoneAlreadyTaken`

---

#### `DeleteUser(ctx, DeleteUserInput) error`

Мягко удаляет пользователя (soft delete).

1. Загружает пользователя по `UserID`
2. Fast-fail проверка версии
3. Вызывает `u.SoftDelete()` — устанавливает `deletedAt`, `isActive = false`
4. Сохраняет через `repo.Update()`

После удаления пользователь не возвращается ни одним методом чтения репозитория.

Возвращаемые ошибки: `ErrUserNotFound`, `ErrVersionConflict`, `ErrAlreadyDeleted`

---

#### `GetUserByID(ctx, uuid.UUID) (*UserOutput, error)`

Возвращает полный профиль пользователя. Используется для `/users/me`.

Возвращаемые ошибки: `ErrUserNotFound`

---

#### `GetPublicUserByID(ctx, uuid.UUID) (*PublicUserOutput, error)`

Возвращает публичный профиль по ID. Используется при просмотре чужого профиля.

Возвращаемые ошибки: `ErrUserNotFound`

---

#### `GetPublicUserByUsername(ctx, string) (*PublicUserOutput, error)`

Возвращает публичный профиль по username. Валидирует username через `vo.NewUsername()`.

Возвращаемые ошибки: `ErrInvalidInput`, `ErrUserNotFound`

---

#### `GetUsersByIDs(ctx, []uuid.UUID) ([]*PublicUserOutput, error)`

Batch-запрос публичных профилей по списку ID. Предназначен для inter-service вызовов (например, chat-service запрашивает данные участников).

Лимит: максимум **500** ID за один запрос.

Возвращаемые ошибки: `ErrInvalidInput` (если превышен лимит)

---

#### `SearchUsers(ctx, SearchUsersInput) ([]*PublicUserOutput, error)`

Поиск пользователей по подстроке в `username` или `displayName`. Регистронезависимый.

- Пустой запрос — ошибка
- `limit` по умолчанию 20, максимум 100
- `offset` для пагинации, минимум 0

Возвращаемые ошибки: `ErrInvalidInput`

---

#### `UpdateLastSeen(ctx, uuid.UUID) error`

Обновляет время последнего визита пользователя. Быстрая операция — не читает весь документ, делает только `$set` по одному полю. Не меняет `version`.

Возвращаемые ошибки: `ErrUserNotFound`

---

### Ошибки usecase

Путь: `internal/usecase/user/errors.go`

| Ошибка | Когда |
|---|---|
| `ErrInvalidInput` | Невалидные входные данные (формат, длина и т.п.) |

Доменные ошибки (`ErrUserNotFound`, `ErrVersionConflict` и др.) пробрасываются наверх как есть — handler маппит их в HTTP-коды.
