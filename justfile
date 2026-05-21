set dotenv-load := false

default:
    @just --list

setup-frontend:
    cd frontend && bun install

setup-backend:
    cd backend && go mod tidy

setup-root:
    bun install

setup: setup-root setup-frontend setup-backend
    @echo "ready"

install-hooks:
    bunx prek install

gen:
    buf generate

clean-gen:
    rm -rf backend/gen frontend/src/api

regen: clean-gen gen

dev-frontend:
    cd frontend && bun dev

dev-backend:
    cd backend && go run ./cmd/server

dev:
    just dev-backend

check-frontend:
    cd frontend && bun run biome check src

check-backend:
    cd backend && go test ./...

check: check-frontend check-backend

build-frontend:
    cd frontend && bun run build && bun run .output/server/index.mjs

build-backend:
    cd backend && go build ./...
