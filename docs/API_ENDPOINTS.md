# Endpoints da API

Documentação técnica dos endpoints do Notifica Carioca.

Este documento descreve:

- contrato HTTP de cada endpoint
- autenticação e autorização
- validações e regras de negócio
- fluxo completo dentro da aplicação
- integração com PostgreSQL, Redis e WebSocket

## Visão Geral

O serviço tem três portas de entrada:

- `POST /api/v1/webhooks/status-change`
- `GET/PATCH /api/v1/notifications...`
- `GET /ws`

Fluxo macro:

1. Um webhook de mudança de status chega assinado com HMAC.
2. O evento é validado, normalizado e processado de forma idempotente.
3. A notificação é persistida no PostgreSQL com `cpf_hash`, nunca com CPF puro.
4. O cache e a deduplicação rápida usam Redis.
5. O evento persistido é publicado em Redis Pub/Sub.
6. As instâncias conectadas distribuem a notificação em tempo real para os clientes WebSocket daquele cidadão.
7. O app do cidadão também pode consultar histórico e contador via REST.

## Convenções

### Autenticação

- Webhook: header `X-Signature-256: sha256=<hmac>`
- REST: `Authorization: Bearer <jwt>`
- WebSocket: `Authorization: Bearer <jwt>` no handshake

### Identidade do Cidadão

O JWT deve conter o CPF em `preferred_username`.

Esse CPF:

1. é validado
2. é normalizado para só dígitos
3. é transformado em `cpf_hash` com HMAC-SHA256 usando `CPF_HASH_KEY`

Esse `cpf_hash` é a identidade usada no banco, cache e roteamento de notificações.

### Formato de Erro

Erros HTTP seguem este contrato:

```json
{
  "code": "invalid_field",
  "error": "tipo must be status_change",
  "field": "tipo"
}
```

## 1. Webhook

### POST `/api/v1/webhooks/status-change`

Recebe atualizações de status do sistema da prefeitura.

### Headers

```http
Content-Type: application/json
X-Signature-256: sha256=<hex>
```

### Payload

```json
{
  "chamado_id": "CH-2024-001234",
  "tipo": "status_change",
  "cpf": "529.982.247-25",
  "status_anterior": "em_analise",
  "status_novo": "em_execucao",
  "titulo": "Buraco na Rua - Atualização",
  "descricao": "Equipe designada para reparo na Rua das Laranjeiras, 100",
  "timestamp": "2026-04-21T21:38:19Z"
}
```

### Regras

- `tipo` deve ser exatamente `status_change`
- `chamado_id`, `cpf`, `status_novo` e `titulo` não podem ser vazios
- `timestamp` deve existir
- `cpf` deve ser válido
- a assinatura HMAC deve bater com o body bruto

### Respostas Principais

Evento novo processado:

```json
{
  "message": "notification created",
  "notification_id": "dbb51d17-2949-4bdf-a53c-edf1a45fe193"
}
```

Evento duplicado:

```json
{
  "message": "webhook already processed for chamado",
  "chamado_id": "CH-2024-001234"
}
```

Erros comuns:

- `401 missing_signature`
- `401 invalid_signature`
- `400 invalid_request`
- `400 invalid_field`
- `400 invalid_cpf`

### Fluxo Interno Completo

1. O middleware de assinatura lê o body cru.
2. O HMAC-SHA256 do body é validado contra `WEBHOOK_SECRET`.
3. O body é recolocado no request para o handler conseguir fazer bind do JSON.
4. O handler valida campos obrigatórios e normaliza strings.
5. O caso de uso `WebhookProcessor` valida e normaliza o CPF.
6. O CPF normalizado vira `cpf_hash`.
7. A chave de idempotência é montada com:
   - `chamado_id`
   - `status_novo`
   - `timestamp` com precisão de nanos
8. O Redis é consultado como fast-path de deduplicação.
9. Se não houver duplicata, a notificação é inserida no PostgreSQL.
10. O PostgreSQL protege novamente com `ON CONFLICT DO NOTHING`.
11. Se a inserção foi nova:
   - grava a chave de idempotência no Redis
   - invalida o cache de unread count
   - publica o evento em Redis Pub/Sub
12. O subscriber local recebe o evento e entrega ao hub WebSocket.

### Garantias

- reenvio do mesmo evento não gera duplicata
- CPF nunca vai em texto para o banco
- se Redis falhar, a fonte de verdade ainda é o PostgreSQL

## 2. Listar Notificações

### GET `/api/v1/notifications?cursor=<uuid>&limit=20`

Lista notificações do cidadão autenticado.

### Auth

```http
Authorization: Bearer <jwt>
```

### Query Params

- `limit`: opcional, padrão `20`, máximo `50`
- `cursor`: opcional, UUID do último item da página anterior

### Resposta

```json
{
  "data": [
    {
      "id": "dbb51d17-2949-4bdf-a53c-edf1a45fe193",
      "chamado_id": "CH-2024-001234",
      "tipo": "status_change",
      "status_anterior": "em_analise",
      "status_novo": "em_execucao",
      "titulo": "Buraco na Rua - Atualização",
      "descricao": "Equipe designada para reparo na Rua das Laranjeiras, 100",
      "event_timestamp": "2026-04-21T21:38:19Z",
      "created_at": "2026-04-21T21:38:19Z"
    }
  ],
  "next_cursor": "dbb51d17-2949-4bdf-a53c-edf1a45fe193",
  "has_more": false
}
```

### Fluxo Interno Completo

1. O middleware JWT valida o token.
2. `preferred_username` é lido do token.
3. O CPF é validado, normalizado e convertido em `cpf_hash`.
4. O handler lê `cursor` e `limit`.
5. `limit` inválido cai para `20`; se for maior que `50`, é truncado para `50`.
6. O caso de uso `NotificationReader` chama o repositório.
7. O PostgreSQL consulta apenas notificações com aquele `cpf_hash`.
8. Se houver cursor, a consulta ancora no registro daquele mesmo dono.
9. O resultado volta em ordem `created_at DESC, id DESC`.

### Garantias

- um cidadão não acessa notificações de outro
- o cursor também é escopado ao dono
- a página não usa `OFFSET`

## 3. Marcar Como Lida

### PATCH `/api/v1/notifications/:id/read`

Marca uma notificação como lida para o cidadão autenticado.

### Auth

```http
Authorization: Bearer <jwt>
```

### Resposta de Sucesso

```json
{
  "message": "notification marked as read"
}
```

### Erros Comuns

- `401 unauthorized`
- `400 invalid_notification_id`
- `404 notification_not_found`

### Fluxo Interno Completo

1. O middleware JWT resolve o `cpf_hash`.
2. O handler valida se `:id` é UUID válido.
3. O caso de uso `NotificationMarker` chama o repositório.
4. O `UPDATE` no PostgreSQL só funciona se:
   - a notificação existir
   - pertencer ao `cpf_hash`
   - `read_at` ainda for `NULL`
5. Se atualizou, o cache de unread count desse `cpf_hash` é invalidado.

### Garantias

- não dá para marcar notificação de outro usuário
- repetir a operação em item já lido não quebra consistência; apenas retorna `404 notification_not_found`

## 4. Contador de Não Lidas

### GET `/api/v1/notifications/unread-count`

Retorna o total de notificações não lidas do cidadão autenticado.

### Auth

```http
Authorization: Bearer <jwt>
```

### Resposta

```json
{
  "count": 3
}
```

### Fluxo Interno Completo

1. O middleware JWT resolve o `cpf_hash`.
2. O caso de uso `NotificationReader` tenta primeiro ler o valor no Redis.
3. Se houver cache hit, responde dali.
4. Se houver cache miss ou erro no Redis:
   - faz `COUNT(*)` no PostgreSQL
   - tenta escrever o valor no cache
5. Quando uma notificação nova chega ou é marcada como lida, esse cache é invalidado.

### Garantias

- o cache melhora leitura repetida
- erro de cache não derruba a API; ela cai para o banco

## 5. WebSocket

### GET `/ws`

Canal de entrega em tempo real.

### Auth

Handshake HTTP com:

```http
Authorization: Bearer <jwt>
```

### Mensagem Enviada ao Cliente

```json
{
  "id": "dbb51d17-2949-4bdf-a53c-edf1a45fe193",
  "chamado_id": "CH-2024-001234",
  "tipo": "status_change",
  "status_anterior": "em_analise",
  "status_novo": "em_execucao",
  "titulo": "Buraco na Rua - Atualização",
  "descricao": "Equipe designada para reparo na Rua das Laranjeiras, 100",
  "event_timestamp": "2026-04-21T21:38:19Z",
  "created_at": "2026-04-21T21:38:19Z"
}
```

### Fluxo Interno Completo

1. O handler extrai o token do header `Authorization`.
2. O token é validado da mesma forma que nos endpoints REST.
3. O `cpf_hash` resultante identifica o grupo de conexões daquele cidadão.
4. O upgrade para WebSocket acontece.
5. O client é registrado no hub em memória local.
6. Quando um webhook novo é persistido:
   - a notificação é publicada em Redis Pub/Sub
   - o subscriber recebe a mensagem
   - o dispatcher chama o hub
   - o hub envia a mensagem apenas para os clients com aquele `cpf_hash`

### Origin e Segurança

- `Origin` vazio é aceito para clientes não-browser
- browsers só passam se estiverem em `WS_ALLOWED_ORIGINS`
- token em query string não é aceito

## 6. Verificação de Saúde

### GET `/health`

Verificação simples de liveness do processo.

### Resposta

```json
{
  "status": "ok"
}
```

## 7. Verificação de Prontidão

### GET `/ready`

Verificação de readiness das dependências de runtime.

### Resposta Pronta

```json
{
  "status": "ready",
  "ws_connections": 0
}
```

### Resposta Não Pronta

```json
{
  "status": "not_ready",
  "ws_connections": 0
}
```

### Fluxo Interno Completo

1. Faz `Ping` no PostgreSQL.
2. Faz `Ping` no Redis.
3. Retorna também o número atual de conexões WebSocket locais.
4. Se alguma dependência falhar:
   - retorna `503`
   - não expõe erro bruto de infraestrutura no body

## Mapa de Dependências por Endpoint

### Webhook

- entrada: Gin handler + middleware de assinatura
- caso de uso: `WebhookProcessor`
- saída: PostgreSQL, Redis cache/idempotência, Redis Pub/Sub

### Notifications REST

- entrada: Gin handler + middleware JWT
- casos de uso: `NotificationReader`, `NotificationMarker`
- saída: PostgreSQL + Redis cache

### WebSocket

- entrada: Gin handler + upgrade Gorilla WebSocket
- autenticação: middleware JWT reutilizado no parser de token
- saída: hub local em memória + eventos via Redis Pub/Sub

## Observações de Entrega

- o endpoint de webhook usa `201` para evento novo e `200` para evento duplicado
- a deduplicação é deliberadamente feita por `chamado_id + status_novo + event_timestamp`
- o Redis melhora performance e tempo real, mas o PostgreSQL continua sendo a fonte da verdade
