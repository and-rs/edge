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

dev backend_addr=":8080" frontend_port="3000":
    ./scripts/dev.sh {{backend_addr}} {{frontend_port}}

build-frontend:
    cd frontend && bun run build && bun run .output/server/index.mjs

build-backend:
    cd backend && go build ./...
