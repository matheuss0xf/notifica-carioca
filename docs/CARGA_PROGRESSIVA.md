# Carga Progressiva

Este roteiro serve para estimar a capacidade do serviço sem chutar números como `1M de requisições`.

## Objetivo

Responder, com evidência:

- até onde o webhook sustenta taxa estável
- até onde a API de leitura continua saudável
- quando começam erros, aumento de latência ou saturação de recursos

## Antes de começar

Suba a stack com os limites de rate limiting relaxados no ambiente local:

```bash
just down
just up
```

Confirme saúde:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

Confirme também os testes-base:

```bash
mise exec -- k6 run ./k6/full_flow.js
mise exec -- k6 run ./k6/load_test.js
```

Se algum desses falhar, não avance para carga mais alta.

## O que observar

Durante cada etapa, rode em outro terminal:

```bash
docker stats
```

E, se necessário:

```bash
docker compose logs -f app
docker compose logs -f postgres
docker compose logs -f redis
```

## Critério de aprovação por etapa

Uma etapa é considerada saudável quando:

- `http_req_failed < 5%`
- `checks > 95%`
- `p95 < 1000ms`
- sem reinício de container
- sem erro recorrente em logs do app, Postgres ou Redis

## Etapas sugeridas

### Etapa 0. Baseline

```bash
APP_URL=http://localhost:8080 \
WEBHOOK_SECRET=dev-webhook-secret \
JWT_SECRET=dev-jwt-secret \
mise exec -- k6 run ./k6/load_test.js
```

Objetivo:

- confirmar o baseline já validado

### Etapa 1. Carga leve

```bash
APP_URL=http://localhost:8080 \
WEBHOOK_SECRET=dev-webhook-secret \
JWT_SECRET=dev-jwt-secret \
WEBHOOK_RATE=50 \
WEBHOOK_DURATION=2m \
READ_TARGET_VUS=20 \
READ_RAMP_UP=30s \
READ_STEADY=1m \
READ_RAMP_DOWN=30s \
mise exec -- k6 run ./k6/load_test.js
```

Objetivo:

- validar conforto acima do baseline

### Etapa 2. Carga moderada

```bash
APP_URL=http://localhost:8080 \
WEBHOOK_SECRET=dev-webhook-secret \
JWT_SECRET=dev-jwt-secret \
WEBHOOK_RATE=100 \
WEBHOOK_DURATION=3m \
WEBHOOK_PREALLOCATED_VUS=50 \
WEBHOOK_MAX_VUS=150 \
READ_TARGET_VUS=40 \
READ_RAMP_UP=30s \
READ_STEADY=2m \
READ_RAMP_DOWN=30s \
mise exec -- k6 run ./k6/load_test.js
```

Objetivo:

- encontrar o primeiro sinal de degradação

### Etapa 3. Carga alta

```bash
APP_URL=http://localhost:8080 \
WEBHOOK_SECRET=dev-webhook-secret \
JWT_SECRET=dev-jwt-secret \
WEBHOOK_RATE=250 \
WEBHOOK_DURATION=5m \
WEBHOOK_PREALLOCATED_VUS=100 \
WEBHOOK_MAX_VUS=300 \
READ_TARGET_VUS=80 \
READ_RAMP_UP=45s \
READ_STEADY=4m \
READ_RAMP_DOWN=45s \
mise exec -- k6 run ./k6/load_test.js
```

Objetivo:

- testar o limite prático do ambiente local

### Etapa 4. Stress controlado

```bash
APP_URL=http://localhost:8080 \
WEBHOOK_SECRET=dev-webhook-secret \
JWT_SECRET=dev-jwt-secret \
WEBHOOK_RATE=500 \
WEBHOOK_DURATION=5m \
WEBHOOK_PREALLOCATED_VUS=150 \
WEBHOOK_MAX_VUS=500 \
READ_TARGET_VUS=120 \
READ_RAMP_UP=1m \
READ_STEADY=4m \
READ_RAMP_DOWN=1m \
mise exec -- k6 run ./k6/load_test.js
```

Objetivo:

- descobrir o ponto de quebra do ambiente local

## Como estimar volume total

O número de requests depende da combinação de webhook e leitura.

Exemplo aproximado da Etapa 3:

- webhook: `250 req/s` por `300s` = `75.000` requests
- leitura: cada iteração faz `2` requests (`list` + `unread-count`)
- total de leitura depende do pacing real dos VUs

Para chegar perto de `1M`, você precisa acumular volume, por exemplo:

- `500 req/s` por `20 minutos` no webhook já gera `600.000` requests
- somando leituras concorrentes, o total pode ultrapassar `1M`

Mas isso só vale se a etapa continuar saudável.

## Interpretação correta

Se o sistema passar em uma etapa mais agressiva, a conclusão correta é:

- “neste ambiente, sustentou X req/s por Y minutos com p95 de Z ms e erro abaixo de 5%”

Não conclua:

- “aguenta 1M” sem dizer em qual janela de tempo, com qual mix de tráfego e em qual ambiente

## Resultado que vale colocar no README

Quando você terminar, registre algo no formato:

- ambiente: Docker Compose local
- cenário: `load_test.js`
- webhook: `N req/s` por `M min`
- leitura: `V` VUs
- `p95`
- taxa de erro
- observações de CPU/memória

Esse formato é defensável e evita marketing sem evidência.

## Exemplo real de registro

Exemplo de resultado já validado neste projeto:

- ambiente: Docker Compose local
- cenário: `k6/load_test.js`
- webhook: `50 req/s` por `2 min`
- leitura: até `20` VUs
- total HTTP: `9621` requests
- `p95`: `2.69ms`
- taxa de erro HTTP: `0%`
- checks: `100%`

Leitura correta desse resultado:

- o serviço ficou estável nesse degrau
- serve como baseline forte para subir para a etapa moderada com segurança
