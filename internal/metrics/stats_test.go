package metrics

import (
	"math"
	"testing"
	"time"
)

func TestCalculateSummary(t *testing.T) {
	// Known dataset: 10ms, 20ms, 30ms, 40ms, 50ms
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}

	summary, err := CalculateSummary(durations, 1024*1024*64, 1000, 1024*1024*10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Count != 5 {
		t.Errorf("expected count 5, got %d", summary.Count)
	}

	if summary.MinMs != 10.0 {
		t.Errorf("expected min 10ms, got %f", summary.MinMs)
	}

	if summary.MaxMs != 50.0 {
		t.Errorf("expected max 50ms, got %f", summary.MaxMs)
	}

	if summary.MedianMs != 30.0 {
		t.Errorf("expected median 30ms, got %f", summary.MedianMs)
	}

	if summary.MeanMs != 30.0 {
		t.Errorf("expected mean 30ms, got %f", summary.MeanMs)
	}

	// Sample variance = sum((x - 30)^2) / 4 = (400 + 100 + 0 + 100 + 400) / 4 = 1000 / 4 = 250
	// Sample std dev = sqrt(250) ≈ 15.8113883 ms
	expectedStdDevMs := math.Sqrt(250)
	if math.Abs(summary.StdDevMs-expectedStdDevMs) > 0.001 {
		t.Errorf("expected std dev ~%f, got %f", expectedStdDevMs, summary.StdDevMs)
	}

	if summary.PeakMemoryMB != 64.0 {
		t.Errorf("expected peak memory 64MB, got %f", summary.PeakMemoryMB)
	}

	// Throughput check (1000 ops in 30ms = 1000 / 0.03 = 33333.33 ops/sec)
	if summary.ThroughputOpsPerSec < 33300 || summary.ThroughputOpsPerSec > 33400 {
		t.Errorf("unexpected throughput ops/sec: %f", summary.ThroughputOpsPerSec)
	}
}

func TestCalculateSpeedup(t *testing.T) {
	baseline := 100.0 // 100ns
	candidate := 50.0 // 50ns
	speedup := CalculateSpeedup(baseline, candidate)
	if speedup != 2.0 {
		t.Errorf("expected 2.0x speedup, got %f", speedup)
	}

	slowerCandidate := 200.0
	slowerSpeedup := CalculateSpeedup(baseline, slowerCandidate)
	if slowerSpeedup != 0.5 {
		t.Errorf("expected 0.5x speedup, got %f", slowerSpeedup)
	}
}

func TestAssessDifference(t *testing.T) {
	// Baseline: 100ms ± 2ms
	bDurations := []time.Duration{
		98 * time.Millisecond,
		99 * time.Millisecond,
		100 * time.Millisecond,
		101 * time.Millisecond,
		102 * time.Millisecond,
	}
	baseline, _ := CalculateSummary(bDurations, 0, 0, 0)

	// Distinctly faster candidate: 50ms ± 2ms
	cDurations := []time.Duration{
		48 * time.Millisecond,
		49 * time.Millisecond,
		50 * time.Millisecond,
		51 * time.Millisecond,
		52 * time.Millisecond,
	}
	candidate, _ := CalculateSummary(cDurations, 0, 0, 0)

	assessment, note := AssessDifference(baseline, candidate, 5.0)
	if assessment != AssessmentStatisticallySig {
		t.Errorf("expected STATISTICALLY_SIGNIFICANT, got %s (note: %s)", assessment, note)
	}

	// Virtually identical candidate
	identicalCandidate, _ := CalculateSummary(bDurations, 0, 0, 0)
	assessment2, _ := AssessDifference(baseline, identicalCandidate, 5.0)
	if assessment2 != AssessmentNegligible {
		t.Errorf("expected NEGLIGIBLE_DIFFERENCE for identical data, got %s", assessment2)
	}
}
