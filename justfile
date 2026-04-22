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
    
# Show k6 load-test usage
load:
    @echo "k6 script: ./k6/load_test.js"
    @echo "If k6 is not installed: mise install"
    @echo "Run example: k6 run ./k6/load_test.js"
    @echo "Optional env vars: APP_URL, WEBHOOK_SECRET, JWT_SECRET, WEBHOOK_RATE, WEBHOOK_DURATION, READ_TARGET_VUS"

# Show full flow k6 usage
load-flow:
    @echo "k6 script: ./k6/full_flow.js"
    @echo "Run example: mise exec -- k6 run ./k6/full_flow.js"
    @echo "Optional env vars: APP_URL, WEBHOOK_SECRET, JWT_SECRET, FLOW_VUS, FLOW_ITERATIONS, WS_FLOW_VUS, WS_FLOW_ITERATIONS, WS_TIMEOUT_MS"

# Show progressive capacity test guide
load-capacity:
    @echo "Progressive load guide: ./docs/CARGA_PROGRESSIVA.md"
    @echo "Suggested order: baseline -> carga leve -> carga moderada -> carga alta -> stress controlado"

# Tidy dependencies
tidy:
    go mod tidy
