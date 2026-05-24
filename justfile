set dotenv-load := false

default:
    @just --list

setup-frontend:
    cd frontend && bun install

setup-backend:
    cd backend && go mod tidy

setup-root:
    bun install

setup: setup-root setup-frontend setup-backend install-hooks
    @echo "ready"

install-hooks:
    bun x @j178/prek install

gen:
    buf generate

clean-gen:
    rm -rf backend/gen frontend/src/api

regen: clean-gen gen

dev backend_addr=":8080" frontend_port="3000":
    ./scripts/dev.sh {{backend_addr}} {{frontend_port}}

build-frontend:
    cd frontend && bun run build && bun run .output/server/index.mjs

build-backend:
    cd backend && go build ./...

fmt-backend:
    cd backend && find . -name '*.go' -print0 | xargs -0 gofmt -w

lint-backend:
    cd backend && go vet ./...

check:
    env NO_COLOR=1 bun x @j178/prek run --all-files

ai-login-openai:
	cd backend && go run ./cmd/ai-auth login-openai

ai-status-openai:
	cd backend && go run ./cmd/ai-auth status-openai

ai-logout-openai:
	cd backend && go run ./cmd/ai-auth logout-openai

dev-openai-oauth backend_addr=":8080" frontend_port="3000" model="gpt-5-nano" reasoning_effort="low":
	env STINT_AI_AUTH_MODE=openai-oauth STINT_AI_MODEL={{model}} STINT_AI_REASONING_EFFORT={{reasoning_effort}} ./scripts/dev.sh {{backend_addr}} {{frontend_port}}

dev-openai-api backend_addr=":8080" frontend_port="3000" base_url="https://api.openai.com/v1" model="gpt-5-nano" api_key="" reasoning_effort="low":
	env STINT_AI_AUTH_MODE=api-key STINT_AI_BASE_URL={{base_url}} STINT_AI_API_MODEL={{model}} STINT_AI_API_KEY={{api_key}} STINT_AI_REASONING_EFFORT={{reasoning_effort}} ./scripts/dev.sh {{backend_addr}} {{frontend_port}}
