.PHONY: all build build-backend build-frontend test test-unit test-integration smoke lint run clean benchmark report report-html report-verify benchmark-report clean-report

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

benchmark: build-backend
	@echo "Running representative benchmark suite across available runtimes..."
	./execlens run --workload=sieve --iterations=5 --warmup=2
	./execlens run --workload=sort --iterations=3 --mode=quicksort
	./execlens lowlevel --workload=lowlevel_reduction --variants=scalar,autovec,simd_avx2,openmp,asm
	./execlens lowlevel --workload=lowlevel_memcpy --variants=memcpy,scalar,unroll64,avx2_stream

report: build-backend
	@echo "Generating ExecLens automated benchmark reports and visualizations..."
	./execlens report-gen --output=reports --baseline=cpp --update-readme=true

report-html: report
	@echo "Standalone HTML report available at: reports/latest/report.html"

report-verify: build-backend
	@echo "Verifying benchmark report artifacts, checksums, and README markers..."
	./execlens report-verify --dir=reports/latest --readme=README.md

benchmark-report: benchmark report report-verify
	@echo "Benchmark execution, report generation, and verification complete!"

clean-report:
	@echo "Cleaning generated reports..."
	rm -rf reports/

run: build
	./execlens server --port 8080

clean:
	rm -f execlens
	rm -rf frontend/dist
	rm -f execlens.db execlens.db-wal execlens.db-shm

