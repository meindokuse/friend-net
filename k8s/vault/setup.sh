#!/bin/sh
# Run this script INSIDE the vault pod after unseal:
#   kubectl exec -it vault-0 -n vault -- sh /tmp/setup.sh
# (copy to pod first: kubectl cp k8s/vault/setup.sh vault/vault-0:/tmp/setup.sh)
set -e

echo ">>> Enabling KV v2 engine at secret/"
vault secrets enable -path=secret kv-v2 || echo "already enabled"

echo ">>> Enabling Kubernetes auth method"
vault auth enable kubernetes || echo "already enabled"

echo ">>> Configuring Kubernetes auth (using in-cluster token)"
vault write auth/kubernetes/config \
  kubernetes_host="https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}" \
  kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
  token_reviewer_jwt="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"

echo ">>> Writing policy: auth-service"
vault policy write auth-service - <<'EOF'
path "secret/data/friend-net/auth" {
  capabilities = ["read"]
}
EOF

echo ">>> Writing policy: postgres"
vault policy write postgres - <<'EOF'
path "secret/data/friend-net/postgres" {
  capabilities = ["read"]
}
EOF

echo ">>> Creating Kubernetes role: auth-service"
vault write auth/kubernetes/role/auth-service \
  bound_service_account_names=auth-service \
  bound_service_account_namespaces=friend-net \
  policies=auth-service \
  ttl=1h

echo ">>> Creating Kubernetes role: postgres"
vault write auth/kubernetes/role/postgres \
  bound_service_account_names=postgres \
  bound_service_account_namespaces=friend-net \
  policies=postgres \
  ttl=1h

echo ""
echo "=== DONE — now put the actual secrets ==="
echo ""
echo "vault kv put secret/friend-net/postgres \\"
echo "  postgres_user=postgres \\"
echo "  postgres_password=YOUR_STRONG_PASSWORD \\"
echo "  postgres_db=auth_service_new"
echo ""
echo "vault kv put secret/friend-net/auth \\"
echo "  postgres_password=YOUR_STRONG_PASSWORD \\"
echo "  jwt_secret=\$(openssl rand -base64 32) \\"
echo "  jwt_refresh_secret=\$(openssl rand -base64 32) \\"
echo "  google_client_id=YOUR_GOOGLE_CLIENT_ID \\"
echo "  google_client_secret=YOUR_GOOGLE_CLIENT_SECRET"
