# Fluxo de Trabalho com Agentes

Este repositório versiona skills locais do projeto para implementação e revisão em Go.

## Política do Repositório

- `.agents/` faz parte do fluxo de trabalho compartilhado do projeto e deve ser versionado quando contiver skills reutilizáveis, orientações e convenções do time.
- Não armazene credenciais, sessões locais, caches específicos da máquina ou qualquer outro estado local de desenvolvedor dentro de `.agents/`.
- Mantenha segredos locais fora do Git:
  - `.env`
  - ambientes locais do Bruno, como `bruno/environments/local.bru`
  - estado de IDE e workspace, como `.idea/` e `.vscode/`
- Quando uma ferramenta local precisar de um exemplo versionado, prefira um modelo sanitizado, como `.env.example` ou `bruno/environments/local.example.bru`.

## Skills Preferenciais

- `@golang-patterns`
  Use para implementação em Go, refatorações, revisão de código, desenho de API, mudanças de concorrência e decisões de tratamento de erro.
- `@golang-testing`
  Use para novos testes, cobertura de regressão, testes orientados a tabelas, subtestes e planejamento de benchmark ou fuzzing.
- `@database-migrations`
  Use para mudanças de esquema, revisão de migrações, planejamento de rollout, backfills e estratégia de rollback.
- `@api-security-best-practices`
  Use para desenho de endpoints, revisão de autenticação e autorização, validação de entrada, segurança de WebSocket e endurecimento da API.
- `@security-review-evidence-first`
  Use para revisões de endpoint e controle de acesso quando você precisar apenas de achados rastreados e não teóricos.
- `@hexagonal-architecture`
  Use para desenho de funcionalidades e refatorações que toquem fronteiras entre domínio, aplicação, portas, adapters e wiring.

## Onde Elas Ficam

- `.agents/skills/golang-patterns/SKILL.md`
- `.agents/skills/golang-testing/SKILL.md`
- `.agents/skills/database-migrations/SKILL.md`
- `.agents/skills/api-security-best-practices/SKILL.md`
- `.agents/skills/security-review-evidence-first/SKILL.md`
- `.agents/skills/hexagonal-architecture/SKILL.md`

## Fluxo Esperado para Mudanças em Go

1. Revise a skill local relevante antes de fazer mudanças não triviais em Go.
2. Mantenha o código idiomático e explícito: fluxo de controle simples, wrapping contextual de erros, interfaces pequenas e tipos amigáveis ao zero value.
3. Adicione testes de regressão para mudanças de comportamento. Prefira testes orientados a tabelas e subtestes com `t.Run(...)`.
4. Execute, quando disponível:
   - `go test ./...`
   - `go test -race ./...`
   - `go test -cover ./...`
5. Para mudanças de esquema, crie arquivos de migração imutáveis em `migrations/` e mantenha DDL e DML separados quando a mudança não for trivial.
6. Para endpoints novos ou alterados, revise autenticação, garantia de ownership, validação, comportamento de origin/CORS e controles contra abuso.
7. Para revisões de segurança, reporte apenas achados rastreados com confiança HIGH/MEDIUM e ignore questões puramente teóricas.
8. Para refatorações estruturais, preserve a direção das dependências para dentro e mantenha o wiring centralizado.

## Observações Específicas do Projeto

- Trate mudanças de WebSocket com cuidado em concorrência e ciclo de vida.
- Mantenha a lógica de idempotência do webhook alinhada entre Redis e PostgreSQL.
- Restrinja o acesso às notificações e à paginação estritamente ao dono autenticado.
- Não flexibilize regras de origin ou autenticação em endpoints expostos a navegadores sem um knob explícito de configuração.
- Para mudanças de banco, prefira mudanças aditivas primeiro, planeje passos de contrato depois e evite editar migrações já implantadas.
- Mantenha os erros de API sanitizados, evite vazar detalhes internos e trate tokens em query string como um fallback de maior risco.
- Adicione limitação de taxa ao expor novos endpoints públicos ou sujeitos a abuso.
- Em achados de segurança, prefira fluxos rastreados de ponta a ponta em vez de auditorias baseadas só em checklist.
- Mantenha as regras de negócio em domínio/aplicação; mantenha detalhes de Gin, Redis, PostgreSQL e WebSocket em adapters e wiring.

## Fonte

Estas skills locais foram derivadas das skills instaladas do marketplace:

- `affaan-m-everything-claude-code-golang-patterns`
- `affaan-m-everything-claude-code-golang-testing`
- `affaan-m-everything-claude-code-database-migrations`
- `davila7-claude-code-templates-api-security-best-practices`
- `getsentry-sentry-sentry-security`
- `affaan-m-everything-claude-code-hexagonal-architecture`
