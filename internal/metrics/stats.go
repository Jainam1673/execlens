package metrics

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Summary represents the calculated statistical summary of benchmark observations.
type Summary struct {
	Count       int     `json:"count"`
	MinNs       int64   `json:"min_ns"`
	MaxNs       int64   `json:"max_ns"`
	MeanNs      float64 `json:"mean_ns"`
	MedianNs    float64 `json:"median_ns"`
	StdDevNs    float64 `json:"std_dev_ns"`
	IQRDuration float64 `json:"iqr_ns"`
	CVPercent   float64 `json:"cv_percent"` // Coefficient of Variation (StdDev / Mean * 100)

	// User-friendly millisecond conversions
	MinMs    float64 `json:"min_ms"`
	MaxMs    float64 `json:"max_ms"`
	MeanMs   float64 `json:"mean_ms"`
	MedianMs float64 `json:"median_ms"`
	StdDevMs float64 `json:"std_dev_ms"`

	// 95% Confidence Interval for the mean (Student's t distribution approximation)
	CI95LowerNs float64 `json:"ci95_lower_ns"`
	CI95UpperNs float64 `json:"ci95_upper_ns"`
	CI95LowerMs float64 `json:"ci95_lower_ms"`
	CI95UpperMs float64 `json:"ci95_upper_ms"`

	// Peak memory in bytes (if available, 0 if unavailable)
	PeakMemoryBytes int64   `json:"peak_memory_bytes"`
	PeakMemoryMB    float64 `json:"peak_memory_mb"`

	// Throughput
	ThroughputOpsPerSec float64 `json:"throughput_ops_per_sec,omitempty"`
	ThroughputMBPerSec  float64 `json:"throughput_mb_per_sec,omitempty"`
}

// CalculateSummary computes statistical metrics from a slice of duration observations and memory.
func CalculateSummary(durations []time.Duration, peakMemoryBytes int64, totalOps int64, totalBytesProcessed int64) (Summary, error) {
	n := len(durations)
	if n == 0 {
		return Summary{}, fmt.Errorf("cannot calculate summary from empty sample list")
	}

	rawNs := make([]float64, n)
	for i, d := range durations {
		rawNs[i] = float64(d.Nanoseconds())
	}
	sort.Float64s(rawNs)

	minVal := rawNs[0]
	maxVal := rawNs[n-1]

	var sum float64
	for _, v := range rawNs {
		sum += v
	}
	mean := sum / float64(n)

	// Median & Quartiles
	var median float64
	if n%2 == 1 {
		median = rawNs[n/2]
	} else {
		median = (rawNs[n/2-1] + rawNs[n/2]) / 2.0
	}

	q1 := percentile(rawNs, 25)
	q3 := percentile(rawNs, 75)
	iqr := q3 - q1

	// Sample Standard Deviation
	var variance float64
	if n > 1 {
		var sqDiffSum float64
		for _, v := range rawNs {
			diff := v - mean
			sqDiffSum += diff * diff
		}
		variance = sqDiffSum / float64(n-1)
	}
	stdDev := math.Sqrt(variance)

	var cv float64
	if mean > 0 {
		cv = (stdDev / mean) * 100.0
	}

	// 95% Confidence Interval using t-distribution critical value
	tCrit := tDistributionCritical95(n - 1)
	marginOfError := 0.0
	if n > 1 {
		marginOfError = tCrit * (stdDev / math.Sqrt(float64(n)))
	}
	ciLower := math.Max(0, mean-marginOfError)
	ciUpper := mean + marginOfError

	s := Summary{
		Count:               n,
		MinNs:               int64(minVal),
		MaxNs:               int64(maxVal),
		MeanNs:              mean,
		MedianNs:            median,
		StdDevNs:            stdDev,
		IQRDuration:         iqr,
		CVPercent:           cv,
		MinMs:               minVal / 1e6,
		MaxMs:               maxVal / 1e6,
		MeanMs:              mean / 1e6,
		MedianMs:            median / 1e6,
		StdDevMs:            stdDev / 1e6,
		CI95LowerNs:         ciLower,
		CI95UpperNs:         ciUpper,
		CI95LowerMs:         ciLower / 1e6,
		CI95UpperMs:         ciUpper / 1e6,
		PeakMemoryBytes:     peakMemoryBytes,
		PeakMemoryMB:        float64(peakMemoryBytes) / (1024 * 1024),
		ThroughputOpsPerSec: 0,
		ThroughputMBPerSec:  0,
	}

	// Throughput calculation using median execution duration in seconds
	medianSec := median / 1e9
	if medianSec > 0 {
		if totalOps > 0 {
			s.ThroughputOpsPerSec = float64(totalOps) / medianSec
		}
		if totalBytesProcessed > 0 {
			mb := float64(totalBytesProcessed) / (1024 * 1024)
			s.ThroughputMBPerSec = mb / medianSec
		}
	}

	return s, nil
}

// percentile computes percentile using linear interpolation between closest ranks.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 || p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[n-1]
	}

	rank := (p / 100.0) * float64(n-1)
	lowerIndex := int(math.Floor(rank))
	upperIndex := int(math.Ceil(rank))
	weight := rank - float64(lowerIndex)

	return sorted[lowerIndex]*(1.0-weight) + sorted[upperIndex]*weight
}

// tDistributionCritical95 returns two-tailed t critical value for alpha=0.05 (95% CI)
// for given degrees of freedom (df = n - 1).
func tDistributionCritical95(df int) float64 {
	switch {
	case df <= 0:
		return 0.0
	case df == 1:
		return 12.706
	case df == 2:
		return 4.303
	case df == 3:
		return 3.182
	case df == 4:
		return 2.776
	case df == 5:
		return 2.571
	case df == 6:
		return 2.447
	case df == 7:
		return 2.365
	case df == 8:
		return 2.306
	case df == 9:
		return 2.262
	case df == 10:
		return 2.228
	case df <= 15:
		return 2.131
	case df <= 20:
		return 2.086
	case df <= 30:
		return 2.042
	default:
		return 1.960 // Normal distribution approximation for large sample sizes
	}
}

// CalculateSpeedup computes speedup relative to a baseline.
// speedup = baseline_median / candidate_median
// A value > 1.0 means candidate is faster.
func CalculateSpeedup(baselineMedianNs, candidateMedianNs float64) float64 {
	if candidateMedianNs <= 0 || baselineMedianNs <= 0 {
		return 0.0
	}
	return baselineMedianNs / candidateMedianNs
}

// CalculateMemoryRatio computes relative memory usage.
// ratio = candidate_memory / baseline_memory
// A value < 1.0 means candidate used less memory.
func CalculateMemoryRatio(baselineMemoryBytes, candidateMemoryBytes int64) float64 {
	if baselineMemoryBytes <= 0 || candidateMemoryBytes <= 0 {
		return 0.0
	}
	return float64(candidateMemoryBytes) / float64(baselineMemoryBytes)
}

// SignificanceAssessment defines the qualitative and statistical assessment of a comparison.
type SignificanceAssessment string

const (
	AssessmentInconclusive     SignificanceAssessment = "INCONCLUSIVE"
	AssessmentNegligible       SignificanceAssessment = "NEGLIGIBLE_DIFFERENCE"
	AssessmentPractical        SignificanceAssessment = "PRACTICAL_DIFFERENCE"
	AssessmentStatisticallySig SignificanceAssessment = "STATISTICALLY_SIGNIFICANT"
)

// AssessDifference compares two summaries and determines if the observed difference
// is within measurement noise, exceeds a practical threshold (e.g. 5%), or is statistically significant.
func AssessDifference(baseline, candidate Summary, practicalThresholdPercent float64) (SignificanceAssessment, string) {
	if baseline.Count < 2 || candidate.Count < 2 {
		return AssessmentInconclusive, "Sample size too small (< 2 observations) to assess statistical significance."
	}

	speedup := CalculateSpeedup(baseline.MedianNs, candidate.MedianNs)
	percentDiff := math.Abs((candidate.MedianNs-baseline.MedianNs)/baseline.MedianNs) * 100.0

	// Check for overlap in 95% Confidence Intervals
	intervalsOverlap := (baseline.CI95LowerNs <= candidate.CI95UpperNs) && (candidate.CI95LowerNs <= baseline.CI95UpperNs)

	if intervalsOverlap {
		if percentDiff < practicalThresholdPercent {
			return AssessmentNegligible, fmt.Sprintf("Observed %.1f%% median difference is within 95%% confidence interval overlap and below the %.1f%% threshold.", percentDiff, practicalThresholdPercent)
		}
		return AssessmentInconclusive, fmt.Sprintf("Observed %.1f%% median difference exceeds threshold, but confidence intervals overlap due to sample variance (noise).", percentDiff)
	}

	// Intervals do not overlap
	if percentDiff < practicalThresholdPercent {
		return AssessmentNegligible, fmt.Sprintf("Statistically distinct distributions, but difference (%.1f%%) is below the practical significance threshold (%.1f%%).", percentDiff, practicalThresholdPercent)
	}

	var direction string
	if speedup > 1.0 {
		direction = fmt.Sprintf("%.2fx faster", speedup)
	} else {
		direction = fmt.Sprintf("%.2fx slower", 1.0/speedup)
	}

	return AssessmentStatisticallySig, fmt.Sprintf("Statistically significant difference with non-overlapping 95%% confidence intervals (%s, %.1f%% difference).", direction, percentDiff)
}
