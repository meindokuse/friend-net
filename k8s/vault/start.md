  HashiCorp Vault + Kubernetes — полный гайд с нуля
  
  Как это работает (концептуально)

  Pod запускается
      │
      ├── vault-agent-init (init-контейнер, добавляется автоматически)
      │       │  Аутентифицируется в Vault через ServiceAccount токен
      │       │  Скачивает секреты и пишет их в /vault/secrets/ (shared emptyDir)
      │       └── завершается
      │
      └── твой контейнер стартует
              Читает /vault/secrets/env → экспортирует переменные → exec бинаря

  Никаких k8s Secret объектов — секреты живут только в Vault и в памяти пода.

  ---
  Что изменилось в манифестах

  ┌───────────────────────────────────┬────────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │               Файл                │                                              Что сделано                                               │
  ├───────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ auth-service/deployment.yaml      │ + serviceAccountName, + Vault annotations, command заменён на sh -c ". /vault/secrets/env && exec ..." │
  ├───────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ infra/postgres/statefulset.yaml   │ + serviceAccountName, + Vault annotations, envFrom secretRef → env *_FILE                              │
  ├───────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ infra/postgres/migration-job.yaml │ + Vault annotations, команда сорсит /vault/secrets/pgenv                                               │
  ├───────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ auth-service/secret.yaml          │ Заменён комментарием — не применять                                                           │
  ├───────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ infra/postgres/secret.yaml        │ Заменён комментарием — не применять                                                           │
  ├───────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ k8s/vault/                        │ Новая директория — всё для Vault                                                           │
  └───────────────────────────────────┴────────────────────────────────────────────────────────────────────────────────────────────────────────┘

  user-service — без изменений: у него нет секретов (MongoDB без авторизации, конфиги публичные).

  ---
  Шаг 0 — Установить Helm

  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
  helm version

  ---
  Шаг 1 — Запустить minikube (если ещё не запущен)

  minikube start --driver=docker --memory=5120 --cpus=4
  # 5Gi памяти — Vault + вся инфра требуют больше чем стандартные 2Gi

  ---
  Шаг 2 — Создать namespace и применить неймспейс Vault

  # Namespace для приложений
  kubectl apply -f k8s/namespace.yaml

  # Namespace для Vault (изолируем от приложений)
  kubectl create namespace vault

  ---
  Шаг 3 — Установить Vault через Helm

  # Добавить репозиторий HashiCorp
  helm repo add hashicorp https://helm.releases.hashicorp.com
  helm repo update

  # Установить Vault в namespace vault
  helm install vault hashicorp/vault \
    --namespace vault \
    --values k8s/vault/vault-values.yaml

  # Дождаться запуска пода (статус будет 0/1 Running — это нормально до unseal)
  kubectl get pods -n vault -w

  ---
  Шаг 4 — Инициализировать и распечатать (unseal) Vault

  # Инициализировать Vault — одноразово, сохрани вывод!
  kubectl exec -n vault vault-0 -- vault operator init \
    -key-shares=1 \
    -key-threshold=1

  # Вывод будет примерно таким:
  # Unseal Key 1: abc123...
  # Initial Root Token: hvs.xyz...

  # СОХРАНИ ОБА ЗНАЧЕНИЯ — они нужны каждый раз после рестарта пода

  # Распечатать (unseal) — вставь свой Unseal Key
  kubectl exec -n vault vault-0 -- vault operator unseal <UNSEAL_KEY>

  # Проверить статус — Initialized: true, Sealed: false
  kubectl exec -n vault vault-0 -- vault status

  ---
  Шаг 5 — Настроить Vault (политики, роли, Kubernetes auth)

  # Войти в под Vault с root токеном
  kubectl exec -it -n vault vault-0 -- sh

  # Внутри пода — логин
  vault login <ROOT_TOKEN>

  # Включить KV v2
  vault secrets enable -path=secret kv-v2

  # Включить Kubernetes auth
  vault auth enable kubernetes

  # Настроить Kubernetes auth (используем токен самого пода Vault)
  vault write auth/kubernetes/config \
    kubernetes_host="https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}"\
    kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
    token_reviewer_jwt="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"

  # Политика для auth-service (только чтение своих секретов)
  vault policy write auth-service - <<'EOF'
  path "secret/data/friend-net/auth" {
    capabilities = ["read"]
  }
  EOF

  # Политика для postgres
  vault policy write postgres - <<'EOF'
  path "secret/data/friend-net/postgres" {
    capabilities = ["read"]
  }
  EOF

  # Роль для auth-service — разрешает SA "auth-service" в namespace "friend-net"
  vault write auth/kubernetes/role/auth-service \
    bound_service_account_names=auth-service \
    bound_service_account_namespaces=friend-net \
    policies=auth-service \
    ttl=1h

  # Роль для postgres
  vault write auth/kubernetes/role/postgres \
    bound_service_account_names=postgres \
    bound_service_account_namespaces=friend-net \
    policies=postgres \
    ttl=1h

  exit

  ---
  Шаг 6 — Записать секреты в Vault

  # Войти в под ещё раз
  kubectl exec -it -n vault vault-0 -- sh
  vault login <ROOT_TOKEN>

  # Секреты PostgreSQL
  vault kv put secret/friend-net/postgres \
    postgres_user="postgres" \
    postgres_password="$(openssl rand -base64 24)" \
    postgres_db="auth_service_new"

  # Секреты auth-service
  vault kv put secret/friend-net/auth \
    postgres_password="ТОТЖЕ_ПАРОЛЬ_ЧТО_ВЫШЕ" \
    jwt_secret="$(openssl rand -base64 32)" \
    jwt_refresh_secret="$(openssl rand -base64 32)" \
    google_client_id="твой-google-client-id" \
    google_client_secret="твой-google-client-secret"

  # Проверить что записалось
  vault kv get secret/friend-net/postgres
  vault kv get secret/friend-net/auth

  exit

  ▎ postgres_password должен быть одинаковым в обоих путях — postgres им создаёт БД, auth-service им коннектится.

  ---
  Шаг 7 — Применить ServiceAccounts

  kubectl apply -f k8s/vault/serviceaccounts.yaml

  ---
  Шаг 8 — Применить всё остальное (обычный порядок)

  # Traefik CRDs + Traefik
  kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v2.11/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml
  kubectl apply -f k8s/traefik/

  # Инфраструктура (secret.yaml НЕ применяем — он заменён комментарием)
  kubectl apply -f k8s/infra/postgres/pvc.yaml
  kubectl apply -f k8s/infra/postgres/service.yaml
  kubectl apply -f k8s/infra/postgres/migration-configmap.yaml
  kubectl apply -f k8s/infra/postgres/statefulset.yaml

  kubectl apply -f k8s/infra/mongodb/
  kubectl apply -f k8s/infra/redis/
  kubectl apply -f k8s/infra/kafka/

  # Дождаться готовности postgres
  kubectl wait --for=condition=ready pod -l app=postgres -n friend-net --timeout=120s

  # Запустить миграцию
  kubectl apply -f k8s/infra/postgres/migration-job.yaml

  # Приложения (secret.yaml НЕ применяем)
  kubectl apply -f k8s/auth-service/configmap.yaml
  kubectl apply -f k8s/auth-service/deployment.yaml
  kubectl apply -f k8s/auth-service/service.yaml
  kubectl apply -f k8s/auth-service/ingressroute.yaml

  kubectl apply -f k8s/user-service/

  ---
  Шаг 9 — Проверка

  # Посмотреть все поды — vault-agent-init должен быть Completed
  kubectl get pods -n friend-net

  # Логи vault-agent init контейнера auth-service пода (если что-то не так)
  kubectl logs <auth-service-pod> -n friend-net -c vault-agent-init

  # Логи самого сервиса
  kubectl logs -l app=auth-service -n friend-net

  # Vault UI (в браузере)
  kubectl port-forward -n vault vault-0 8200:8200
  # открыть http://localhost:8200 — логин root токеном

  ---
  Что происходит при старте пода (детально)

  1. k8s видит аннотации vault.hashicorp.com/* на поде
  2. Vault Injector (мутирующий webhook) добавляет init-контейнер vault-agent-init
  3. vault-agent-init:
     - берёт ServiceAccount токен пода из /var/run/secrets/kubernetes.io/serviceaccount/token
     - отправляет POST vault/vault:8200/v1/auth/kubernetes/login с этим токеном и role=auth-service
     - Vault проверяет токен через k8s API → выдаёт Vault токен
     - vault-agent рендерит шаблон → пишет /vault/secrets/env
     - init-контейнер завершается
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  kubectl port-forward -n vault vault-0 8200:8200
  # открыть http://localhost:8200 — логин root токеном

  ---
  Что происходит при старте пода (детально)

  1. k8s видит аннотации vault.hashicorp.com/* на поде
  2. Vault Injector (мутирующий webhook) добавляет init-контейнер vault-agent-init
  3. vault-agent-init:
     - берёт ServiceAccount токен пода из /var/run/secrets/kubernetes.io/serviceaccount/token
     - отправляет POST vault/vault:8200/v1/auth/kubernetes/login с этим токеном и role=auth-service
     - Vault проверяет токен через k8s API → выдаёт Vault токен
     - vault-agent рендерит шаблон → пишет /vault/secrets/env
     - init-контейнер завершается
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  # открыть http://localhost:8200 — логин root токеном

  ---
  Что происходит при старте пода (детально)

  1. k8s видит аннотации vault.hashicorp.com/* на поде
  2. Vault Injector (мутирующий webhook) добавляет init-контейнер vault-agent-init
  3. vault-agent-init:
     - берёт ServiceAccount токен пода из /var/run/secrets/kubernetes.io/serviceaccount/token
     - отправляет POST vault/vault:8200/v1/auth/kubernetes/login с этим токеном и role=auth-service
     - Vault проверяет токен через k8s API → выдаёт Vault токен
     - vault-agent рендерит шаблон → пишет /vault/secrets/env
     - init-контейнер завершается
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
     - vault-agent рендерит шаблон → пишет /vault/secrets/env
     - init-контейнер завершается
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service        │                               - init-контейнер завершается
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service        │                          4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service        │
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤                    - Vault проверяет токен через k8s API → выдаёт Vault токен
     - vault-agent рендерит шаблон → пишет /vault/secrets/env
     - init-контейнер завершается
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service        │             3. vault-agent-init:
     - берёт ServiceAccount токен пода из /var/run/secrets/kubernetes.io/serviceaccount/token
     - отправляет POST vault/vault:8200/v1/auth/kubernetes/login с этим токеном и role=auth-service
     - Vault проверяет токен через k8s API → выдаёт Vault токен
     - vault-agent рендерит шаблон → пишет /vault/secrets/env
     - init-контейнер завершается
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤          4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤      4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤     4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service        │
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов
     - init-контейнер завершается
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  ┌────────────────────────────┬──────────────────────┬─────────────────────────────────────┐
  │         Vault path         │         Ключ         │             Потребитель        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_user        │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_password    │ StatefulSet postgres, migration job │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/postgres │ postgres_db          │ StatefulSet postgres        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ postgres_password    │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_secret           │ auth-service        │
  ├────────────────────────────┼──────────────────────┼─────────────────────────────────────┤
  │ secret/friend-net/auth     │ jwt_refresh_secret   │ auth-service  # Инфраструктура (secret.yaml НЕ применяем — он заменён комментарием)
  kubectl apply -f k8s/infra/postgres/pvc.yaml
  kubectl apply -f k8s/infra/postgres/service.yaml
  kubectl apply -f k8s/infra/postgres/migration-configmap.yaml
  kubectl apply -f k8s/infra/postgres/statefulset.yaml

  kubectl apply -f k8s/infra/mongodb/
  kubectl apply -f k8s/infra/redis/
  kubectl apply -f k8s/infra/kafka/

  # Дождаться готовности postgres
  kubectl wait --for=condition=ready pod -l app=postgres -n friend-net
  --timeout=120s

  # Запустить миграцию
  kubectl apply -f k8s/infra/postgres/migration-job.yaml

  # Приложения (secret.yaml НЕ применяем)
  kubectl apply -f k8s/auth-service/configmap.yaml
  kubectl apply -f k8s/auth-service/deployment.yaml
  kubectl apply -f k8s/auth-service/service.yaml
  kubectl apply -f k8s/auth-service/ingressroute.yaml

  kubectl apply -f k8s/user-service/

  ---
  Шаг 9 — Проверка

  # Посмотреть все поды — vault-agent-init должен быть Completed
  kubectl get pods -n friend-net

  # Логи vault-agent init контейнера auth-service пода (если что-то не так)
  kubectl logs <auth-service-pod> -n friend-net -c vault-agent-init

  # Логи самого сервиса
  kubectl logs -l app=auth-service -n friend-net

  # Vault UI (в браузере)
  kubectl port-forward -n vault vault-0 8200:8200
  # открыть http://localhost:8200 — логин root токеном

  ---
  Что происходит при старте пода (детально)

  1. k8s видит аннотации vault.hashicorp.com/* на поде
  2. Vault Injector (мутирующий webhook) добавляет init-контейнер vault-agent-init
  3. vault-agent-init:
     - берёт ServiceAccount токен пода из
  /var/run/secrets/kubernetes.io/serviceaccount/token
     - отправляет POST vault/vault:8200/v1/auth/kubernetes/login с этим токеном и
  role=auth-service
     - Vault проверяет токен через k8s API → выдаёт Vault токен
     - vault-agent рендерит шаблон → пишет /vault/secrets/env
     - init-контейнер завершается
  4. Стартует твой контейнер:
     sh -c "set -a && . /vault/secrets/env && set +a && exec /app/auth-service"
     - set -a: все переменные из sourced файла будут экспортированы
     - exec: заменяет sh процессом бинаря (PID 1 — корректный graceful shutdown)

  ---
  Карта секретов

  Vault path: secret/friend-net/postgres
  Ключ: postgres_user
  Потребитель: StatefulSet postgres
  ────────────────────────────────────────
  Vault path: secret/friend-net/postgres
  Ключ: postgres_password
  Потребитель: StatefulSet postgres, migration job
  ────────────────────────────────────────
  Vault path: secret/friend-net/postgres
  Ключ: postgres_db
  Потребитель: StatefulSet postgres
  ────────────────────────────────────────
  Vault path: secret/friend-net/auth
  Ключ: postgres_password
  Потребитель: auth-service
  ────────────────────────────────────────
  Vault path: secret/friend-net/auth
  Ключ: jwt_secret
  Потребитель: auth-service
  ────────────────────────────────────────
  Vault path: secret/friend-net/auth
  Ключ: jwt_refresh_secret
  Потребитель: auth-service
  ────────────────────────────────────────
  Vault path: secret/friend-net/auth
  Ключ: google_client_id
  Потребитель: auth-service
  ────────────────────────────────────────
  Vault path: secret/friend-net/auth
  Ключ: google_client_secret
  Потребитель: auth-service

  user-service — секретов нет (MongoDB без auth). Если захочешь добавить MongoDB
  auth — скажи, адаптируем.
