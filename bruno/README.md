# Bruno Collection

Collection pronta para demonstrar o fluxo HTTP local da aplicacao com a stack do `docker compose`.

## Ambiente

Use o ambiente `local` em `bruno/environments/local.bru`.

Ele ja vem com:

- `baseUrl=http://localhost:8080`
- `jwtSecret=dev-jwt-secret`
- `webhookSecret=dev-webhook-secret`
- variaveis do payload do webhook
- geracao automatica de JWT para as rotas autenticadas
- geracao automatica de `webhookChamadoId` e `webhookTimestamp` a cada execucao do webhook

## Fluxo Sugerido

1. `Health`
2. `Ready`
3. `Webhook - Status Change`
4. `Notifications - Unread Count`
5. `Notifications - List`
6. `Notifications - Mark Read`
7. `Notifications - Unread Count`

## O que a collection faz automaticamente

- Assina o body do webhook com HMAC-SHA256 no `script:pre-request`
- Gera um `chamado_id` unico por execucao para evitar bater na idempotencia sem querer
- Gera um JWT HS256 de desenvolvimento usando o mesmo CPF do webhook
- Salva o `notificationId` retornado pelo webhook ou encontrado na listagem
- Salva `lastUnreadCount` e `nextCursor` para inspecao rapida
- Falha cedo quando uma resposta nao segue o contrato esperado

## Variaveis importantes

- `webhookCpf`: CPF usado no payload do webhook e no claim `preferred_username` do JWT
- `jwtSecret`: precisa bater com `JWT_SECRET` da API
- `webhookSecret`: precisa bater com `WEBHOOK_SECRET` da API
- `webhookChamadoPrefix`: prefixo usado para gerar o `chamado_id` dinamico

## WebSocket

O fluxo de WebSocket nao esta dentro da collection do Bruno.

Para validar o push em tempo real junto com o fluxo HTTP:

1. Rode qualquer request autenticado da collection para gerar o `bearerToken`
2. Abra um cliente WS com `Authorization: Bearer <bearerToken>`
3. Conecte em `ws://localhost:8080/ws`
4. Execute `Webhook - Status Change`
5. Verifique o recebimento da notificacao no cliente WS

Exemplo com `wscat`:

```bash
wscat -c ws://localhost:8080/ws -H "Authorization: Bearer <bearerToken>"
```

## Observacoes

- Os scripts usam `require("crypto-js")`, entao a collection precisa rodar com o sandbox padrao de desenvolvedor do Bruno.
- Se voce quiser testar idempotencia, execute o webhook uma vez, copie os valores gerados de `webhookChamadoId` e `webhookTimestamp`, fixe esses valores no ambiente e repita o request.
