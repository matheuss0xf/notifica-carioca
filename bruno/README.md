# Collection do Bruno

Collection pronta para testar o fluxo HTTP local da aplicação com a stack do `docker compose`.

## Ambiente

Use o ambiente `local` em `bruno/environments/local.bru`.

Ele já vem com:

- `baseUrl=http://localhost:8080`
- um `bearerToken` estático de desenvolvimento compatível com `JWT_SECRET=dev-jwt-secret`
- variáveis do payload do webhook
- `webhookSecret=dev-webhook-secret`

## Ordem Sugerida

1. `Health`
2. `Ready`
3. `Webhook - Status Change`
4. `Notifications - Unread Count`
5. `Notifications - List`
6. `Notifications - List` já persiste automaticamente o primeiro `id` em `notificationId`
7. `Notifications - Mark Read`
8. `Notifications - Unread Count`

## Observações

- O request de webhook agora assina o body dinamicamente no `script:pre-request`, então você pode editar as variáveis `webhook*` no ambiente sem recalcular HMAC manualmente.
- O script usa `require("crypto")`, então a collection precisa rodar com o sandbox padrão de desenvolvedor do Bruno.
- O fluxo de WebSocket não está incluído na collection do Bruno. Para validar push em tempo real, continue usando um cliente WS com `Authorization: Bearer <token>`.
