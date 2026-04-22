# Notifica Carioca

**Sistema de notificações em tempo real para cidadãos do Rio de Janeiro.**

Recebe webhooks de mudanças de status de chamados da prefeitura, persiste no PostgreSQL e entrega ao cidadão via API REST e WebSocket, com privacidade de CPF, idempotência e escalabilidade horizontal.

## Fluxo com LLM

Este repositório inclui skills locais do projeto em `.agents/skills/`.

- Use `@golang-patterns` para implementação, refatorações e revisão de código.
- Use `@golang-testing` para cobertura de regressão, testes orientados a tabelas e planejamento de testes.
- Use `@database-migrations` para mudanças de esquema, revisão de migrações e planejamento de rollout.
- Use `@api-security-best-practices` para endurecimento de endpoints, revisão de autenticação/autorização e checagens de segurança da API.
- Use `@security-review-evidence-first` para revisões de segurança focadas apenas em achados rastreados.
- Use `@hexagonal-architecture` para desenho de funcionalidades e refatorações com fronteiras arquiteturais claras.

Veja `AGENTS.md` para o fluxo de trabalho esperado.

O diretório `.agents/` é versionado intencionalmente neste repositório porque contém fluxo de trabalho compartilhado do time e skills locais reutilizáveis do projeto. Segredos locais e estado específico de desenvolvedor não devem ser commitados:

- use `.env.example` como modelo versionado, não `.env`
- use `bruno/environments/local.example.bru` como modelo versionado do Bruno, não `local.bru`
- mantenha estado de IDE/workspace, como `.idea/` e `.vscode/`, fora do Git

---

## Decisões Arquiteturais

### Por que Redis + PostgreSQL, sem Kafka?

O volume estimado do desafio é bem atendido por Go + PostgreSQL + Redis. Kafka adicionaria complexidade operacional sem benefício proporcional para este cenário.

| Componente | Papel |
|---|---|
| **PostgreSQL** | Fonte da verdade para notificações. A unique constraint garante idempotência. |
| **Redis Pub/Sub** | Ponte entre o webhook recebido e o push WebSocket em tempo real. O filtro por `cpf_hash` é feito localmente no Hub. |
| **Redis Cache** | Cache do contador de não lidas e deduplicação rápida de webhooks como fast-path. |

### Privacidade do CPF

O CPF nunca é armazenado em texto ou em hash reversível.

- **HMAC-SHA256** com chave secreta (`CPF_HASH_KEY`) para evitar rainbow table
- **Determinístico**, permitindo lookup eficiente
- **Separado** de `WEBHOOK_SECRET` e `JWT_SECRET`
- **Validado e normalizado** na entrada

### Idempotência em Duas Camadas

1. **Redis** como fast-path com TTL de 24h
2. **PostgreSQL** como fonte da verdade com `UNIQUE(chamado_id, status_novo, event_timestamp)` e `ON CONFLICT DO NOTHING`

### Paginação por Cursor

Em vez de `OFFSET`, a paginação usa cursor ancorado em `(created_at, id)` para manter performance consistente.

### WebSocket com Canal Único

Uma goroutine subscrita ao canal `notifications` do Redis Pub/Sub recebe eventos e o Hub entrega apenas para conexões do `cpf_hash` correspondente.

---

## Stack

| Tecnologia | Versão | Propósito |
|---|---|---|
| Go | 1.26.2+ | Linguagem principal |
| Gin | v1.10 | Roteador HTTP |
| PostgreSQL | 16 | Persistência |
| Redis | 7 | Pub/Sub, cache e deduplicação |
| pgx | v5 | Driver PostgreSQL |
| gorilla/websocket | v1.5 | Servidor WebSocket |
| golang-jwt | v5 | Validação JWT |
| Docker Compose | v2 | Orquestração local |

---

## Arquitetura

O projeto segue uma arquitetura hexagonal leve: domínio e casos de uso no centro, com HTTP, Redis, PostgreSQL e WebSocket como adapters.

- `internal/domain/`: linguagem e regras de negócio
- `internal/application/`: casos de uso e portas
- `internal/adapters/in/`: HTTP, middleware e entrada WebSocket
- `internal/adapters/out/`: PostgreSQL, Redis, Hub e readiness
- `internal/infra/`: configuração, criptografia e wiring

---

## Endpoints

Documentação técnica detalhada dos contratos e fluxos:

- [docs/API_ENDPOINTS.md](https://github.com/matheuss0xf/notifica-carioca/blob/main/docs/API_ENDPOINTS.md)

| Método | Path | Auth | Descrição |
|---|---|---|---|
| `POST` | `/api/v1/webhooks/status-change` | HMAC-SHA256 (`X-Signature-256`) | Receber webhook de mudança de status |
| `GET` | `/api/v1/notifications?cursor=&limit=20` | JWT Bearer | Listar notificações do cidadão |
| `PATCH` | `/api/v1/notifications/:id/read` | JWT Bearer | Marcar notificação como lida |
| `GET` | `/api/v1/notifications/unread-count` | JWT Bearer | Total de não lidas |
| `GET` | `/ws` | JWT (`Authorization: Bearer <token>`) | WebSocket para push em tempo real |
| `GET` | `/health` | — | Verificação de liveness |
| `GET` | `/ready` | — | Verificação de readiness de PostgreSQL e Redis |

---

## Formato de Erro

As respostas HTTP seguem este formato:

```json
{
  "code": "invalid_field",
  "error": "tipo must be status_change",
  "field": "tipo"
}
```

No webhook de status-change, além do schema JSON, o payload é validado para:

- `tipo` deve ser exatamente `status_change`
- `chamado_id`, `cpf`, `status_novo` e `titulo` não podem ser vazios após trim
- `timestamp` deve estar presente
- `cpf` deve ser válido, com ou sem máscara

---

## Como Executar

### Pré-requisitos

- Docker + Docker Compose
- Go 1.26.2+ para desenvolvimento local
- [just](https://github.com/casey/just) como task runner, opcional

### Com Docker Compose

```bash
just up
# ou
docker compose up --build -d

just logs

just down

just reset
```

### Desenvolvimento Local

```bash
docker compose up -d postgres redis

cp .env.example .env

just dev
# ou
source .env && go run ./cmd/api
```

### Rodando Testes

```bash
just test
just test-race
just test-unit

# Cobertura útil do projeto
go run ./scripts/coverage
```

### Testes HTTP com Bruno

O repositório inclui uma collection do Bruno em [bruno/README.md](https://github.com/matheuss0xf/notifica-carioca/blob/main/bruno/README.md) para validar o fluxo HTTP local com a stack do `docker compose`.

Fluxo sugerido:

1. subir a stack com `just up`
2. abrir a pasta `./bruno` no Bruno
3. selecionar o ambiente `local`
4. executar `Health`, `Ready`, `Webhook - Status Change`, `Notifications - Unread Count` e `Notifications - List`
5. copiar o `id` retornado em `Notifications - List` para a variável `notificationId`
6. executar `Notifications - Mark Read` e validar o `Unread Count` novamente

Atalho:

```bash
just bruno
```

### Validação do Core

Antes de investir em diferenciais, valide o núcleo funcional do sistema usando o roteiro em [docs/VALIDACAO_CORE.md](https://github.com/matheuss0xf/notifica-carioca/blob/main/docs/VALIDACAO_CORE.md).

Esse roteiro cobre:

- saúde da stack
- webhook válido
- idempotência
- API REST
- marcação como lida
- WebSocket
- isolamento por cidadão
- privacidade do CPF

Atalho:

```bash
just validate-core
```

---

## Exemplos de Uso

### Enviar Webhook

```bash
BODY='{"chamado_id":"CH-2024-001","tipo":"status_change","cpf":"529.982.247-25","status_anterior":"aberto","status_novo":"em_execucao","titulo":"Reparo de buraco na Rua X","descricao":"Equipe designada para reparo","timestamp":"2024-03-15T10:30:00Z"}'

SIGNATURE="sha256=$(echo -n "$BODY" | openssl dgst -sha256 -hmac "dev-webhook-secret" | awk '{print $2}')"

curl -X POST http://localhost:8080/api/v1/webhooks/status-change \
  -H "Content-Type: application/json" \
  -H "X-Signature-256: $SIGNATURE" \
  -d "$BODY"
```

### Consultar Notificações

```bash
TOKEN=$(just jwt 52998224725)

curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/notifications?limit=10

curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/notifications/unread-count

curl -X PATCH -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/notifications/{id}/read
```

### WebSocket

```bash
wscat -c "ws://localhost:8080/ws" -H "Authorization: Bearer $TOKEN"
```

---

## Variáveis de Ambiente

| Variável | Obrigatória | Padrão | Descrição |
|---|---|---|---|
| `DATABASE_URL` | ✅ | — | String de conexão do PostgreSQL |
| `WEBHOOK_SECRET` | ✅ | — | Chave HMAC para validar webhooks |
| `CPF_HASH_KEY` | ✅ | — | Chave HMAC para hash de CPF |
| `JWT_SECRET` | ✅ | — | Chave para validar JWT (HS256) |
| `SERVER_PORT` | ❌ | 8080 | Porta do servidor HTTP |
| `REDIS_PASSWORD` | ❌ | dev only | Senha do Redis no ambiente local |
| `REDIS_URL` | ❌ | redis://default:my-redis-password-change-me@localhost:6379/0 | String de conexão do Redis |
| `IDEMPOTENCY_TTL` | ❌ | 24h | TTL da chave de idempotência no Redis |
| `UNREAD_CACHE_TTL` | ❌ | 1h | TTL do cache de não lidas |
| `SHUTDOWN_TIMEOUT` | ❌ | 10s | Timeout para graceful shutdown |
| `READ_HEADER_TIMEOUT` | ❌ | 5s | Timeout para leitura inicial de headers |
| `READ_TIMEOUT` | ❌ | 15s | Timeout total de leitura HTTP |
| `WRITE_TIMEOUT` | ❌ | 30s | Timeout de escrita HTTP |
| `IDLE_TIMEOUT` | ❌ | 60s | Timeout de conexão ociosa |
| `WS_ALLOWED_ORIGINS` | ❌ | vazio | Lista de origins permitidos no WebSocket |

---

## O que Eu Faria com Mais Tempo

- rate limiting por IP no webhook e por cidadão na API
- SSE como fallback do WebSocket
- endpoint batch para receber múltiplos webhooks
- particionamento da tabela `notifications` por mês
- métricas Prometheus
- tracing com OpenTelemetry
- dead letter queue com Redis Streams
- circuit breaker no PostgreSQL e no Redis
- manifestos de Kubernetes com HPA e limites de recursos
- testes de carga com k6
