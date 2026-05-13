# Kubernetes — запуск friend-net

## Архитектура

```
Internet :80
    │
    ▼
Traefik (LoadBalancer)
    ├── /auth/*  ──► auth-service:8080
    │                  публичные: /register /login /refresh /introspect /google/*
    │                  приватные: jwt-auth middleware → /auth/validate → X-Account-ID
    └── /users/* ──► user-service:8081
                       jwt-auth + rate-limit

auth-service ─── Kafka (accounts.events) ──► user-service
     │
     └── Postgres (outbox + accounts) + Redis (sessions)
```

### Неймспейс: `friend-net`

| Ресурс | Тип | Реплики | Порт |
|--------|-----|---------|------|
| traefik | Deployment | 1 | 80 (LB) |
| auth-service | Deployment | 2 | 8080 |
| user-service | Deployment | 2 | 8081 |
| postgres | StatefulSet | 1 | 5432 |
| mongodb | StatefulSet | 1 | 27017 |
| redis | Deployment | 1 | 6379 |
| kafka | StatefulSet | 1 | 9092/9093 |

Persistent storage: postgres 5Gi, mongodb 5Gi, redis 1Gi, kafka 10Gi — итого ~21Gi.

---

## Блокеры — без этого запуск невозможен

| # | Проблема | Файл | Что сделать |
|---|----------|------|-------------|
| 1 | `kubectl`, `minikube` не установлены | — | см. Шаг 0 |
| 2 | Образы `auth-service:latest` / `user-service:latest` не собраны | `*/deployment.yaml` | см. Шаг 2 |
| 3 | Endpoint `/auth/validate` отсутствует в коде | `traefik/middlewares.yaml` | добавить хэндлер или поменять адрес на `/auth/introspect` |
| 4 | Секреты — плейсхолдеры (`your-secret-key...`) | `auth-service/secret.yaml` | заполнить реальными значениями |

---

## Образы — где хранить и как собирать

### Локально (minikube) — самый простой путь

```bash
# Переключить Docker-контекст на внутренний демон minikube
eval $(minikube docker-env)

# Собрать образы — они окажутся прямо внутри кластера
docker build -t auth-service:latest ./auth-service
docker build -t user-service:latest ./user-service

# В deployment.yaml оставить imagePullPolicy: Never (или IfNotPresent)
# чтобы k8s не пытался тянуть образ из интернета
```

> После `eval $(minikube docker-env)` все `docker build/run` идут в minikube.
> Вернуться к системному Docker: `eval $(minikube docker-env --unset)`

### GitHub Container Registry (ghcr.io) — для CI/CD

```bash
# Логин
echo $GITHUB_TOKEN | docker login ghcr.io -u meindokuse --password-stdin

# Собрать и запушить
docker build -t ghcr.io/meindokuse/friend-net/auth-service:latest ./auth-service
docker push ghcr.io/meindokuse/friend-net/auth-service:latest

docker build -t ghcr.io/meindokuse/friend-net/user-service:latest ./user-service
docker push ghcr.io/meindokuse/friend-net/user-service:latest
```

После этого в `k8s/auth-service/deployment.yaml` поменять:
```yaml
image: ghcr.io/meindokuse/friend-net/auth-service:latest
```

Для приватного репозитория нужен imagePullSecret:
```bash
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=meindokuse \
  --docker-password=$GITHUB_TOKEN \
  -n friend-net
```
И добавить в deployment.yaml:
```yaml
spec:
  imagePullSecrets:
    - name: ghcr-secret
```

### Docker Hub — альтернатива

```bash
docker login
docker build -t meindokuse/auth-service:latest ./auth-service
docker push meindokuse/auth-service:latest
```

---

## Шаг 0 — Установить инструменты

```bash
# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -sL https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl && sudo mv kubectl /usr/local/bin/
kubectl version --client

# minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
minikube version
```

---

## Шаг 1 — Запустить кластер

```bash
minikube start --driver=docker --memory=4096 --cpus=4
minikube status
```

---

## Шаг 2 — Собрать образы (локальный путь)

```bash
eval $(minikube docker-env)

docker build -t auth-service:latest ./auth-service
docker build -t user-service:latest ./user-service

# Убедиться что образы видны внутри minikube
docker images | grep -E "auth-service|user-service"
```

Добавить `imagePullPolicy: Never` в оба deployment.yaml:
```yaml
containers:
  - name: auth-service
    image: auth-service:latest
    imagePullPolicy: Never
```

---

## Шаг 3 — Заполнить секреты

Отредактировать `k8s/auth-service/secret.yaml`:

```yaml
stringData:
  POSTGRES_PASSWORD: "сюда-пароль"
  JWT_SECRET: "минимум-32-случайных-символа"
  JWT_REFRESH_SECRET: "другие-32-случайных-символа"
  GOOGLE_CLIENT_ID: "реальный-id из Google Console"
  GOOGLE_CLIENT_SECRET: "реальный-secret из Google Console"
```

Сгенерировать случайные ключи:
```bash
openssl rand -base64 32   # для JWT_SECRET
openssl rand -base64 32   # для JWT_REFRESH_SECRET
```

---

## Шаг 4 — Установить Traefik CRDs

Манифесты используют `IngressRoute` и `Middleware` — это CRD от Traefik.
Без них `kubectl apply` упадёт с `no matches for kind "IngressRoute"`.

```bash
kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v2.11/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml
```

---

## Шаг 5 — Исправить /auth/validate

Traefik jwt-auth middleware вызывает `GET /auth/validate`, которого нет в auth-service.
Есть `POST /auth/introspect` — нужно либо:

**Вариант A** — добавить хэндлер `GET /auth/validate` в auth-service (обёртка над introspect).

**Вариант B** — временно поменять адрес в `k8s/traefik/middlewares.yaml`:
```yaml
forwardAuth:
  address: "http://auth-service.friend-net.svc.cluster.local:8080/auth/introspect"
```
> Но introspect ожидает тело с токеном, а forwardAuth передаёт Authorization header — нужен отдельный хэндлер.

---

## Шаг 6 — Применить манифесты

```bash
# Неймспейс
kubectl apply -f k8s/namespace.yaml

# Traefik
kubectl apply -f k8s/traefik/

# Инфраструктура
kubectl apply -f k8s/infra/postgres/
kubectl apply -f k8s/infra/mongodb/
kubectl apply -f k8s/infra/redis/
kubectl apply -f k8s/infra/kafka/

# Ждём готовности инфры
kubectl wait --for=condition=ready pod -l app=postgres -n friend-net --timeout=120s
kubectl wait --for=condition=ready pod -l app=mongodb -n friend-net --timeout=120s
kubectl wait --for=condition=ready pod -l app=redis -n friend-net --timeout=120s
kubectl wait --for=condition=ready pod -l app=kafka -n friend-net --timeout=120s

# Приложения
kubectl apply -f k8s/auth-service/
kubectl apply -f k8s/user-service/
```

---

## Шаг 7 — Проверка

```bash
# Статус подов
kubectl get pods -n friend-net

# URL для доступа (minikube)
minikube service traefik -n friend-net --url

# Проверить роуты
curl http://<minikube-ip>/auth/healthz

# Логи
kubectl logs -l app=auth-service -n friend-net --tail=50
kubectl logs -l app=user-service -n friend-net --tail=50
kubectl logs -l app=kafka -n friend-net --tail=50
```

---

## Некритичные проблемы (до prod)

| Проблема | Где | Рекомендация |
|----------|-----|--------------|
| `/healthz` нет в user-service, readiness через TCP | `user-service/deployment.yaml` | добавить HTTP healthz хэндлер |
| `image: apache/kafka:latest` — нет пинного тега | `infra/kafka/statefulset.yaml` | заменить на `apache/kafka:3.9.0` |
| `GOOGLE_REDIRECT_URL: http://localhost/...` | `auth-service/configmap.yaml` | менять при деплое на реальный домен |
| `imagePullPolicy` не выставлен явно | оба deployment.yaml | добавить `Never` для minikube или `Always` для registry |
