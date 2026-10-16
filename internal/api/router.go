package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/engine"
	"github.com/execlens/execlens/internal/jobs"
	"github.com/execlens/execlens/internal/languages"
	"github.com/execlens/execlens/internal/reports"
	"github.com/execlens/execlens/internal/workloads"
)

// Server holds API dependencies.
type Server struct {
	db          *database.DB
	pool        *jobs.WorkerPool
	langReg     *languages.Registry
	workloadReg *workloads.Registry
	frontendDir string
}

func NewServer(db *database.DB, pool *jobs.WorkerPool, langReg *languages.Registry, wlReg *workloads.Registry, frontendDir string) *Server {
	return &Server{
		db:          db,
		pool:        pool,
		langReg:     langReg,
		workloadReg: wlReg,
		frontendDir: frontendDir,
	}
}

// Handler sets up all routes using standard library net/http ServeMux with CORS middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// REST API v1
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/stats", s.handleStats)
	mux.HandleFunc("GET /api/v1/languages", s.handleLanguages)
	mux.HandleFunc("GET /api/v1/workloads", s.handleWorkloads)
	mux.HandleFunc("GET /api/v1/runs", s.handleListRuns)
	mux.HandleFunc("POST /api/v1/runs", s.handleCreateRun)
	mux.HandleFunc("GET /api/v1/runs/{runId}", s.handleGetRun)
	mux.HandleFunc("GET /api/v1/runs/{runId}/results", s.handleGetRunResults)
	mux.HandleFunc("POST /api/v1/runs/{runId}/cancel", s.handleCancelRun)
	mux.HandleFunc("GET /api/v1/runs/{runId}/export", s.handleExportRun)
	mux.HandleFunc("GET /api/v1/runs/{runId}/report", s.handleReportRun)
	mux.HandleFunc("GET /api/v1/compare", s.handleCompareRuns)

	// Static frontend assets and SPA fallback
	if s.frontendDir != "" {
		absDir, err := filepath.Abs(s.frontendDir)
		distDir := filepath.Join(s.frontendDir, "dist")
		if err == nil {
			distDir = filepath.Join(absDir, "dist")
		}
		if info, err := os.Stat(distDir); err == nil && info.IsDir() {
			fileServer := http.FileServer(http.Dir(distDir))
			mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/") {
					http.NotFound(w, r)
					return
				}
				// SPA fallback: if file does not exist, serve index.html
				path := filepath.Join(distDir, filepath.Clean(r.URL.Path))
				if _, err := os.Stat(path); os.IsNotExist(err) {
					http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
					return
				}
				fileServer.ServeHTTP(w, r)
			})
		}
	}

	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Helpers
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"code":  code,
		"error": message,
	})
}

// Handlers
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "ExecLens",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetDashboardStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	runtimes := s.langReg.DetectAll(r.Context())
	availCount := 0
	for _, rt := range runtimes {
		if rt.IsAvailable {
			availCount++
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_runs":          stats.TotalRuns,
		"completed_runs":      stats.CompletedRuns,
		"failed_runs":         stats.FailedRuns,
		"last_run_at":         stats.LastRunAt,
		"supported_languages": len(runtimes),
		"available_languages": availCount,
	})
}

func (s *Server) handleLanguages(w http.ResponseWriter, r *http.Request) {
	runtimes := s.langReg.DetectAll(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"languages": runtimes,
	})
}

func (s *Server) handleWorkloads(w http.ResponseWriter, r *http.Request) {
	list := s.workloadReg.List()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"workloads": list,
	})
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if val, err := strconv.Atoi(lStr); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if val, err := strconv.Atoi(oStr); err == nil && val >= 0 {
			offset = val
		}
	}

	workloadID := r.URL.Query().Get("workload_id")
	status := r.URL.Query().Get("status")

	runs, total, err := s.db.ListRuns(r.Context(), limit, offset, workloadID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"runs":   runs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

type CreateRunRequest struct {
	WorkloadID       string                 `json:"workload_id"`
	Languages        []string               `json:"languages"`
	WorkloadParams   map[string]interface{} `json:"workload_params"`
	Iterations       int                    `json:"iterations"`
	WarmupIterations *int                   `json:"warmup_iterations"`
	ExecutionMode    string                 `json:"execution_mode"`
}

func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Failed to parse request body: "+err.Error())
		return
	}

	// Workload validation
	workload, err := s.workloadReg.Get(req.WorkloadID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_WORKLOAD", err.Error())
		return
	}

	// Languages validation
	if len(req.Languages) == 0 {
		writeError(w, http.StatusBadRequest, "NO_LANGUAGES", "At least one language must be selected")
		return
	}
	for _, l := range req.Languages {
		if _, err := s.langReg.Get(l); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_LANGUAGE", fmt.Sprintf("Unknown language '%s'", l))
			return
		}
	}

	// Iterations validation
	if req.Iterations <= 0 {
		req.Iterations = 5
	} else if req.Iterations > 50 {
		writeError(w, http.StatusBadRequest, "INVALID_ITERATIONS", "Iterations cannot exceed 50")
		return
	}

	warmup := 2
	if req.WarmupIterations != nil {
		warmup = *req.WarmupIterations
		if warmup < 0 || warmup > 20 {
			writeError(w, http.StatusBadRequest, "INVALID_WARMUP", "Warmup iterations must be between 0 and 20")
			return
		}
	}

	mode := req.ExecutionMode
	if mode == "" {
		mode = "standard"
	}
	if mode != "standard" && mode != "cold_start" && mode != "quicksort" && mode != "stdlib" {
		writeError(w, http.StatusBadRequest, "INVALID_MODE", "Execution mode must be standard, cold_start, quicksort, or stdlib")
		return
	}

	// Default params if not supplied
	if req.WorkloadParams == nil {
		req.WorkloadParams = make(map[string]interface{})
	}
	for _, p := range workload.Parameters {
		if _, exists := req.WorkloadParams[p.Name]; !exists {
			req.WorkloadParams[p.Name] = p.Default
		}
	}

	// Generate Run ID
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	runID := fmt.Sprintf("run_%s", hex.EncodeToString(b))

	run := &database.Run{
		ID:               runID,
		WorkloadID:       req.WorkloadID,
		WorkloadParams:   req.WorkloadParams,
		Iterations:       req.Iterations,
		WarmupIterations: warmup,
		ExecutionMode:    mode,
		Status:           database.StatusQueued,
		CreatedAt:        time.Now().UTC(),
		SystemInfo:       engine.CollectSystemInfo(),
	}

	if err := s.db.CreateRun(r.Context(), run, req.Languages); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to persist run: "+err.Error())
		return
	}

	// Submit to worker pool
	if err := s.pool.Submit(runID); err != nil {
		_ = s.db.UpdateRunStatus(r.Context(), runID, database.StatusFailed, "Queue submission failed: "+err.Error())
		writeError(w, http.StatusServiceUnavailable, "QUEUE_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")
	baseline := r.URL.Query().Get("baseline")

	detail, err := s.db.GetRunDetail(r.Context(), runID, baseline)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if detail == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Run '%s' not found", runID))
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleGetRunResults(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")
	baseline := r.URL.Query().Get("baseline")

	detail, err := s.db.GetRunDetail(r.Context(), runID, baseline)
	if err != nil || detail == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Run not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"run_id":     detail.ID,
		"executions": detail.Executions,
	})
}

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")
	if err := s.pool.Cancel(runID); err != nil {
		writeError(w, http.StatusBadRequest, "CANCEL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Run '%s' cancellation initiated", runID),
	})
}

func (s *Server) handleExportRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	detail, err := s.db.GetRunDetail(r.Context(), runID, "")
	if err != nil || detail == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Run not found")
		return
	}

	if format == "csv" {
		data, err := reports.ExportRunCSV(detail)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "EXPORT_ERROR", err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"execlens_%s.csv\"", runID))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	data, err := reports.ExportRunJSON(detail)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "EXPORT_ERROR", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"execlens_%s.json\"", runID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleReportRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")
	detail, err := s.db.GetRunDetail(r.Context(), runID, "")
	if err != nil || detail == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Run not found")
		return
	}

	md := reports.GenerateMarkdownReport(detail)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(md))
}

func (s *Server) handleCompareRuns(w http.ResponseWriter, r *http.Request) {
	runIDsParam := r.URL.Query().Get("run_ids")
	if runIDsParam == "" {
		writeError(w, http.StatusBadRequest, "MISSING_PARAM", "Query parameter 'run_ids' required (comma-separated)")
		return
	}

	parts := strings.Split(runIDsParam, ",")
	var runs []database.RunDetail

	for _, id := range parts {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		detail, err := s.db.GetRunDetail(r.Context(), id, "")
		if err == nil && detail != nil {
			runs = append(runs, *detail)
		}
	}

	if len(runs) == 0 {
		writeError(w, http.StatusNotFound, "NO_RUNS_FOUND", "None of the specified runs were found")
		return
	}

	comparison := reports.CompareRuns(runs)
	writeJSON(w, http.StatusOK, comparison)
}
