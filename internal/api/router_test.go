package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/engine"
	"github.com/execlens/execlens/internal/jobs"
	"github.com/execlens/execlens/internal/languages"
	"github.com/execlens/execlens/internal/workloads"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	langReg := languages.NewDefaultRegistry()
	wlReg := workloads.NewRegistry()
	benchmarksDir := filepath.Join(tempDir, "benchmarks")

	executor := engine.NewExecutor(db, langReg, wlReg, benchmarksDir)
	pool := jobs.NewWorkerPool(db, executor, 1)

	server := NewServer(db, pool, langReg, wlReg, nil, "")
	cleanup := func() {
		_ = pool.Shutdown(context.Background())
		_ = db.Close()
	}
	return server, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res["status"] != "healthy" {
		t.Errorf("expected healthy status, got %v", res["status"])
	}
}

func TestLanguagesEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/languages", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	langs, ok := res["languages"].([]interface{})
	if !ok || len(langs) != 6 {
		t.Errorf("expected 6 languages, got %v", len(langs))
	}
}

func TestWorkloadsEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/workloads", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	wls, ok := res["workloads"].([]interface{})
	if !ok || len(wls) != 6 {
		t.Errorf("expected 6 workloads, got %v", len(wls))
	}
}

func TestCreateRunValidation(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Invalid workload ID
	badPayload := `{"workload_id": "nonexistent", "languages": ["go"]}`
	req := httptest.NewRequest("POST", "/api/v1/runs", bytes.NewBufferString(badPayload))
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad workload, got %d", w.Code)
	}

	// 2. Empty languages
	badPayload2 := `{"workload_id": "sieve", "languages": []}`
	req2 := httptest.NewRequest("POST", "/api/v1/runs", bytes.NewBufferString(badPayload2))
	w2 := httptest.NewRecorder()
	server.Handler().ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty languages, got %d", w2.Code)
	}

	// 3. Valid run creation
	validPayload := `{"workload_id": "sieve", "languages": ["go"], "iterations": 3}`
	req3 := httptest.NewRequest("POST", "/api/v1/runs", bytes.NewBufferString(validPayload))
	w3 := httptest.NewRecorder()
	server.Handler().ServeHTTP(w3, req3)
	if w3.Code != http.StatusCreated {
		t.Errorf("expected 201 for valid run, got %d (body: %s)", w3.Code, w3.Body.String())
	}
}

func TestMultitrackEndpoints(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. GET /api/v1/tracks
	req := httptest.NewRequest("GET", "/api/v1/tracks", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/v1/tracks, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "languages") || !strings.Contains(w.Body.String(), "lowlevel") || !strings.Contains(w.Body.String(), "accelerator") {
		t.Errorf("expected tracks in response, got %s", w.Body.String())
	}

	// 2. GET /api/v1/lowlevel/catalog
	reqLL := httptest.NewRequest("GET", "/api/v1/lowlevel/catalog", nil)
	wLL := httptest.NewRecorder()
	server.Handler().ServeHTTP(wLL, reqLL)
	if wLL.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/v1/lowlevel/catalog, got %d", wLL.Code)
	}
	if !strings.Contains(wLL.Body.String(), "lowlevel_reduction") {
		t.Errorf("expected lowlevel_reduction in catalog, got %s", wLL.Body.String())
	}

	// 3. GET /api/v1/accelerators
	reqAcc := httptest.NewRequest("GET", "/api/v1/accelerators", nil)
	wAcc := httptest.NewRecorder()
	server.Handler().ServeHTTP(wAcc, reqAcc)
	if wAcc.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/v1/accelerators, got %d", wAcc.Code)
	}
	if !strings.Contains(wAcc.Body.String(), "cuda") {
		t.Errorf("expected cuda in accelerators list, got %s", wAcc.Body.String())
	}

	// 4. Create Low-Level Run
	llPayload := `{"track": "lowlevel", "workload_id": "lowlevel_reduction", "languages": ["autovec", "simd_avx2"], "iterations": 2}`
	reqCreate := httptest.NewRequest("POST", "/api/v1/runs", bytes.NewBufferString(llPayload))
	wCreate := httptest.NewRecorder()
	server.Handler().ServeHTTP(wCreate, reqCreate)
	if wCreate.Code != http.StatusCreated {
		t.Errorf("expected 201 for lowlevel run creation, got %d (body: %s)", wCreate.Code, wCreate.Body.String())
	}
}
