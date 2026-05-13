● Теперь у меня полная картина. Составлю подробный гайд.

  ---
  Обзор проекта и полный гайд по Kubernetes

  Что из себя представляет проект

  Browser → Traefik (:80)
                ├── /auth/*  → auth-service (:8080)   [Gin, Postgres, Redis, Kafka producer]
                └── /users/* → user-service (:8081)   [chi, MongoDB, Kafka consumer]

  auth-service ──Kafka accounts.events──► user-service

  auth-service — JWT-аутентификация, OAuth Google, сессии в Redis, аккаунты в Postgres. При регистрации пишет событие в outbox_events → flusher горутина
  публикует в Kafka каждые 5 сек.

  user-service — профили пользователей. Консьюмит Kafka, при AccountCreated создаёт документ в MongoDB. Идемпотентно (по account_id).

  Конфигурация грузится из config/config.yaml, переменные окружения имеют приоритет. Секреты (JWT, Google OAuth, пароль PG) — только через env, не хранятся
  в YAML.

  ---
  Состояние k8s — что есть

  Структура k8s/

  k8s/
  ├── namespace.yaml           # namespace: friend-net
  ├── traefik/                 # Traefik как ingress-контроллер
  │   ├── deployment.yaml      # Traefik v2.11
  │   ├── service.yaml         # LoadBalancer :80 + :8080 (dashboard)
  │   ├── rbac.yaml            # ServiceAccount + ClusterRole (нужен для чтения CRD)
  │   └── middlewares.yaml     # jwt-auth (forwardAuth) + api-rate-limit
  ├── infra/
  │   ├── postgres/            # StatefulSet + PVC 5Gi + Secret + Migration Job
  │   ├── mongodb/             # StatefulSet + PVC 5Gi
  │   ├── redis/               # Deployment + PVC 1Gi
  │   └── kafka/               # StatefulSet + PVC 10Gi (KRaft, без Zookeeper)
  ├── auth-service/
  │   ├── deployment.yaml      # 2 реплики, healthz пробы
  │   ├── configmap.yaml       # все env кроме секретов
  │   ├── secret.yaml          # POSTGRES_PASSWORD, JWT_SECRET*, GOOGLE_*
  │   ├── service.yaml         # ClusterIP :8080
  │   └── ingressroute.yaml    # Traefik IngressRoute (CRD)
  └── user-service/
      ├── deployment.yaml      # 2 реплики, TCP probe (нет /healthz)
      ├── configmap.yaml       # env
      ├── service.yaml         # ClusterIP :8081
      └── ingressroute.yaml    # Traefik IngressRoute (CRD)

  ---
  Блокеры — без исправлений кластер не запустится

  ┌─────┬──────────────────────────────────────────────────────────────────────────┬──────────────────────────────┬─────────────────────────────┐
  │  #  │                                 Проблема                                 │             Где              │           Статус            │
  ├─────┼──────────────────────────────────────────────────────────────────────────┼──────────────────────────────┼─────────────────────────────┤
  │ 1   │ golang:1.25-alpine — такой версии не существует (последняя 1.23.x)       │ auth-service/Dockerfile      │ СЛОМАН                      │
  ├─────┼──────────────────────────────────────────────────────────────────────────┼──────────────────────────────┼─────────────────────────────┤
  │ 2   │ /auth/validate — эндпоинта нет в коде, есть только POST /auth/introspect │ k8s/traefik/middlewares.yaml │ СЛОМАН                      │
  ├─────┼──────────────────────────────────────────────────────────────────────────┼──────────────────────────────┼─────────────────────────────┤
  │ 3   │ Секреты — плейсхолдеры your-secret-key...                                │ k8s/auth-service/secret.yaml │ нужно заполнить             │
  ├─────┼──────────────────────────────────────────────────────────────────────────┼──────────────────────────────┼─────────────────────────────┤
  │ 4   │ imagePullPolicy: Never не выставлен                                      │ оба deployment.yaml          │ нужно добавить для minikube │
  └─────┴──────────────────────────────────────────────────────────────────────────┴──────────────────────────────┴─────────────────────────────┘

  Хорошие новости — kubectl и minikube уже установлены, Docker запущен. Кластер не стартован (minikube start не запускался).

  ---
  Понятие о k8s — ликбез перед стартом
  
  Kubernetes (k8s) — система, которая запускает и управляет контейнерами на кластере машин.

  Ключевые объекты:

  ┌─────────────────────────────┬────────────────────────────────────────┬──────────────────────────────────────────────────────────────────────────────┐
  │           Объект            │        Аналог из docker-compose        │                                  Что делает                                  │
  ├─────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────┤
  │ Pod                         │ контейнер                              │ минимальная единица запуска, 1+ контейнеров                                  │
  ├─────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────┤
  │ Deployment                  │ services: + replicas:                  │ поддерживает N реплик пода, умеет rolling update                             │
  ├─────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────┤
  │ StatefulSet                 │ как Deployment, но с постоянным именем │ для БД: postgres-0, kafka-0 — имя не меняется                                │
  ├─────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────┤
  │ Service                     │ ports: / networks:                     │ постоянный DNS-адрес внутри кластера (postgres.friend-net.svc.cluster.local) │
  ├─────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────┤
  │ ConfigMap                   │ environment:                           │ ненужные переменные окружения                                                │
  ├─────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────┤
  │ Secret                      │ .env файл                              │ чувствительные данные (base64-encoded)                                       │
  ├─────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────┤
  │ PersistentVolumeClaim (PVC) │ volumes:                               │ запрос на дисковое пространство                                              │
  ├─────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────┤
  │ Namespace                   │ --project-name в compose               │ изоляция ресурсов внутри кластера                                            │
  ├─────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────┤
  │ IngressRoute                │ labels: traefik.*                      │ маршрутизация HTTP-трафика (CRD Traefik)                                     │
  └─────────────────────────────┴────────────────────────────────────────┴──────────────────────────────────────────────────────────────────────────────┘

  minikube — локальный одноузловой k8s кластер внутри Docker. Удобно для разработки.

  ---
  Полный гайд — шаг за шагом

  Шаг 0 — Исправить Dockerfile auth-service

  # Открой auth-service/Dockerfile и замени:
  # FROM golang:1.25-alpine  →  FROM golang:1.23-alpine
  sed -i 's/golang:1.25-alpine/golang:1.23-alpine/' /home/dryzer/go-projects/friend-net-1/auth-service/Dockerfile

  Шаг 1 — Добавить /auth/validate в auth-service

  Traefik middleware делает GET /auth/validate с заголовком Authorization: Bearer <token>. В коде есть только POST /auth/introspect. Нужно добавить новый
  обработчик (это отдельная задача — скажи, сделаю).

  Временный обход для тестирования: убрать middlewares из ingressroute.yaml у users-private, чтобы запустить без JWT-проверки.

  Шаг 2 — Запустить кластер minikube

  minikube start --driver=docker --memory=4096 --cpus=4

  # Проверить статус
  minikube status
  kubectl get nodes

  ▎ Что происходит: minikube поднимает Docker-контейнер, который симулирует k8s-узел. Внутри него работает API-сервер Kubernetes, в котором мы будем 
  ▎ создавать ресурсы.

  Шаг 3 — Переключить Docker на minikube

  eval $(minikube docker-env)
  # Теперь docker build/images работают ВНУТРИ minikube

  docker build -t auth-service:latest ./auth-service
  docker build -t user-service:latest ./user-service

  # Убедиться что образы видны
  docker images | grep -E "auth|user"

  Шаг 4 — Добавить imagePullPolicy: Never

  В k8s/auth-service/deployment.yaml и k8s/user-service/deployment.yaml под image::

  image: auth-service:latest
  imagePullPolicy: Never   # ← добавить эту строку

  Без этого k8s будет пытаться скачать образ из Docker Hub и не найдёт его.

  Шаг 5 — Заполнить секреты

  # Сгенерировать JWT ключи
  openssl rand -base64 32
  openssl rand -base64 32

  Отредактировать k8s/auth-service/secret.yaml — вставить реальные значения вместо плейсхолдеров. Google OAuth можно оставить пустым если не нужен.

  Шаг 6 — Установить Traefik CRDs

  kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v2.11/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml

  ▎ Почему: IngressRoute, Middleware — это не стандартные k8s объекты, а расширения (Custom Resource Definitions) от Traefik. Без них kubectl apply упадёт с
  ▎  no matches for kind "IngressRoute".

  Шаг 7 — Применить манифесты

  cd /home/dryzer/go-projects/friend-net-1

  # 1. Неймспейс (изолированное пространство для всех ресурсов проекта)
  kubectl apply -f k8s/namespace.yaml

  # 2. Traefik (ingress-контроллер, аналог nginx-proxy)
  kubectl apply -f k8s/traefik/

  # 3. Инфраструктура — порядок важен!
  kubectl apply -f k8s/infra/postgres/
  kubectl apply -f k8s/infra/mongodb/
  kubectl apply -f k8s/infra/redis/
  kubectl apply -f k8s/infra/kafka/

  # 4. Дождаться готовности инфры
  kubectl wait --for=condition=ready pod -l app=postgres -n friend-net --timeout=120s
  kubectl wait --for=condition=ready pod -l app=mongodb -n friend-net --timeout=120s
  kubectl wait --for=condition=ready pod -l app=redis -n friend-net --timeout=120s
  kubectl wait --for=condition=ready pod -l app=kafka -n friend-net --timeout=120s

  # 5. Приложения
  kubectl apply -f k8s/auth-service/
  kubectl apply -f k8s/user-service/

  Шаг 8 — Проверка

  # Все поды в namespace friend-net
  kubectl get pods -n friend-net

  # Ожидаемый вывод:
  # NAME                            READY   STATUS    
  # traefik-xxx                     1/1     Running   
  # auth-service-xxx (x2)           1/1     Running   
  # user-service-xxx (x2)           1/1     Running   
  # postgres-0                      1/1     Running   
  # mongodb-0                       1/1     Running   
  # redis-xxx                       1/1     Running   
  # kafka-0                         1/1     Running   

  # URL для доступа снаружи
  minikube service traefik -n friend-net --url

  # Проверить healthcheck
  curl http://<minikube-ip>/auth/healthz

  # Логи конкретного сервиса
  kubectl logs -l app=auth-service -n friend-net --tail=50
  kubectl logs -l app=postgres -n friend-net --tail=30

  # Войти в под (как docker exec)
  kubectl exec -it deployment/auth-service -n friend-net -- sh

  # Перезапустить деплой после изменений
  kubectl rollout restart deployment/auth-service -n friend-net

  Шаг 9 — Полезные команды для отладки

  # Описание пода с событиями (если не стартует)
  kubectl describe pod <pod-name> -n friend-net

  # Все события в namespace
  kubectl get events -n friend-net --sort-by=.lastTimestamp

  # Статус всех ресурсов
  kubectl get all -n friend-net

  # Остановить кластер (сохраняет состояние)
  minikube stop

  # Удалить кластер полностью
  minikube delete

  ---
  Итоговый чеклист

  - kubectl установлен (v1.36.1)
  - minikube установлен (v1.38.1)
  - Docker запущен
  - Dockerfiles есть в обоих сервисах
  - k8s манифесты написаны
  - Исправить golang:1.25-alpine → golang:1.23-alpine в Dockerfile
  - Добавить GET /auth/validate обработчик (или временно убрать jwt middleware)
  - Заполнить k8s/auth-service/secret.yaml реальными значениями
  - Добавить imagePullPolicy: Never в оба deployment.yaml
  - Запустить minikube start
  - Собрать образы через minikube docker-env
  - Применить манифесты

  ---
  Хочешь, я сразу исправлю все блокеры (Dockerfile, imagePullPolicy, добавлю GET /auth/validate)? Это займёт 5 минут и после этого можно будет запустить
  кластер.
