.PHONY: all build build-backend build-frontend test test-unit test-integration smoke lint run clean

all: build

build: build-frontend build-backend

build-frontend:
	@echo "Building frontend..."
	npm --prefix frontend install
	npm --prefix frontend run build

build-backend:
	@echo "Building execlens binary..."
	go build -trimpath -o execlens ./cmd/execlens

test: test-unit test-integration

test-unit:
	@echo "Running Go unit tests..."
	go test -v -race ./internal/...

test-integration:
	@echo "Running Go integration tests..."
	go test -v ./tests/...

smoke: build
	@echo "Running end-to-end smoke test..."
	./scripts/smoke_test.sh

lint:
	@echo "Linting Go code..."
	go vet ./internal/... ./cmd/... ./tests/...
	@echo "Linting frontend code..."
	npm --prefix frontend run lint

run: build
	./execlens server --port 8080

clean:
	rm -f execlens
	rm -rf frontend/dist
	rm -f execlens.db execlens.db-wal execlens.db-shm
