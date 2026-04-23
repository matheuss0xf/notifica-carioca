# Kubernetes Manifests

Manifests enxutos para rodar o `Notifica Carioca` em um cluster Kubernetes local.

Inclui:

- `namespace.yaml`
- `configmap.yaml`
- `examples/secret.example.yaml`
- `postgres.yaml`
- `redis.yaml`
- `api.yaml`
- `network-policy.yaml`
- `hpa.yaml` opcional

## O que esses manifests assumem

- voce ja tem um cluster local, por exemplo Minikube
- a imagem da API sera carregada no cluster com a tag `notifica-carioca:local`
- PostgreSQL e Redis sobem no proprio cluster, com `Service` interno

## Rodando localmente com Minikube

### 1. Suba o cluster

```bash
minikube start
```

Se quiser usar o HPA localmente:

```bash
minikube addons enable metrics-server
```

### 2. Gere a imagem da API

No diretorio raiz do projeto:

```bash
docker build -t notifica-carioca:local .
```

### 3. Carregue a imagem no Minikube

```bash
minikube image load notifica-carioca:local
```

### 4. Crie o Secret localmente

Nao versione credenciais no Git. Gere o Secret diretamente no cluster:

```bash
kubectl apply -f deploy/k8s/namespace.yaml

export POSTGRES_USER=notifica
export POSTGRES_PASSWORD='<set-locally>'
export POSTGRES_DB=notifica_carioca
export REDIS_PASSWORD='<set-locally>'
export WEBHOOK_SECRET='<set-locally>'
export CPF_HASH_KEY='<set-locally>'
export JWT_SECRET='<set-locally>'

kubectl -n notifica-carioca create secret generic notifica-carioca-secrets \
  --from-literal=POSTGRES_USER="$POSTGRES_USER" \
  --from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
  --from-literal=POSTGRES_DB="$POSTGRES_DB" \
  --from-literal=REDIS_PASSWORD="$REDIS_PASSWORD" \
  --from-literal=DATABASE_URL="postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@postgres:5432/$POSTGRES_DB?sslmode=disable" \
  --from-literal=REDIS_URL="redis://default:$REDIS_PASSWORD@redis:6379/0" \
  --from-literal=WEBHOOK_SECRET="$WEBHOOK_SECRET" \
  --from-literal=CPF_HASH_KEY="$CPF_HASH_KEY" \
  --from-literal=JWT_SECRET="$JWT_SECRET"
```

Se precisar recriar:

```bash
kubectl -n notifica-carioca delete secret notifica-carioca-secrets
```

O arquivo `examples/secret.example.yaml` existe apenas como referencia do formato esperado.

### 5. Aplique os manifests

```bash
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/postgres.yaml
kubectl apply -f deploy/k8s/redis.yaml
kubectl apply -f deploy/k8s/api.yaml
kubectl apply -f deploy/k8s/network-policy.yaml
```

Se quiser testar autoscaling:

```bash
kubectl apply -f deploy/k8s/hpa.yaml
```

### 6. Espere os pods ficarem prontos

```bash
kubectl -n notifica-carioca get pods -w
```

### 7. Exponha a API no localhost

Use `port-forward`:

```bash
kubectl -n notifica-carioca port-forward svc/notifica-carioca-api 8080:8080
```

Com isso, a API fica disponivel em:

```text
http://localhost:8080
```

### 8. Teste

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Observabilidade basica

Confira os recursos:

```bash
kubectl -n notifica-carioca get all
kubectl -n notifica-carioca get pvc
kubectl -n notifica-carioca logs deploy/notifica-carioca-api
```

## Limpando tudo

```bash
kubectl delete namespace notifica-carioca
```

## Notas

- `examples/secret.example.yaml` e apenas um modelo sanitizado; ele nao deve ser aplicado como fonte de verdade.
- se `secret.yaml` ja chegou a existir em algum commit local ou remoto, faca rotacao das senhas e chaves antes de publicar.
- os manifests usam `port-forward` como forma mais simples de acesso local; nao ha `Ingress` nem `NodePort`.
- `network-policy.yaml` depende do CNI do cluster aplicar `NetworkPolicy`; no Minikube com alguns drivers isso pode ser limitado.
