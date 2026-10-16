package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/execlens/execlens/internal/accelerators"
	"github.com/execlens/execlens/internal/api"
	"github.com/execlens/execlens/internal/config"
	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/engine"
	"github.com/execlens/execlens/internal/jobs"
	"github.com/execlens/execlens/internal/languages"
	"github.com/execlens/execlens/internal/lowlevel"
	"github.com/execlens/execlens/internal/reports"
	"github.com/execlens/execlens/internal/workloads"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "server":
		runServer(os.Args[2:])
	case "run":
		runBenchmark(os.Args[2:])
	case "lowlevel":
		runLowLevel(os.Args[2:])
	case "cuda":
		runCUDA(os.Args[2:])
	case "accelerators":
		listAccelerators()
	case "languages":
		listLanguages()
	case "workloads":
		listWorkloads()
	case "list":
		listRuns(os.Args[2:])
	case "report":
		if len(os.Args) == 2 || (len(os.Args) > 2 && strings.HasPrefix(os.Args[2], "-")) {
			runReportGen(os.Args[2:])
		} else {
			showReport(os.Args[2:])
		}
	case "report-gen":
		runReportGen(os.Args[2:])
	case "report-verify":
		runReportVerify(os.Args[2:])
	case "export":
		exportRun(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("ExecLens v%s — Measure fairly. Compare deeply. Understand why.\n", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ExecLens — Cross-Language, Low-Level & GPU Systems Performance Engineering Platform

Usage:
  execlens <command> [arguments]

Core Commands:
  server        Start the REST API and web dashboard
  run           Execute a cross-language benchmark from CLI
  lowlevel      Execute a CPU low-level / compiler optimization benchmark
  cuda          Execute a GPU CUDA benchmark (with CPU reference)
  accelerators  Inspect detected accelerator backends (CUDA, OpenCL, SYCL, HIP)
  languages     Inspect supported languages and host runtime availability
  workloads     List available benchmark workloads and contracts
  list          View historical benchmark runs
  report        Generate comprehensive benchmark reports and charts (or display single run)
  report-gen    Generate automated reports, vector SVG charts, and update README
  report-verify Validate report schemas, artifact checksums, image links, and README markers
  export        Export run results in JSON or CSV format
  version       Show ExecLens version

Run 'execlens <command> --help' for details on a specific command.`)
}

func runServer(args []string) {
	cfg := config.Load()
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	cfg.BindFlags(fs)
	_ = fs.Parse(args)

	log.Printf("[ExecLens] Initializing database at %s...", cfg.DBPath)
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.ReconcileInterruptedRuns(context.Background()); err != nil {
		log.Printf("Warning: error reconciling interrupted runs: %v", err)
	}

	langReg := languages.NewDefaultRegistry()
	wlReg := workloads.NewRegistry()
	accelReg := accelerators.NewRegistry(context.Background())

	executor := engine.NewExecutor(db, langReg, wlReg, cfg.BenchmarksDir)
	lowlevelRunner := lowlevel.NewRunner(db, cfg.BenchmarksDir)
	accelRunner := accelerators.NewRunner(db, accelReg, cfg.BenchmarksDir)
	executor.RegisterTrackRunner("lowlevel", lowlevelRunner)
	executor.RegisterTrackRunner("accelerator", accelRunner)

	pool := jobs.NewWorkerPool(db, executor, cfg.Concurrency)

	server := api.NewServer(db, pool, langReg, wlReg, accelReg, cfg.FrontendDir)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[ExecLens] Server listening on http://%s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	log.Println("[ExecLens] Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = pool.Shutdown(shutdownCtx)
	_ = httpServer.Shutdown(shutdownCtx)
	log.Println("[ExecLens] Goodbye.")
}

func runBenchmark(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	workloadID := fs.String("workload", "sieve", "Workload ID")
	langsFlag := fs.String("languages", "go,rust,cpp,python,javascript,java", "Comma-separated language list")
	track := fs.String("track", "languages", "Track: languages, lowlevel, accelerator")
	iterations := fs.Int("iterations", 5, "Number of timed iterations")
	warmup := fs.Int("warmup", 2, "Number of warm-up iterations")
	mode := fs.String("mode", "standard", "Execution mode: standard, cold_start, quicksort, stdlib")
	dbPath := fs.String("db", "execlens.db", "Database file path")
	benchmarksDir := fs.String("benchmarks", "benchmarks", "Benchmarks directory")
	_ = fs.Parse(args)

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	langReg := languages.NewDefaultRegistry()
	wlReg := workloads.NewRegistry()
	accelReg := accelerators.NewRegistry(context.Background())

	executor := engine.NewExecutor(db, langReg, wlReg, *benchmarksDir)
	lowlevelRunner := lowlevel.NewRunner(db, *benchmarksDir)
	accelRunner := accelerators.NewRunner(db, accelReg, *benchmarksDir)
	executor.RegisterTrackRunner("lowlevel", lowlevelRunner)
	executor.RegisterTrackRunner("accelerator", accelRunner)

	selectedLangs := strings.Split(*langsFlag, ",")
	for i := range selectedLangs {
		selectedLangs[i] = strings.TrimSpace(selectedLangs[i])
	}

	b := make([]byte, 6)
	_, _ = rand.Read(b)
	runID := fmt.Sprintf("run_%s", hex.EncodeToString(b))

	params := make(map[string]interface{})
	if *track == "languages" {
		wl, err := wlReg.Get(*workloadID)
		if err != nil {
			log.Fatalf("Invalid workload: %v", err)
		}
		for _, p := range wl.Parameters {
			params[p.Name] = p.Default
		}
	}

	run := &database.Run{
		ID:               runID,
		Track:            *track,
		WorkloadID:       *workloadID,
		WorkloadParams:   params,
		Iterations:       *iterations,
		WarmupIterations: *warmup,
		ExecutionMode:    *mode,
		Status:           database.StatusRunning,
		CreatedAt:        time.Now().UTC(),
		SystemInfo:       engine.CollectSystemInfo(),
	}

	if err := db.CreateRun(context.Background(), run, selectedLangs); err != nil {
		log.Fatalf("Failed to create run: %v", err)
	}

	fmt.Printf("\n🚀 ExecLens: Starting Benchmark Run [%s]\n", runID)
	fmt.Printf("Track: %s | Workload: %s\n", *track, *workloadID)
	fmt.Printf("Targets: %s\n", strings.Join(selectedLangs, ", "))
	fmt.Printf("Config: %d iterations, %d warmups, mode=%s\n\n", *iterations, *warmup, *mode)

	if err := executor.ExecuteRun(context.Background(), runID); err != nil {
		fmt.Printf("❌ Run execution finished with error: %v\n", err)
	}

	detail, err := db.GetRunDetail(context.Background(), runID, "")
	if err != nil {
		log.Fatalf("Failed to load results: %v", err)
	}

	fmt.Println("\n" + reports.GenerateMarkdownReport(detail))
}

func runLowLevel(args []string) {
	fs := flag.NewFlagSet("lowlevel", flag.ExitOnError)
	workloadID := fs.String("workload", "lowlevel_reduction", "Workload ID (lowlevel_reduction, lowlevel_dot_product, lowlevel_layout, lowlevel_pointer_chase, lowlevel_memcpy)")
	variantsFlag := fs.String("variants", "", "Comma-separated variant list (e.g. scalar,autovec,simd_avx2,openmp,asm)")
	iterations := fs.Int("iterations", 3, "Number of timed iterations")
	dbPath := fs.String("db", "execlens.db", "Database file path")
	benchmarksDir := fs.String("benchmarks", "benchmarks", "Benchmarks directory")
	_ = fs.Parse(args)

	wl, err := lowlevel.GetWorkload(*workloadID)
	if err != nil {
		log.Fatalf("Invalid low-level workload: %v", err)
	}

	var variants []string
	if *variantsFlag != "" {
		for _, v := range strings.Split(*variantsFlag, ",") {
			variants = append(variants, strings.TrimSpace(v))
		}
	} else {
		for _, v := range wl.Variants {
			variants = append(variants, v.ID)
		}
	}

	runArgs := []string{
		"-track=lowlevel",
		fmt.Sprintf("-workload=%s", *workloadID),
		fmt.Sprintf("-languages=%s", strings.Join(variants, ",")),
		fmt.Sprintf("-iterations=%d", *iterations),
		fmt.Sprintf("-db=%s", *dbPath),
		fmt.Sprintf("-benchmarks=%s", *benchmarksDir),
	}
	runBenchmark(runArgs)
}

func runCUDA(args []string) {
	fs := flag.NewFlagSet("cuda", flag.ExitOnError)
	workloadID := fs.String("workload", "cuda_vec_add_saxpy", "Workload ID (cuda_vec_add_saxpy, cuda_parallel_reduction, cuda_matrix_mul, cuda_convolution_1d, cuda_histogram_scan, cuda_bandwidth_transfer)")
	iterations := fs.Int("iterations", 3, "Number of timed iterations")
	dbPath := fs.String("db", "execlens.db", "Database file path")
	benchmarksDir := fs.String("benchmarks", "benchmarks", "Benchmarks directory")
	_ = fs.Parse(args)

	runArgs := []string{
		"-track=accelerator",
		fmt.Sprintf("-workload=%s", *workloadID),
		"-languages=cuda",
		fmt.Sprintf("-iterations=%d", *iterations),
		fmt.Sprintf("-db=%s", *dbPath),
		fmt.Sprintf("-benchmarks=%s", *benchmarksDir),
	}
	runBenchmark(runArgs)
}

func listAccelerators() {
	reg := accelerators.NewRegistry(context.Background())
	backends := reg.GetAll()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tBACKEND\tSTATUS\tDEVICE / DETAILS")
	fmt.Fprintln(w, "--\t-------\t------\t----------------")

	for _, b := range backends {
		status := "✅ Available"
		detail := fmt.Sprintf("%s (%s, CC %s)", b.DeviceName, b.ToolkitVersion, b.ComputeCapability)
		if !b.IsAvailable {
			status = "❌ Unavailable"
			detail = b.MissingReason
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", b.ID, b.DisplayName, status, detail)
	}
	w.Flush()
}

func listLanguages() {
	langReg := languages.NewDefaultRegistry()
	runtimes := langReg.DetectAll(context.Background())

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tAVAILABLE\tVERSION / REASON\tWARMUP")
	fmt.Fprintln(w, "--\t----\t---------\t----------------\t------")

	for _, rt := range runtimes {
		status := "✅ Yes"
		ver := rt.Version
		if !rt.IsAvailable {
			status = "❌ No"
			ver = rt.MissingReason
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", rt.ID, rt.DisplayName, status, ver, rt.DefaultWarmup)
	}
	w.Flush()
}

func listWorkloads() {
	wlReg := workloads.NewRegistry()
	list := wlReg.List()

	fmt.Println("=== Languages Track Workloads ===")
	for _, w := range list {
		fmt.Printf("• %s (%s, v%s)\n", w.Name, w.ID, w.Version)
		fmt.Printf("  Purpose: %s\n", w.AlgorithmPurpose)
		fmt.Printf("  Modes: %s\n\n", strings.Join(w.SupportedModes, ", "))
	}

	fmt.Println("=== Low-Level Code Lab Workloads ===")
	for _, w := range lowlevel.Catalog {
		fmt.Printf("• %s (%s)\n", w.Name, w.ID)
		fmt.Printf("  %s\n\n", w.Description)
	}

	fmt.Println("=== GPU Accelerator Lab Workloads ===")
	for _, w := range accelerators.Catalog {
		fmt.Printf("• %s (%s)\n", w.Name, w.ID)
		fmt.Printf("  %s\n\n", w.Description)
	}
}

func listRuns(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	dbPath := fs.String("db", "execlens.db", "Database file path")
	limit := fs.Int("limit", 20, "Max runs to show")
	track := fs.String("track", "", "Filter by track (languages, lowlevel, accelerator)")
	_ = fs.Parse(args)

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	runs, total, err := db.ListRuns(context.Background(), *limit, 0, *track, "", "")
	if err != nil {
		log.Fatalf("Failed to query runs: %v", err)
	}

	fmt.Printf("Total runs recorded: %d (showing up to %d)\n\n", total, *limit)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RUN ID\tTRACK\tWORKLOAD\tSTATUS\tITERATIONS\tMODE\tCREATED (UTC)")
	fmt.Fprintln(w, "------\t-----\t--------\t------\t----------\t----\t-------------")

	for _, r := range runs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			r.ID, r.Track, r.WorkloadID, r.Status, r.Iterations, r.ExecutionMode, r.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()
}

func showReport(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: execlens report <run_id> [--db=path]")
		os.Exit(1)
	}
	runID := args[0]
	dbPath := "execlens.db"
	for _, a := range args[1:] {
		if strings.HasPrefix(a, "--db=") {
			dbPath = strings.TrimPrefix(a, "--db=")
		}
	}

	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	detail, err := db.GetRunDetail(context.Background(), runID, "")
	if err != nil || detail == nil {
		log.Fatalf("Run '%s' not found: %v", runID, err)
	}

	fmt.Println(reports.GenerateMarkdownReport(detail))
}

func exportRun(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: execlens export <run_id> [--format=json|csv] [--db=path]")
		os.Exit(1)
	}
	runID := args[0]
	format := "json"
	dbPath := "execlens.db"

	for _, a := range args[1:] {
		if strings.HasPrefix(a, "--format=") {
			format = strings.TrimPrefix(a, "--format=")
		} else if strings.HasPrefix(a, "--db=") {
			dbPath = strings.TrimPrefix(a, "--db=")
		}
	}

	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	detail, err := db.GetRunDetail(context.Background(), runID, "")
	if err != nil || detail == nil {
		log.Fatalf("Run '%s' not found: %v", runID, err)
	}

	if format == "csv" {
		data, err := reports.ExportRunCSV(detail)
		if err != nil {
			log.Fatalf("Export error: %v", err)
		}
		os.Stdout.Write(data)
	} else {
		data, err := reports.ExportRunJSON(detail)
		if err != nil {
			log.Fatalf("Export error: %v", err)
		}
		os.Stdout.Write(data)
	}
}

func runReportGen(args []string) {
	cfg := reports.DefaultReportConfig()
	fs := flag.NewFlagSet("report-gen", flag.ExitOnError)

	dbPath := "execlens.db"
	runIDsStr := ""
	updateReadme := true

	fs.StringVar(&dbPath, "db", "execlens.db", "Path to SQLite database")
	fs.StringVar(&cfg.OutputDir, "output", "reports", "Output directory for reports and charts")
	fs.StringVar(&cfg.BaselineID, "baseline", "cpp", "Baseline candidate for speedup comparison")
	fs.StringVar(&cfg.WorkloadFilter, "workload", "", "Optional workload filter")
	fs.StringVar(&cfg.TrackFilter, "track", "", "Optional track filter (languages, lowlevel, gpu)")
	fs.StringVar(&runIDsStr, "run-ids", "", "Comma-separated list of specific run IDs")
	fs.BoolVar(&updateReadme, "update-readme", true, "Update designated section in README.md")
	fs.StringVar(&cfg.ReadmePath, "readme", "README.md", "Path to README.md")
	fs.Float64Var(&cfg.PracticalThresholdPercent, "threshold", 5.0, "Practical significance threshold in percent")

	_ = fs.Parse(args)
	cfg.DBPath = dbPath
	cfg.UpdateReadme = updateReadme

	if runIDsStr != "" {
		for _, id := range strings.Split(runIDsStr, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				cfg.RunIDs = append(cfg.RunIDs, id)
			}
		}
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Printf("[ExecLens Report] Generating report from database '%s'...\n", cfg.DBPath)
	res, err := reports.RunGenerator(context.Background(), db, cfg)
	if err != nil {
		log.Fatalf("Report generation failed: %v", err)
	}

	fmt.Printf("✅ Report generated successfully!\n")
	fmt.Printf("   • Report ID:   %s\n", res.ReportID)
	fmt.Printf("   • Output:      %s/\n", res.LatestDir)
	fmt.Printf("   • Markdown:    %s/report.md\n", res.LatestDir)
	fmt.Printf("   • HTML:        %s/report.html\n", res.LatestDir)
	fmt.Printf("   • Manifest:    %s/manifest.json\n", res.LatestDir)
	fmt.Printf("   • Artifacts:   %d files generated\n", len(res.GeneratedFiles))
	if res.ReadmeUpdated {
		fmt.Printf("   • README.md:   Updated generated section between markers\n")
	}
}

func runReportVerify(args []string) {
	fs := flag.NewFlagSet("report-verify", flag.ExitOnError)
	reportDir := "reports/latest"
	readmePath := "README.md"

	fs.StringVar(&reportDir, "dir", "reports/latest", "Path to report directory to verify")
	fs.StringVar(&readmePath, "readme", "README.md", "Path to README.md to verify")
	_ = fs.Parse(args)

	fmt.Printf("[ExecLens Verify] Auditing report artifacts in '%s'...\n", reportDir)
	if err := reports.VerifyReport(reportDir, readmePath); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Report verification failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Report verification passed! All artifact checksums, JSON schemas, image links, and README markers are valid.")
}

