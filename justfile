# Notifica Carioca — Task Runner
# Usage: just <recipe>

# Default recipe — show available commands
default:
    @just --list

# Start all services from scratch
up:
    docker compose up --build -d
    docker compose ps

# Stop all services
down:
    docker compose down

# Reset everything (volumes included)
reset:
    docker compose down -v
    docker compose up --build -d
    docker compose ps

# View logs
logs:
    docker compose logs -f app

# Run all tests
test:
    go test ./... -v -count=1

# Run project coverage excluding bootstrap/scripts packages that do not add signal
cover:
    go run ./scripts/coverage

# Run tests with race detector
test-race:
    go test ./... -v -race -count=1

# Run only unit tests (no integration)
test-unit:
    go test ./... -v -short -count=1

# Run linter
lint:
    golangci-lint run ./...

# Format code
fmt:
    gofmt -s -w .

# Run locally (requires .env and running PG/Redis)
dev:
    @cp -n .env.example .env 2>/dev/null || true
    @set -a && source .env && set +a && go run ./cmd/api

# Generate a test JWT for development
jwt cpf="12345678901":
    @go run ./scripts/gen_jwt {{cpf}}

# Send a test webhook
webhook:
    @go run ./scripts/send_webhook

# Show Bruno collection location and suggested usage
bruno:
    @echo "Bruno collection: ./bruno"
    @echo "Use environment: local"
    @echo "Suggested order: Health -> Ready -> Webhook - Status Change -> Notifications - Unread Count -> Notifications - List -> Notifications - Mark Read"

# Show the core validation guide
validate-core:
    @echo "Core validation guide: ./docs/VALIDACAO_CORE.md"
    @echo "Suggested order: stack health -> webhook -> idempotency -> REST -> mark read -> websocket -> ownership -> CPF privacy"

# Tidy dependencies
tidy:
    go mod tidy
